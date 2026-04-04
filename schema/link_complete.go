package schema

const MethodLinkComplete = "link.complete"

// LinkCompleteParams is the params payload for a "link.complete" notification.
// Sent when device linking finishes successfully.
type LinkCompleteParams struct {
	Account string `json:"account"`
	From    Party  `json:"from"`
	To      Party  `json:"to"`
	ACI     string `json:"aci"`
	Phone   string `json:"phone"`
}
