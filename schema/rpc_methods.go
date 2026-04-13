package schema

// RPC method name constants for use in both server dispatch and client calls.
const (
	MethodSend         = "send"
	MethodSendGroup    = "send.group"
	MethodReceive      = "receive"
	MethodSubscribe    = "subscribe"
	MethodContactsList = "contacts.list"
	MethodGroupsList   = "groups.list"
	MethodReceiptRead  = "receipt.read"
	MethodTyping       = "typing"
	MethodStatus       = "status"
	MethodUnlink       = "unlink"
)
