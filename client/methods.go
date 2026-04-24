package client

import (
	"context"

	"github.com/tenebris-tech/secmsg/schema"
)

// subscribeParams is the wire payload for the subscribe RPC request.
type subscribeParams struct {
	Accounts []string `json:"accounts,omitempty"`
}

// subscribeResult is the wire response for the subscribe RPC request.
type subscribeResult struct {
	Subscribed bool     `json:"subscribed"`
	Accounts   []string `json:"accounts,omitempty"`
}

// sendMessageParams is the wire payload for sending a 1:1 message.
type sendMessageParams struct {
	Service string `json:"service"`
	Account string `json:"account"`
	To      string `json:"to"`
	Body    string `json:"body"`
}

// sendGroupMessageParams is the wire payload for sending a group message.
type sendGroupMessageParams struct {
	Service string `json:"service"`
	Account string `json:"account"`
	GroupID string `json:"group_id"`
	Body    string `json:"body"`
}

// receiptReadParams is the wire payload for sending read receipts.
type receiptReadParams struct {
	Service    string   `json:"service"`
	Account    string   `json:"account"`
	To         string   `json:"to"`
	Timestamps []uint64 `json:"timestamps"`
}

// typingParams is the wire payload for sending a typing indicator.
//
// Wire format asymmetry: the outbound request uses {typing: bool}, while the
// inbound schema.TypingParams notification uses {action: "started"/"stopped"}.
// This is intentional — sigd translates the bool to an action string before
// broadcasting the notification to subscribers.
type typingParams struct {
	Service string `json:"service"`
	Account string `json:"account"`
	To      string `json:"to"`
	Typing  bool   `json:"typing"`
}

// statusParams is the wire payload for the status request.
type statusParams struct {
	Account string `json:"account,omitempty"`
}

// unlinkParams is the wire payload for the unlink request.
type unlinkParams struct {
	Account string `json:"account"`
}

// linkStatusParams is the wire payload for the link.status request.
type linkStatusParams struct {
	Account string `json:"account"`
}

// contactsParams is the wire payload for the contacts.list request.
type contactsParams struct {
	Service string `json:"service"`
	Account string `json:"account"`
}

// groupsParams is the wire payload for the groups.list request.
type groupsParams struct {
	Service string `json:"service"`
	Account string `json:"account"`
}

// SendMessage sends a text message to a 1:1 recipient.
func (c *Client) SendMessage(ctx context.Context, service, account, to, body string) error {
	params := sendMessageParams{
		Service: service,
		Account: account,
		To:      to,
		Body:    body,
	}
	return c.call(ctx, schema.MethodSend, params, nil)
}

// SendGroupMessage sends a text message to a group.
func (c *Client) SendGroupMessage(ctx context.Context, service, account, groupID, body string) error {
	params := sendGroupMessageParams{
		Service: service,
		Account: account,
		GroupID: groupID,
		Body:    body,
	}
	return c.call(ctx, schema.MethodSendGroup, params, nil)
}

// LinkRequest initiates device linking and returns the current link state.
func (c *Client) LinkRequest(ctx context.Context, account, name string) (*schema.LinkReply, error) {
	params := schema.LinkRequestParams{
		Account: account,
		Name:    name,
	}
	var reply schema.LinkReply
	if err := c.call(ctx, schema.MethodLinkRequest, params, &reply); err != nil {
		return nil, err
	}
	return &reply, nil
}

// LinkStatus returns the current link state without starting a new link flow.
func (c *Client) LinkStatus(ctx context.Context, account string) (*schema.LinkReply, error) {
	params := linkStatusParams{Account: account}
	var reply schema.LinkReply
	if err := c.call(ctx, schema.MethodLinkStatus, params, &reply); err != nil {
		return nil, err
	}
	return &reply, nil
}

