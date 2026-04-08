package schema

const MethodReceipt = "receipt"

// Receipt type values for ReceiptParams.Type.
const (
	ReceiptTypeDelivery = "delivery"
	ReceiptTypeRead     = "read"
	ReceiptTypeViewed   = "viewed"
)

// ReceiptParams is the params payload for a "receipt" notification.
type ReceiptParams struct {
	Service    string   `json:"service"`
	Account    string   `json:"account"`
	From       Party    `json:"from"`
	To         Party    `json:"to"`
	Type       string   `json:"type"`
	Timestamps []uint64 `json:"timestamps"`
}
