// Package schema defines the wire types, method name constants, and
// notification payloads for the sigd JSON-RPC 2.0 protocol.
package schema

import "encoding/json"

// Envelope is the outer wrapper for all server-push notifications.
// Parse Method first, then unmarshal Params into the appropriate typed struct.
type Envelope struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}
