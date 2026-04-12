package client

import (
	"encoding/json"
	"fmt"

	"github.com/securityguy/secmsg/schema"
)

// ---------------------------------------------------------------------------
// Request / result types matching sigd's internal RPC types
// ---------------------------------------------------------------------------

// statusParams are the optional params for the "status" method.
type statusParams struct {
	Account string `json:"account,omitempty"`
}

// StatusResult is the result of a Status call.
type StatusResult struct {
	Linked    bool   `json:"linked"`
	Connected bool   `json:"connected"`
	Account   string `json:"account,omitempty"`
	ACI       string `json:"aci,omitempty"`
	Phone     string `json:"phone,omitempty"`
}

// sendParams are the params for the "send" method.
type sendParams struct {
	Account string `json:"account"`
	To      string `json:"to"`
	Body    string `json:"body"`
}

// sendGroupParams are the params for the "send.group" method.
type sendGroupParams struct {
	GroupID string `json:"groupId"`
	Body    string `json:"body"`
}

// sendResult is the result for both send methods.
type sendResult struct {
	Timestamp uint64 `json:"timestamp"`
}

// receiveParams are the params for the "receive" method.
type receiveParams struct {
	Timeout int `json:"timeout"`
}

// MessageItem is a single message in a Receive result.
type MessageItem struct {
	From      string `json:"from"`
	Body      string `json:"body,omitempty"`
	Timestamp uint64 `json:"timestamp"`
	Type      string `json:"type"`
}

// receiveResult is the result for the "receive" method.
type receiveResult struct {
	Messages []MessageItem `json:"messages"`
}

// ContactItem is a single contact in a ContactsList result.
type ContactItem struct {
	ACI        string `json:"aci"`
	Phone      string `json:"phone,omitempty"`
	Name       string `json:"name,omitempty"`
	About      string `json:"about,omitempty"`
	AboutEmoji string `json:"about_emoji,omitempty"`
	Avatar     string `json:"avatar,omitempty"`
}

// contactsListResult is the result for the "contacts.list" method.
type contactsListResult struct {
	Contacts []ContactItem `json:"contacts"`
}

// GroupItem is a single group in a GroupsList result.
type GroupItem struct {
	ID      string `json:"id"`
	Members int    `json:"memberCount"`
}

// groupsListResult is the result for the "groups.list" method.
type groupsListResult struct {
	Groups []GroupItem `json:"groups"`
}

// receiptReadParams are the params for the "receipt.read" method.
type receiptReadParams struct {
	To         string   `json:"to"`
	Timestamps []uint64 `json:"timestamps"`
}

// typingParams are the params for the "typing" method.
type typingParams struct {
	Account string `json:"account"`
	To      string `json:"to"`
	Typing  bool   `json:"typing"`
}

// subscribeResult is the acknowledgement result for the "subscribe" method.
type subscribeResult struct {
	Subscribed bool `json:"subscribed"`
}

// ---------------------------------------------------------------------------
// Methods
// ---------------------------------------------------------------------------

// Link initiates device linking. It sends a link.request and returns the
// QR URI the caller should display. Poll LinkStatus to detect completion.
func (c *Client) Link(name, account string) (*schema.LinkReply, error) {
	params := schema.LinkRequestParams{
		Name:    name,
		Account: account,
	}
	raw, err := c.invoke(schema.MethodLinkRequest, params)
	if err != nil {
		return nil, fmt.Errorf("link.request: %w", err)
	}
	var reply schema.LinkReply
	if err := json.Unmarshal(raw, &reply); err != nil {
		return nil, fmt.Errorf("link.request: decode response: %w", err)
	}
	return &reply, nil
}

// LinkStatus polls for the result of a pending Link call.
func (c *Client) LinkStatus() (*schema.LinkReply, error) {
	raw, err := c.invoke(schema.MethodLinkStatus, nil)
	if err != nil {
		return nil, fmt.Errorf("link.status: %w", err)
	}
	var reply schema.LinkReply
	if err := json.Unmarshal(raw, &reply); err != nil {
		return nil, fmt.Errorf("link.status: decode response: %w", err)
	}
	return &reply, nil
}

