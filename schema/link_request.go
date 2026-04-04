package schema

const MethodLinkRequest = "link.request"

// Link status values returned in LinkReply.Status.
const (
	LinkStatusPending  = "pending"  // QR displayed or waiting for scan
	LinkStatusComplete = "complete" // linking succeeded
	LinkStatusError    = "error"    // linking failed
)

// LinkRequestParams are the params the client sends to initiate device linking.
// Name is the user-provided label for this account (e.g. "myphone"), which sigd
// uses to form the profile identifier (e.g. "signal-myphone").
type LinkRequestParams struct {
	Name string `json:"name"`
}

// LinkReply is returned by both link.request and link.status.
// Status is always present. URI is set when a QR code is ready to display.
// ACI and Phone are set on completion. Error is set on failure.
type LinkReply struct {
	Status string `json:"status"`
	URI    string `json:"uri,omitempty"`
	ACI    string `json:"aci,omitempty"`
	Phone  string `json:"phone,omitempty"`
	Error  string `json:"error,omitempty"`
}
