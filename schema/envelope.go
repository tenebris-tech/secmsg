package schema

import "encoding/json"

// Envelope is the outer wrapper for all server-push notifications.
// Parse Method first, then unmarshal Params into the appropriate typed struct.
type Envelope struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}
