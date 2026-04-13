// Package client implements a JSON-RPC client for communicating with sigd.
package client

import (
	"bufio"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/tenebris-tech/secmsg/global"
)

// DefaultAddr is the default address of the sigd daemon.
const DefaultAddr = "127.0.0.1:9801"

const defaultTimeout = 10 * time.Second

// Option configures a Client.
type Option func(*Client)

// WithTimeout sets the per-call RPC timeout applied when the caller's context
// has no deadline. Default: 10s.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.timeout = d
	}
}

// WithLogger injects a logger for outbound RPC calls, responses, and
// connection errors. Pass nil (or omit the option) for silent operation.
func WithLogger(l global.Logger) Option {
	return func(c *Client) {
		c.logger = l
	}
}

// Client holds an active connection to sigd and manages request/response
// multiplexing and notification dispatch.
type Client struct {
	addr   string
	conn   net.Conn
	reader *bufio.Reader

	timeout time.Duration
	logger  global.Logger

	// mu protects pending, nextID, and readErr — never held across I/O.
	mu      sync.Mutex
	pending map[uint64]chan *rpcResponse
	nextID  uint64
	readErr error // set by readLoop on unexpected connection error

	// writeMu serialises concurrent writes to conn without blocking reads or
	// request-tracking operations.
	writeMu sync.Mutex

	// subs is guarded by subsMu.
	subsMu sync.RWMutex
	subs   []*subscription

	closeOnce sync.Once
	doneOnce  sync.Once
	done      chan struct{}
	wg        sync.WaitGroup
}

// New creates a configured but unconnected Client. Use Dial to establish the
// connection, or use Dial directly as a one-shot constructor.
func New(addr string, opts ...Option) *Client {
	c := &Client{
		addr:    addr,
		pending: make(map[uint64]chan *rpcResponse),
		done:    make(chan struct{}),
		timeout: defaultTimeout,
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Dial connects to sigd at addr (e.g. "127.0.0.1:9801") and performs the
// hello handshake. The returned *Client is ready for use.
func Dial(addr string, opts ...Option) (*Client, error) {
	c := New(addr, opts...)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}
	c.conn = conn
	c.reader = bufio.NewReaderSize(conn, 256*1024)

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
		c.doneOnce.Do(func() { close(c.done) })
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
