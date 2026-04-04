package schema

// Party identifies a participant in a message exchange.
// ID and Name are intentionally generic to support multiple messaging backends.
// Additional fields can be added here as more contact data becomes available.
type Party struct {
	ID   string `json:"id,omitempty"`   // ACI, phone number, or service identifier
	Name string `json:"name,omitempty"` // display name when known
}
