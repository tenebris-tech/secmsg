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
	// MethodReceive and MethodSubscribe are not implemented by this package.
	// They are defined here for use by sigd, which imports this schema package
	// and uses them as server-side RPC method names.
	MethodReceive   = "receive"
	MethodSubscribe = "subscribe"
)
