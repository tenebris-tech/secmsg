// Package client implements a JSON-RPC client for communicating with sigd.
package client

import (
	"bufio"
	"fmt"
	"net"
	"sync"
)

// DefaultAddr is the default address of the sigd daemon.
const DefaultAddr = "127.0.0.1:9801"

// Client holds an active connection to sigd and manages request/response
// multiplexing and notification dispatch.
type Client struct {
	conn   net.Conn
	reader *bufio.Reader

	// mu protects pending and nextID only — never held across I/O.
	mu      sync.Mutex
	pending map[uint64]chan *rpcResponse
	nextID  uint64

	// writeMu serialises concurrent writes to conn without blocking reads or
	// request-tracking operations.
	writeMu sync.Mutex

	// subs is guarded by subsMu.
	subsMu sync.RWMutex
	subs   []*subscription

	closeOnce sync.Once
	done      chan struct{}
	wg        sync.WaitGroup
}

// Dial connects to sigd at addr (e.g. "localhost:7777") and performs the
// hello handshake.  The returned *Client is ready for use.
func Dial(addr string) (*Client, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}

	c := &Client{
		conn:    conn,
		reader:  bufio.NewReader(conn),
		pending: make(map[uint64]chan *rpcResponse),
		done:    make(chan struct{}),
	}

	if err := c.hello(); err != nil {
		conn.Close()
		return nil, err
	}

	c.wg.Add(1)
	go c.readLoop()

	return c, nil
}

// Close shuts down the client, draining all pending requests and subscriptions.
func (c *Client) Close() error {
	var err error
	c.closeOnce.Do(func() {
		close(c.done)
		err = c.conn.Close()

		// Wake any goroutine blocked in readLoop.
		c.wg.Wait()

		// Drain pending RPCs.
		c.mu.Lock()
		for id, ch := range c.pending {
			close(ch)
			delete(c.pending, id)
		}
		c.mu.Unlock()

		// Close all subscription channels.
		c.subsMu.Lock()
		for _, s := range c.subs {
			s.close()
		}
		c.subs = nil
		c.subsMu.Unlock()
	})
	return err
}
