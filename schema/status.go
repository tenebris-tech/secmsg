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

// StatusReply is the response for a single-account "status" RPC call.
type StatusReply struct {
	Linked    bool   `json:"linked"`
	Connected bool   `json:"connected"`
	Account   string `json:"account"`
	ACI       string `json:"aci,omitempty"`
	Phone     string `json:"phone,omitempty"`
}

// StatusAllReply is the response for an all-accounts "status" RPC call.
type StatusAllReply struct {
	Accounts []StatusReply `json:"accounts"`
}
