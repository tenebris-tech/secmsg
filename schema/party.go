package schema

import (
	"fmt"
	"strings"
)

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

// Format renders the party as uuid[device]name.
// Brackets are always present except when both ID and Name are empty ("self").
// Device 0 means unknown and renders as []. Device > 0 renders as [N].
// Examples:
//
//	Party{ID: "uuid", Device: 3, Name: "Eric"}.Format() => "uuid[3]Eric"
//	Party{ID: "uuid", Name: "Eric"}.Format()             => "uuid[]Eric"
//	Party{ID: "uuid", Device: 1}.Format()                => "uuid[1]"
//	Party{ID: "uuid"}.Format()                           => "uuid[]"
//	Party{Device: 4, Name: "Eric"}.Format()              => "[4]Eric"
//	Party{Name: "Eric"}.Format()                         => "[]Eric"
//	Party{}.Format()                                     => "self"
func (p Party) Format() string {
	if p.ID == "" && p.Name == "" {
		return "self"
	}
	var sb strings.Builder
	if p.ID != "" {
		sb.WriteString(p.ID)
	}
	if p.Device > 0 {
		fmt.Fprintf(&sb, "[%d]", p.Device)
	} else {
		sb.WriteString("[]")
	}
	if p.Name != "" {
		sb.WriteString(p.Name)
	}
	return sb.String()
}
