package schema

const MethodConversationCleared = "conversation.cleared"

// ConversationClearedParams is the params payload for a "conversation.cleared" notification.
// Peer identifies the other party: ACI for 1:1, hex-encoded group ID for groups.
// IsFullDelete is true when all messages were deleted; false when only cleared/archived.
type ConversationClearedParams struct {
	Service      string `json:"service"`
	Account      string `json:"account"`
	Peer         string `json:"peer"`
	IsFullDelete bool   `json:"is_full_delete,omitempty"`
}
