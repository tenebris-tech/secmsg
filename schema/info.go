package schema

// Method name constants for the discovery and capability protocol (proto 1).
const (
	MethodHello        = "hello"
	MethodInfo         = "info"
	MethodAccountsList = "accounts.list"
)

// ProtoVersion is the current protocol version defined by this package.
const ProtoVersion = 1

// AccountIDField describes one identifier key in the account_id_schema.
// Key is the string key used in an account's identifiers map; Label is a
// human-readable display name clients can show without hardcoding
// service-specific field names (e.g. "aci", "phone").
type AccountIDField struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// InfoParams is the payload of the hello notification and the info command
// result. Both return identical data; clients that read the hello notification
// on connect may skip the explicit info call.
type InfoParams struct {
	Proto           int              `json:"proto"`
	Daemon          string           `json:"daemon"`
	Version         string           `json:"version"`
	Service         string           `json:"service"`
	LinkMethod      string           `json:"link_method"`
	Accounts        int              `json:"accounts"`
	AccountIDSchema []AccountIDField `json:"account_id_schema"`
	Capabilities    []string         `json:"capabilities"`
}
