package schema

// JSON-RPC error codes defined by the secmsg protocol. Standard JSON-RPC 2.0
// codes (-32700..-32600, -32601, -32602, -32603) are used as usual; the codes
// below are protocol-specific extensions in the implementation-defined
// -32000..-32099 range. Shared here so daemons emit and clients recognize the
// same values without hardcoding magic numbers.
const (
	// ErrCodeStealth indicates an operation was refused because the target
	// account is in stealth (receive-only) mode. Returned by send methods.
	ErrCodeStealth = -32005
)
