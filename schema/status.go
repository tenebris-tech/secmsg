package schema

const MethodStatus = "status"

// StatusReply is the response for a single-account "status" RPC call.
// Identifiers contains service-specific fields (e.g. "phone", "aci") as
// declared in the daemon's account_id_schema from the hello/info response.
type StatusReply struct {
	Linked      bool              `json:"linked"`
	Connected   bool              `json:"connected"`
	Account     string            `json:"account"`
	Identifiers map[string]string `json:"identifiers,omitempty"`
}

// StatusAllReply is the response for an all-accounts "status" RPC call.
type StatusAllReply struct {
	Accounts []StatusReply `json:"accounts"`
}
