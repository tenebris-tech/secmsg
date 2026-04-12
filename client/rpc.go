package client

import (
	"encoding/json"
	"fmt"
	"io"
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

// hello reads and validates the server greeting.
func (c *Client) hello() error {
	if !c.scanner.Scan() {
		err := c.scanner.Err()
		if err == nil {
			err = io.EOF
		}
		return fmt.Errorf("hello: %w", err)
	}
	line := c.scanner.Bytes()

	var greeting struct {
		JSONRPC string `json:"jsonrpc"`
		Method  string `json:"method"`
	}
	if err := json.Unmarshal(line, &greeting); err != nil {
		return fmt.Errorf("hello: malformed greeting: %w", err)
	}
	if greeting.Method != "hello" {
		return fmt.Errorf("hello: unexpected greeting method %q", greeting.Method)
	}
	return nil
}

// call sends a JSON-RPC request and waits for the corresponding response.
// mu is held only while registering/deregistering the pending channel; writes
// to the connection use a separate writeMu so they never block readers.
func (c *Client) call(method string, params any, result any) error {
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

	// Serialise writes without holding mu.
	c.writeMu.Lock()
	_, err = c.conn.Write(data)
	c.writeMu.Unlock()
	if err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return fmt.Errorf("write request: %w", err)
	}

	// Wait for the response or client shutdown.
	resp, ok := <-ch
	if !ok {
		return fmt.Errorf("client closed before response for id %d", id)
	}
	if resp.Error != nil {
		return resp.Error
	}
	if result != nil && resp.Result != nil {
		if err := json.Unmarshal(resp.Result, result); err != nil {
			return fmt.Errorf("unmarshal result: %w", err)
		}
	}
	return nil
}

// readLoop reads newline-delimited JSON from the connection and dispatches
// each message to either a pending RPC channel or to notification subscribers.
// It runs in its own goroutine and exits when the connection is closed.
func (c *Client) readLoop() {
	defer c.wg.Done()

	for c.scanner.Scan() {
		line := c.scanner.Bytes()

		// Peek at the ID field to decide whether this is a response or a notification.
		var peek struct {
			ID *uint64 `json:"id"`
		}
		if err := json.Unmarshal(line, &peek); err != nil {
			continue
		}

		if peek.ID != nil {
			// Response to a pending call.
			var resp rpcResponse
			if err := json.Unmarshal(line, &resp); err != nil {
				continue
			}
			c.mu.Lock()
			ch, ok := c.pending[resp.ID]
			if ok {
				delete(c.pending, resp.ID)
			}
			c.mu.Unlock()
			if ok {
				ch <- &resp
			}
		} else {
			// Server-push notification.
			var notif rpcNotification
			if err := json.Unmarshal(line, &notif); err != nil {
				continue
			}
			c.dispatchNotification(&notif)
		}
	}
}
