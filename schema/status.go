package schema

const MethodStatus = "status"

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
