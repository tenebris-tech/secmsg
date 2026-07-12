package schema

// StatusReply is the response for a single-account "status" RPC call.
// Identifiers contains service-specific fields (e.g. "phone", "user_id") as
// declared in the daemon's account_id_schema from the hello/info response.
type StatusReply struct {
	Linked      bool              `json:"linked"`
	Connected   bool              `json:"connected"`
	Account     string            `json:"account"`
	Stealth     bool              `json:"stealth"`
	Identifiers map[string]string `json:"identifiers,omitempty"`
}

// StatusAllReply is the response for an all-accounts "status" RPC call.
type StatusAllReply struct {
	Accounts []StatusReply `json:"accounts"`
}
