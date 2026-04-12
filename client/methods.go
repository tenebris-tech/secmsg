package client

import (
	"encoding/json"

	"github.com/tenebris-tech/secmsg/schema"
)

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
type typingParams struct {
	Service string `json:"service"`
	Account string `json:"account"`
	To      string `json:"to"`
	Typing  bool   `json:"typing"`
}

// SendMessage sends a text message to a 1:1 recipient.
func (c *Client) SendMessage(service, account, to, body string) error {
	params := sendMessageParams{
		Service: service,
		Account: account,
		To:      to,
		Body:    body,
	}
	return c.call("send", params, nil)
}

// SendGroupMessage sends a text message to a group.
func (c *Client) SendGroupMessage(service, account, groupID, body string) error {
	params := sendGroupMessageParams{
		Service: service,
		Account: account,
		GroupID: groupID,
		Body:    body,
	}
	return c.call("send.group", params, nil)
}

// LinkRequest initiates device linking and returns the current link state.
func (c *Client) LinkRequest(account, name string) (*schema.LinkReply, error) {
	params := schema.LinkRequestParams{
		Account: account,
		Name:    name,
	}
	var reply schema.LinkReply
	if err := c.call(schema.MethodLinkRequest, params, &reply); err != nil {
		return nil, err
	}
	return &reply, nil
}

// LinkStatus returns the current link state without starting a new link flow.
func (c *Client) LinkStatus(account string) (*schema.LinkReply, error) {
	params := map[string]string{"account": account}
	var reply schema.LinkReply
	if err := c.call(schema.MethodLinkStatus, params, &reply); err != nil {
		return nil, err
	}
	return &reply, nil
}

// Contacts returns the contact list for the given account.
func (c *Client) Contacts(service, account string) ([]schema.Party, error) {
	params := map[string]string{"service": service, "account": account}
	var result []schema.Party
	if err := c.call("contacts", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Groups returns the group list for the given account.
func (c *Client) Groups(service, account string) ([]schema.Party, error) {
	params := map[string]string{"service": service, "account": account}
	var result []schema.Party
	if err := c.call("groups", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// SendReceiptRead sends read receipts for one or more message timestamps.
func (c *Client) SendReceiptRead(service, account, to string, timestamps []uint64) error {
	params := receiptReadParams{
		Service:    service,
		Account:    account,
		To:         to,
		Timestamps: timestamps,
	}
	return c.call("receipt.read", params, nil)
}

// SendTyping sends a typing started or stopped indicator.
func (c *Client) SendTyping(service, account, to string, typing bool) error {
	params := typingParams{
		Service: service,
		Account: account,
		To:      to,
		Typing:  typing,
	}
	return c.call("typing", params, nil)
}

// statusParams is the wire payload for the status request.
type statusParams struct {
	Account string `json:"account,omitempty"`
}

// Status returns the linked/connected state for one or all accounts.
// account may be empty to request all accounts.
// The returned bytes are the raw JSON result from sigd.
func (c *Client) Status(account string) (json.RawMessage, error) {
	params := statusParams{Account: account}
	var raw json.RawMessage
	if err := c.call("status", params, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// unlinkParams is the wire payload for the unlink request.
type unlinkParams struct {
	Account string `json:"account"`
}

// Unlink removes the named account from sigd, returning it to an unlinked state.
func (c *Client) Unlink(account string) error {
	params := unlinkParams{Account: account}
	return c.call("unlink", params, nil)
}
