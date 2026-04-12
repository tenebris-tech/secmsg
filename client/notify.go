package client

import (
	"encoding/json"

	"github.com/securityguy/secmsg/schema"
)

// subscriber holds a channel receiving notifications for one Subscribe call.
type subscriber struct {
	ch chan *schema.Envelope
}

// addSubscriber registers a subscriber.
func (c *Client) addSubscriber(sub *subscriber) {
	c.subMu.Lock()
	c.subscribers = append(c.subscribers, sub)
	c.subMu.Unlock()
}

// removeSubscriber removes a subscriber and drains its channel.
func (c *Client) removeSubscriber(sub *subscriber) {
	c.subMu.Lock()
	for i, s := range c.subscribers {
		if s == sub {
			c.subscribers = append(c.subscribers[:i], c.subscribers[i+1:]...)
			break
		}
	}
	c.subMu.Unlock()

	// Drain any buffered notifications so the goroutine does not leak.
	for {
		select {
		case <-sub.ch:
		default:
			return
		}
	}
}

// dispatchNotification delivers an incoming notification to all registered subscribers.
func (c *Client) dispatchNotification(resp *rpcResponse) {
	env := &schema.Envelope{
		JSONRPC: resp.JSONRPC,
		Method:  resp.Method,
		Params:  json.RawMessage(resp.Params),
	}

	c.subMu.RLock()
	subs := make([]*subscriber, len(c.subscribers))
	copy(subs, c.subscribers)
	c.subMu.RUnlock()

	for _, sub := range subs {
		select {
		case sub.ch <- env:
		default:
			// Drop the notification if the subscriber is not keeping up.
		}
	}
}
