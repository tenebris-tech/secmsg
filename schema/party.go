package schema

// Party identifies a participant in a message exchange.
// ID and Name are intentionally generic to support multiple messaging backends.
// Additional fields can be added here as more contact data becomes available.
type Party struct {
	ID         string `json:"id,omitempty"`          // ACI, phone number, or service identifier
	Name       string `json:"name,omitempty"`        // display name when known
	Device     uint32 `json:"device,omitempty"`      // device ID (non-zero when known)
	About      string `json:"about,omitempty"`       // bio text (with emoji prefix when both present)
	AboutEmoji string `json:"about_emoji,omitempty"` // profile emoji
	Avatar     string `json:"avatar,omitempty"`      // CDN path
}
