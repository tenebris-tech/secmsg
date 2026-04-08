package schema

const MethodStatus = "status"

// StatusParams is the params payload for a "status" notification.
// Sent when the Signal WebSocket connection state changes.
type StatusParams struct {
	Service   string `json:"service"`
	Account   string `json:"account"`
	From      Party  `json:"from"`
	To        Party  `json:"to"`
	Connected bool   `json:"connected"`
}
