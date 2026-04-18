package schema

// Message type values for MessageParams.Type.
const (
	MessageTypeText     = "text"
	MessageTypeEdit     = "edit"
	MessageTypeRetract  = "retract"
	MessageTypeReaction = "reaction"
	MessageTypeSticker  = "sticker"
)

// MessageParams is the params payload for a "message" notification.
// Account identifies which linked account received or sent the message.
// From is the sender; To is the recipient.
// For inbound messages From is the remote party and To is our account.
// For outbound sync messages (copies of messages we sent) From is our account and To is the remote party.
// Ref is the target message timestamp for edit, retract, and reaction types.
// RefAuthor is the ACI of the author of the referenced message (reaction only).
type MessageParams struct {
	Service     string       `json:"service"`
	Account     string       `json:"account"`
	From        Party        `json:"from"`
	To          Party        `json:"to"`
	Type        string       `json:"type"`
	Timestamp   uint64       `json:"timestamp"`
	Ref         uint64       `json:"ref,omitempty"`
	RefAuthor   string       `json:"ref_author,omitempty"`
	Body        string       `json:"body,omitempty"`
	ViewOnce    bool         `json:"view_once,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}
