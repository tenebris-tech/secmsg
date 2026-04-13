package client

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// newTestServer creates a in-process TCP listener that acts as a minimal sigd
// stub.  It sends a hello notification, then for each request it calls handler
// which returns the raw JSON to send back as the response body.  The returned
// addr is ready to Dial.
func newTestServer(t *testing.T, handler func(method string, id uint64) json.RawMessage) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	t.Cleanup(func() { ln.Close() })

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// Send hello.
		fmt.Fprintf(conn, `{"jsonrpc":"2.0","method":"hello"}`+"\n")

		reader := bufio.NewReader(conn)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			var req rpcRequest
			if err := json.Unmarshal([]byte(strings.TrimRight(line, "\n")), &req); err != nil {
				continue
			}
			result := handler(req.Method, req.ID)
			resp, _ := json.Marshal(rpcResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  result,
			})
			fmt.Fprintf(conn, "%s\n", resp)
		}
	}()

	return ln.Addr().String()
}

// TestCallBasic verifies a round-trip RPC call.
func TestCallBasic(t *testing.T) {
	addr := newTestServer(t, func(method string, id uint64) json.RawMessage {
		if method != "send" {
			t.Errorf("unexpected method %q", method)
		}
		return json.RawMessage(`"ok"`)
	})

	c, err := Dial(addr)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer c.Close()

	if err := c.SendMessage(context.Background(), "signal", "myaccount", "+15550001111", "hello"); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
}

// TestConcurrentCalls verifies that multiple goroutines can call concurrently
// without data races or deadlocks.
func TestConcurrentCalls(t *testing.T) {
	addr := newTestServer(t, func(method string, id uint64) json.RawMessage {
		return json.RawMessage(`"ok"`)
	})

	c, err := Dial(addr)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer c.Close()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := c.SendMessage(context.Background(), "signal", "acc", "+1", "msg"); err != nil {
				t.Errorf("SendMessage: %v", err)
			}
		}()
	}
	wg.Wait()
}

// TestRPCError verifies that a server-side error is surfaced to the caller.
func TestRPCError(t *testing.T) {
	// We need a server that sends an error response; use a bespoke listener.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		fmt.Fprintf(conn, `{"jsonrpc":"2.0","method":"hello"}`+"\n")
		reader := bufio.NewReader(conn)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			var req rpcRequest
			if err := json.Unmarshal([]byte(strings.TrimRight(line, "\n")), &req); err != nil {
				continue
			}
			resp, _ := json.Marshal(rpcResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &rpcError{Code: -32600, Message: "invalid request"},
			})
			fmt.Fprintf(conn, "%s\n", resp)
		}
	}()

	c, err := Dial(ln.Addr().String())
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer c.Close()

	err = c.SendMessage(context.Background(), "signal", "acc", "+1", "msg")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// TestSubscribeAndDispatch verifies that notifications reach subscribers.
func TestSubscribeAndDispatch(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	notifSent := make(chan struct{})
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		fmt.Fprintf(conn, `{"jsonrpc":"2.0","method":"hello"}`+"\n")
		// Give the client a moment to subscribe, then send a notification.
		<-notifSent
		fmt.Fprintf(conn, `{"jsonrpc":"2.0","method":"message","params":{"body":"hi"}}`+"\n")
		// Keep connection open.
		time.Sleep(time.Second)
	}()

	c, err := Dial(ln.Addr().String())
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer c.Close()

	ch, cancel := c.Subscribe()
	defer cancel()

	close(notifSent)

	select {
	case env := <-ch:
		if env.Method != "message" {
			t.Errorf("expected method=message, got %q", env.Method)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for notification")
	}
}

// TestSubscribeCancelNoPanic verifies that cancelling a subscription after the
// client dispatches a notification does not panic (send on closed channel).
func TestSubscribeCancelNoPanic(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		fmt.Fprintf(conn, `{"jsonrpc":"2.0","method":"hello"}`+"\n")
		// Blast many notifications.
		for i := 0; i < 100; i++ {
			fmt.Fprintf(conn, `{"jsonrpc":"2.0","method":"message","params":{}}`+"\n")
		}
		time.Sleep(500 * time.Millisecond)
	}()

	c, err := Dial(ln.Addr().String())
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer c.Close()

	_, cancel := c.Subscribe()
	// Cancel immediately — dispatch should not panic.
	cancel()

	// Give the read loop time to process all notifications against the now-
	// cancelled subscription.
	time.Sleep(200 * time.Millisecond)
}
