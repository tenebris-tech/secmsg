package schema

const MethodLinkQR = "link.qr"

// LinkQRParams is the params payload for a "link.qr" notification.
// URI is the sgnl:// provisioning URI to be rendered as a QR code.
// Account is the name the user provided for this link.
type LinkQRParams struct {
	Account string `json:"account"`
	From    Party  `json:"from"`
	To      Party  `json:"to"`
	URI     string `json:"uri"`
}
