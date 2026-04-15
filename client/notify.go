package client

import (
	"sync"

	"github.com/tenebris-tech/secmsg/schema"
)

// subscription tracks a single subscriber channel together with its
// cancellation state.  The done channel is closed by Unsubscribe; the ch
// channel is closed only after done is closed, ensuring dispatchNotification
// never sends on a closed channel.
type subscription struct {
	ch   chan *schema.Envelope
	done chan struct{}
	once sync.Once
}

// close signals the subscription is done and closes ch so readers see EOF.
// dispatchNotification guards all sends with the done channel, so no send can
// race with close(ch) after close(done) is observed.
func (s *subscription) close() {
	s.once.Do(func() {
		close(s.done)
		close(s.ch)
	})
}

// localSubscribe registers a local channel to receive dispatched notifications.
// It is used by Subscribe to set up the channel before sending the RPC so that
// notifications arriving immediately after the ack are not lost.
func (c *Client) localSubscribe() (<-chan *schema.Envelope, func()) {
	sub := &subscription{
		ch:   make(chan *schema.Envelope, 16),
		done: make(chan struct{}),
	}

	c.subsMu.Lock()
	c.subs = append(c.subs, sub)
	c.subsMu.Unlock()

	cancel := func() {
		sub.close()

		c.subsMu.Lock()
		for i, s := range c.subs {
			if s == sub {
				c.subs = append(c.subs[:i], c.subs[i+1:]...)
				break
			}
		}
		c.subsMu.Unlock()
	}

	return sub.ch, cancel
}

// dispatchNotification fans out a notification to all live subscriptions.
// It uses a select with the subscription's done channel to guarantee that a
// send is never attempted after the channel has been closed.
func (c *Client) dispatchNotification(notif *rpcNotification) {
	env := &schema.Envelope{
		JSONRPC: notif.JSONRPC,
		Method:  notif.Method,
		Params:  notif.Params,
	}

	c.subsMu.RLock()
	subs := make([]*subscription, len(c.subs))
	copy(subs, c.subs)
	c.subsMu.RUnlock()

	for _, s := range subs {
		// Use a non-blocking send guarded by the done channel so we never
		// send on a closed channel and never block the read loop.
		select {
		case <-s.done:
			// Subscription has been cancelled; skip it.
		case s.ch <- env:
			// Delivered.
		default:
			// Consumer is too slow; drop this notification rather than
			// blocking the read loop.
		}
	}
}
