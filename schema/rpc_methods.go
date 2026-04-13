package schema

// RPC method name constants for use in both server dispatch and client calls.
const (
	MethodSend         = "send"
	MethodSendGroup    = "send.group"
	MethodContactsList = "contacts.list"
	MethodGroupsList   = "groups.list"
	MethodReceiptRead  = "receipt.read"
	MethodTyping       = "typing"
	MethodStatus       = "status"
	MethodUnlink       = "unlink"
)
