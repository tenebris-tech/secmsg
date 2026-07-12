package schema

import (
	"fmt"
	"strings"
)

// Party identifies a participant in a message exchange.
// ID and Name are intentionally generic to support multiple messaging backends.
// Additional fields can be added here as more contact data becomes available.
type Party struct {
	ID         string `json:"id,omitempty"`          // user identifier, phone number, or service-specific identifier
	Name       string `json:"name,omitempty"`        // display name when known
	Device     uint32 `json:"device,omitempty"`      // device ID (non-zero when known)
	About      string `json:"about,omitempty"`       // bio text (with emoji prefix when both present)
	AboutEmoji string `json:"about_emoji,omitempty"` // profile emoji
	Avatar     string `json:"avatar,omitempty"`      // CDN path
	Self       bool   `json:"self,omitempty"`        // true when this party is the local account
}

// Format renders the party as uuid[device]name.
// Self parties render as "me" or "me[N]" (device N when known).
// Brackets are always present for non-self parties except when both ID and Name are empty.
// Device 0 means unknown and renders as []. Device > 0 renders as [N].
// Examples:
//
//	Party{Self: true}.Format()                           => "me"
//	Party{Self: true, Device: 3}.Format()                => "me[3]"
//	Party{ID: "uuid", Device: 3, Name: "Eric"}.Format() => "uuid[3]Eric"
//	Party{ID: "uuid", Name: "Eric"}.Format()             => "uuid[]Eric"
//	Party{ID: "uuid", Device: 1}.Format()                => "uuid[1]"
//	Party{ID: "uuid"}.Format()                           => "uuid[]"
//	Party{Device: 4, Name: "Eric"}.Format()              => "[4]Eric"
//	Party{Name: "Eric"}.Format()                         => "[]Eric"
//	Party{}.Format()                                     => ""
func (p Party) Format() string {
	if p.Self {
		if p.Device > 0 {
			return fmt.Sprintf("me[%d]", p.Device)
		}
		return "me"
	}
	if p.ID == "" && p.Name == "" {
		return ""
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
