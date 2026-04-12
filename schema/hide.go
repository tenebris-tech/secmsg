package schema

const MethodHide = "message.hidden"

// HideParams is the params payload for a "message.hidden" notification.
// Emitted when the user locally hides one or more messages ("Delete for Me")
// on another device. This is a local-only action with no network effect.
type HideParams struct {
	Service string   `json:"service"`
	Account string   `json:"account"`
	Peer    string   `json:"peer"`
	Ref     []uint64 `json:"ref,omitempty"`
}
