package schema

const MethodTyping = "typing"

// Typing action values for TypingParams.Action.
const (
	TypingActionStarted = "started"
	TypingActionStopped = "stopped"
)

// TypingParams is the params payload for a "typing" notification.
type TypingParams struct {
	Account string `json:"account"`
	From    Party  `json:"from"`
	To      Party  `json:"to"`
	Action  string `json:"action"`
}