// Status returns the current linked/connected state of the daemon.
// Pass an empty string for account to query the default account.
func (c *Client) Status(account string) (*StatusResult, error) {
	var params *statusParams
	if account != "" {
		params = &statusParams{Account: account}
	}
	raw, err := c.invoke("status", params)
	if err != nil {
		return nil, fmt.Errorf("status: %w", err)
	}
	var result StatusResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("status: decode response: %w", err)
	}
	return &result, nil
}

// Send sends a 1:1 text message. to is an ACI UUID or E.164 phone number.
// It returns the message timestamp on success.
func (c *Client) Send(to, body, account string) (uint64, error) {
	params := sendParams{
		Account: account,
		To:      to,
		Body:    body,
	}
	raw, err := c.invoke("send", params)
	if err != nil {
		return 0, fmt.Errorf("send: %w", err)
	}
	var result sendResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return 0, fmt.Errorf("send: decode response: %w", err)
	}
	return result.Timestamp, nil
}

// SendGroup sends a message to a Signal group identified by groupId (base64).
// It returns the message timestamp on success.
func (c *Client) SendGroup(groupID, body string) (uint64, error) {
	params := sendGroupParams{
		GroupID: groupID,
		Body:    body,
	}
	raw, err := c.invoke("send.group", params)
	if err != nil {
		return 0, fmt.Errorf("send.group: %w", err)
	}
	var result sendResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return 0, fmt.Errorf("send.group: decode response: %w", err)
	}
	return result.Timestamp, nil
}

// Receive polls for queued messages. timeout is in seconds (1-300).
// Returns an empty slice when no messages are available before the timeout.
func (c *Client) Receive(timeout int) ([]MessageItem, error) {
	params := receiveParams{Timeout: timeout}
	raw, err := c.invoke("receive", params)
	if err != nil {
		return nil, fmt.Errorf("receive: %w", err)
	}
	var result receiveResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("receive: decode response: %w", err)
	}
	return result.Messages, nil
}

// Subscribe registers this connection for push notifications and returns a
// channel that receives incoming Envelope values. The channel is closed when
// Close is called or the connection is lost.
func (c *Client) Subscribe() (<-chan *schema.Envelope, error) {
	raw, err := c.invoke("subscribe", nil)
	if err != nil {
		return nil, fmt.Errorf("subscribe: %w", err)
	}
	var ack subscribeResult
	if err := json.Unmarshal(raw, &ack); err != nil {
		return nil, fmt.Errorf("subscribe: decode ack: %w", err)
	}
	if !ack.Subscribed {
		return nil, fmt.Errorf("subscribe: server did not acknowledge subscription")
	}

	sub := &subscriber{ch: make(chan *schema.Envelope, 64)}
	c.addSubscriber(sub)

	// Remove the subscriber when the connection closes.
	go func() {
		<-c.closeCh
		c.removeSubscriber(sub)
		close(sub.ch)
	}()

	return sub.ch, nil
}

// ContactsList returns all contacts known to the daemon.
func (c *Client) ContactsList() ([]ContactItem, error) {
	raw, err := c.invoke("contacts.list", nil)
	if err != nil {
		return nil, fmt.Errorf("contacts.list: %w", err)
	}
	var result contactsListResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("contacts.list: decode response: %w", err)
	}
	return result.Contacts, nil
}

// GroupsList returns all groups known to the daemon.
func (c *Client) GroupsList() ([]GroupItem, error) {
	raw, err := c.invoke("groups.list", nil)
	if err != nil {
		return nil, fmt.Errorf("groups.list: %w", err)
	}
	var result groupsListResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("groups.list: decode response: %w", err)
	}
	return result.Groups, nil
}

// ReceiptRead sends read receipts for the given message timestamps to the
// specified recipient ACI.
func (c *Client) ReceiptRead(to string, timestamps []uint64) error {
	params := receiptReadParams{
		To:         to,
		Timestamps: timestamps,
	}
	if err := c.invokeNoResult("receipt.read", params); err != nil {
		return fmt.Errorf("receipt.read: %w", err)
	}
	return nil
}

// Typing sends a typing started or stopped indicator to a recipient.
func (c *Client) Typing(to string, typing bool, account string) error {
	params := typingParams{
		Account: account,
		To:      to,
		Typing:  typing,
	}
	if err := c.invokeNoResult("typing", params); err != nil {
		return fmt.Errorf("typing: %w", err)
	}
	return nil
}
