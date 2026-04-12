// Package client provides a typed Go client for the sigd JSON-RPC 2.0 API.
// It handles connecting to the daemon and provides one method per RPC call.
package client

import (
	"bufio"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
)

// Client is a connection to a sigd daemon.
// It is safe for concurrent use after construction.
type Client struct {
	conn   net.Conn
	reader *bufio.Reader

	// mu guards writes to conn and the pending map.
	mu      sync.Mutex
	pending map[uint64]*call

	// nextID is an atomic counter for generating unique request IDs.
	nextID atomic.Uint64

	// closeCh is closed when Close is called.
	closeCh   chan struct{}
	closeOnce sync.Once

	// subMu guards the subscribers slice.
	subMu       sync.RWMutex
	subscribers []*subscriber
}

// Dial connects to the sigd daemon at addr and returns a ready-to-use Client.
// The caller is responsible for calling Close when done.
func Dial(addr string) (*Client, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}

	c := &Client{
		conn:    conn,
		reader:  bufio.NewReader(conn),
		pending: make(map[uint64]*call),
		closeCh: make(chan struct{}),
	}

	// Start the read loop that dispatches responses and notifications.
	go c.readLoop()

	return c, nil
}

// Close closes the connection to the daemon.
func (c *Client) Close() error {
	var err error
	c.closeOnce.Do(func() {
		close(c.closeCh)
		err = c.conn.Close()

		// Fail all pending calls.
		c.mu.Lock()
		for _, call := range c.pending {
			call.respond(nil, fmt.Errorf("connection closed"))
		}
		c.pending = make(map[uint64]*call)
		c.mu.Unlock()
	})
	return err
}
