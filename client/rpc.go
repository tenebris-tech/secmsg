package client

import (
	"encoding/json"
	"fmt"
)

// rpcRequest is a JSON-RPC 2.0 request object sent to the server.
type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      uint64          `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// rpcResponse is a JSON-RPC 2.0 response or notification received from the server.
type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *uint64         `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
	Method  string          `json:"method,omitempty"` // set for notifications
	Params  json.RawMessage `json:"params,omitempty"` // set for notifications
}

// rpcError is a JSON-RPC 2.0 error object.
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Error implements the error interface.
func (e *rpcError) Error() string {
	return fmt.Sprintf("rpc error %d: %s", e.Code, e.Message)
}

// call represents an in-flight RPC call.
type call struct {
	done chan struct{}
	resp json.RawMessage
	err  error
}

// respond delivers a result or error to the waiting caller.
func (c *call) respond(resp json.RawMessage, err error) {
	c.resp = resp
	c.err = err
	close(c.done)
}

// invoke sends a JSON-RPC request and waits for the response.
// It returns the raw result JSON on success.
func (c *Client) invoke(method string, params any) (json.RawMessage, error) {
	id := c.nextID.Add(1)

	var rawParams json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("marshal params: %w", err)
		}
		rawParams = b
	}

	req := rpcRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  rawParams,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	data = append(data, '\n')

	call := &call{done: make(chan struct{})}

	c.mu.Lock()
	c.pending[id] = call
	_, err = c.conn.Write(data)
	c.mu.Unlock()

	if err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("write request: %w", err)
	}

	select {
	case <-call.done:
		return call.resp, call.err
	case <-c.closeCh:
		return nil, fmt.Errorf("connection closed")
	}
}

// invokeNoResult sends a request and returns an error if the response contains
// a JSON-RPC error. The result payload is ignored.
func (c *Client) invokeNoResult(method string, params any) error {
	_, err := c.invoke(method, params)
	return err
}

// readLoop reads lines from the connection and dispatches responses to waiting
// callers or notifications to registered subscribers.
func (c *Client) readLoop() {
	defer c.Close() //nolint:errcheck

	for {
		line, err := c.reader.ReadBytes('\n')
		if err != nil {
			return
		}

		var resp rpcResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			continue
		}

		// Notification (no id field).
		if resp.ID == nil && resp.Method != "" {
			c.dispatchNotification(&resp)
			continue
		}

		// Response to a pending call.
		if resp.ID != nil {
			c.mu.Lock()
			call, ok := c.pending[*resp.ID]
			if ok {
				delete(c.pending, *resp.ID)
			}
			c.mu.Unlock()

			if ok {
				if resp.Error != nil {
					call.respond(nil, resp.Error)
				} else {
					call.respond(resp.Result, nil)
				}
			}
		}
	}
}
