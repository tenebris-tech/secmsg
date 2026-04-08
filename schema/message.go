package schema

const MethodMessage = "message"

// Message type values for MessageParams.Type.
const (
	MessageTypeText   = "text"
	MessageTypeEdit   = "edit"
	MessageTypeDelete = "delete"
)

// MessageParams is the params payload for a "message" notification.
// Account identifies which linked account received or sent the message.
// From is the sender; To is the recipient.
// For inbound messages From is the remote party and To is our account.
// For outbound sync messages (copies of messages we sent) From is our account and To is the remote party.
// RefTimestamp is the target message timestamp for edit and delete types.
type MessageParams struct {
	Service      string `json:"service"`
	Account      string `json:"account"`
	From         Party  `json:"from"`
	To           Party  `json:"to"`
	Type         string `json:"type"`
	Timestamp    uint64 `json:"timestamp"`
	RefTimestamp uint64       `json:"ref_timestamp,omitempty"`
	Body         string       `json:"body,omitempty"`
	Attachments  []Attachment `json:"attachments,omitempty"`
}