// Contacts returns the contact list for the given account.
func (c *Client) Contacts(ctx context.Context, service, account string) ([]schema.Party, error) {
	params := contactsParams{Service: service, Account: account}
	var result []schema.Party
	if err := c.call(ctx, schema.MethodContactsList, params, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Groups returns the group list for the given account.
func (c *Client) Groups(ctx context.Context, service, account string) ([]schema.Party, error) {
	params := groupsParams{Service: service, Account: account}
	var result []schema.Party
	if err := c.call(ctx, schema.MethodGroupsList, params, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// SendReceiptRead sends read receipts for one or more message timestamps.
func (c *Client) SendReceiptRead(ctx context.Context, service, account, to string, timestamps []uint64) error {
	params := receiptReadParams{
		Service:    service,
		Account:    account,
		To:         to,
		Timestamps: timestamps,
	}
	return c.call(ctx, schema.MethodReceiptRead, params, nil)
}

// SendTyping sends a typing started or stopped indicator.
func (c *Client) SendTyping(ctx context.Context, service, account, to string, typing bool) error {
	params := typingParams{
		Service: service,
		Account: account,
		To:      to,
		Typing:  typing,
	}
	return c.call(ctx, schema.MethodTyping, params, nil)
}

// Status returns the linked/connected state for a single account.
func (c *Client) Status(ctx context.Context, account string) (*schema.StatusReply, error) {
	params := statusParams{Account: account}
	var reply schema.StatusReply
	if err := c.call(ctx, schema.MethodStatus, params, &reply); err != nil {
		return nil, err
	}
	return &reply, nil
}

// StatusAll returns the linked/connected state for all accounts.
func (c *Client) StatusAll(ctx context.Context) (*schema.StatusAllReply, error) {
	params := statusParams{}
	var reply schema.StatusAllReply
	if err := c.call(ctx, schema.MethodStatus, params, &reply); err != nil {
		return nil, err
	}
	return &reply, nil
}

// Unlink removes the named account from sigd, returning it to an unlinked state.
func (c *Client) Unlink(ctx context.Context, account string) error {
	params := unlinkParams{Account: account}
	return c.call(ctx, schema.MethodUnlink, params, nil)
}

// stealthSetParams is the wire payload for stealth.set.
type stealthSetParams struct {
	Account string `json:"account"`
	Enabled bool   `json:"enabled"`
}

// stealthSetResult is the wire response for stealth.set.
type stealthSetResult struct {
	Account string `json:"account"`
	Stealth bool   `json:"stealth"`
}

// stealthStatusParams is the wire payload for stealth.status.
type stealthStatusParams struct {
	Account string `json:"account"`
}

// stealthStatusResult is the wire response for stealth.status.
type stealthStatusResult struct {
	Account        string `json:"account"`
	GlobalStealth  bool   `json:"global_stealth"`
	AccountStealth bool   `json:"account_stealth"`
	Active         bool   `json:"active"`
}

// StealthSet enables or disables stealth mode for the given account.
func (c *Client) StealthSet(ctx context.Context, account string, enabled bool) (*stealthSetResult, error) {
	params := stealthSetParams{Account: account, Enabled: enabled}
	var result stealthSetResult
	if err := c.call(ctx, schema.MethodStealthSet, params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// StealthStatus returns the current stealth mode status for the given account.
func (c *Client) StealthStatus(ctx context.Context, account string) (*stealthStatusResult, error) {
	params := stealthStatusParams{Account: account}
	var result stealthStatusResult
	if err := c.call(ctx, schema.MethodStealthStatus, params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// receiveParams is the wire payload for the receive RPC request.
type receiveParams struct {
	Account string `json:"account,omitempty"`
	Timeout int    `json:"timeout,omitempty"`
	Limit   int    `json:"limit,omitempty"`
}

// receiveResult is the wire response for the receive RPC request.
type receiveResult struct {
	Messages []schema.Envelope `json:"messages"`
	More     bool              `json:"more"`
}

// Receive polls sigd for queued messages. If account is non-empty only that
// account is polled; otherwise all accounts are polled. Timeout is the
// server-side long-poll duration in seconds (0 uses server default).
// Limit caps the number of messages returned per call (0 uses server default).
// The returned bool indicates whether more messages remain queued.
func (c *Client) Receive(ctx context.Context, account string, timeout, limit int) ([]schema.Envelope, bool, error) {
	params := receiveParams{Account: account, Timeout: timeout, Limit: limit}
	var result receiveResult
	if err := c.call(ctx, schema.MethodReceive, params, &result); err != nil {
		return nil, false, err
	}
	return result.Messages, result.More, nil
}

// Subscribe sends the subscribe RPC to sigd to register this connection for
// push notifications, then returns a channel on which incoming notifications
// are delivered. The caller must call the returned cancel function to release
// resources when it no longer needs notifications.
//
// accounts is an optional filter; pass no arguments to receive notifications
// for all linked accounts.
//
// The local channel is registered before the RPC is sent so that notifications
// arriving immediately after the ack are not lost.
func (c *Client) Subscribe(ctx context.Context, accounts ...string) (<-chan *schema.Envelope, func(), error) {
	// Register the local channel first so no notifications are missed between
	// the ack and the caller entering the receive loop.
	ch, cancelLocal := c.localSubscribe()

	params := subscribeParams{Accounts: accounts}
	var result subscribeResult
	if err := c.call(ctx, schema.MethodSubscribe, params, &result); err != nil {
		cancelLocal()
		return nil, nil, err
	}

	return ch, cancelLocal, nil
}
