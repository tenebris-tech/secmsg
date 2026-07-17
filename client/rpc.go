package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/tenebris-tech/secmsg/schema"
)

// rpcRequest is the wire format for a JSON-RPC 2.0 request.
type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      uint64 `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// rpcResponse is the wire format for a JSON-RPC 2.0 response.
type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      uint64          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

// rpcNotification is the wire format for a JSON-RPC 2.0 notification (no ID).
type rpcNotification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// rpcError carries a JSON-RPC 2.0 error object.
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *rpcError) Error() string {
	return fmt.Sprintf("rpc error %d: %s", e.Code, e.Message)
}

// RPCErrorCode reports the JSON-RPC error code carried by err when err (or any
// error it wraps) originated as a daemon RPC error. ok is false for errors that
// are not daemon RPC errors (transport failures, context cancellation, …). This
// lets callers branch on protocol error codes (e.g. schema.ErrCodeStealth)
// without matching on Error() strings.
func RPCErrorCode(err error) (code int, ok bool) {
	var re *rpcError
	if errors.As(err, &re) {
		return re.Code, true
	}
	return 0, false
}

// hello reads and validates the server greeting.
func (c *Client) hello() error {
	line, err := c.reader.ReadString('\n')
	if err != nil {
		if err == io.EOF && len(line) == 0 {
			return fmt.Errorf("hello: %w", io.EOF)
		}
		if err != io.EOF {
			return fmt.Errorf("hello: %w", err)
		}
		// err == io.EOF && len(line) > 0: the server sent the greeting without
		// a trailing newline. Accept the partial line as a valid greeting.
	}
	line = strings.TrimRight(line, "\n")

	var greeting struct {
		JSONRPC string            `json:"jsonrpc"`
		Method  string            `json:"method"`
		Params  schema.InfoParams `json:"params"`
	}
	if err := json.Unmarshal([]byte(line), &greeting); err != nil {
		return fmt.Errorf("hello: malformed greeting: %w", err)
	}
	if greeting.Method != schema.MethodHello {
		return fmt.Errorf("hello: unexpected greeting method %q", greeting.Method)
	}
	c.info = greeting.Params
	return nil
}

// call sends a JSON-RPC request and waits for the corresponding response.
// mu is held only while registering/deregistering the pending channel; writes
// to the connection use a separate writeMu so they never block readers.
func (c *Client) call(ctx context.Context, method string, params any, result any) error {
	// Apply the configured timeout when the caller's context has no deadline.
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	c.mu.Lock()
	c.nextID++
	id := c.nextID
	ch := make(chan *rpcResponse, 1)
	c.pending[id] = ch
	c.mu.Unlock()

	req := rpcRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}
	data, err := json.Marshal(req)
	if err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return fmt.Errorf("marshal request: %w", err)
	}
	data = append(data, '\n')

	if c.logger != nil {
		c.logger.Debugf("rpc send method=%s", method)
	}

	// Serialise writes without holding mu. Apply context deadline if available,
	// then clear it explicitly before releasing writeMu so that a subsequent
	// goroutine cannot observe a stale deadline.
	c.writeMu.Lock()
	if deadline, ok := ctx.Deadline(); ok {
		c.conn.SetWriteDeadline(deadline)
	}
	_, err = c.conn.Write(data)
	c.conn.SetWriteDeadline(time.Time{})
	c.writeMu.Unlock()
	if err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return fmt.Errorf("write request: %w", err)
	}

	// Wait for the response, context cancellation, or client shutdown.
	select {
	case resp, ok := <-ch:
		if !ok {
			return fmt.Errorf("client closed before response for id %d", id)
		}
		if resp.Error != nil {
			if c.logger != nil {
				c.logger.Warningf("rpc error method=%s err=%v", method, resp.Error)
			}
			return resp.Error
		}
		if c.logger != nil {
			c.logger.Debugf("rpc recv method=%s", method)
		}
		if result != nil && resp.Result != nil {
			if err := json.Unmarshal(resp.Result, result); err != nil {
				return fmt.Errorf("unmarshal result: %w", err)
			}
		}
		return nil
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return ctx.Err()
	}
}

// readLoop reads newline-delimited JSON from the connection and dispatches
// each message to either a pending RPC channel or to notification subscribers.
// It runs in its own goroutine and exits when the connection is closed.
func (c *Client) readLoop() {
	defer c.wg.Done()

	// S3: On exit (any cause), signal disconnection and drain pending RPCs so
	// callers return an error instead of hanging until context timeout.
	// mu is released before sending to channels to avoid deadlock with callers
	// that hold ctx cancellation and are trying to acquire mu simultaneously.
	defer func() {
		c.doneOnce.Do(func() { close(c.done) })

		c.mu.Lock()
		pending := make(map[uint64]chan *rpcResponse, len(c.pending))
		for id, ch := range c.pending {
			pending[id] = ch
		}
		c.pending = make(map[uint64]chan *rpcResponse)
		c.mu.Unlock()

		connErr := &rpcError{Code: -32000, Message: "connection closed"}
		for _, ch := range pending {
			select {
			case ch <- &rpcResponse{Error: connErr}:
			default:
			}
		}

		// Close subscription channels so subscribers blocked on receive observe
		// the disconnect (channel close) and can reconnect, instead of hanging
		// until the client is explicitly closed. close() is idempotent, so a later
		// Close()/Unsubscribe on the same subscription is safe.
		c.subsMu.Lock()
		subs := c.subs
		c.subs = nil
		c.subsMu.Unlock()
		for _, s := range subs {
			s.close()
		}
	}()

	for {
		line, err := c.reader.ReadString('\n')

		// Process whatever was read before checking the error.
		if line != "" {
			line = strings.TrimRight(line, "\n")

			// Peek at the ID field to decide whether this is a response or a notification.
			var peek struct {
				ID *uint64 `json:"id"`
			}
			if jsonErr := json.Unmarshal([]byte(line), &peek); jsonErr == nil {
				if peek.ID != nil {
					// Response to a pending call.
					var resp rpcResponse
					if jsonErr := json.Unmarshal([]byte(line), &resp); jsonErr != nil {
						if c.logger != nil {
							c.logger.Warningf("readLoop: failed to unmarshal response err=%v", jsonErr)
						}
					} else {
						c.mu.Lock()
						ch, ok := c.pending[resp.ID]
						if ok {
							delete(c.pending, resp.ID)
						}
						c.mu.Unlock()
						if ok {
							ch <- &resp
						}
					}
				} else {
					// Server-push notification.
					var notif rpcNotification
					if jsonErr := json.Unmarshal([]byte(line), &notif); jsonErr != nil {
						if c.logger != nil {
							c.logger.Warningf("readLoop: failed to unmarshal notification err=%v", jsonErr)
						}
					} else {
						c.dispatchNotification(&notif)
					}
				}
			} else if c.logger != nil {
				c.logger.Warningf("malformed JSON from daemon: read %d bytes, parse error: %v", len(line), jsonErr)
			}
		}

		if err != nil {
			// On io.EOF check if client was intentionally closed.
			if err == io.EOF {
				break
			}
			// For any other error, check if it's due to an intentional close.
			select {
			case <-c.done:
			default:
				c.mu.Lock()
				c.readErr = err
				c.mu.Unlock()
				if c.logger != nil {
					c.logger.Errorf("connection error err=%v", err)
				}
			}
			break
		}
	}

}
