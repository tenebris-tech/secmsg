# secmsg

> [!WARNING]
> **This project is a work in progress and should not be used in production.**

A Go client library and CLI for [sigd](https://github.com/tenebris-tech/sigd), the Signal Messenger daemon. Lets you send and receive Signal messages from any application without carrying any Signal or cryptographic code.

```
[Signal servers] <--websocket--> [sigd (AGPL)] <--tcp/json-rpc--> [your app (MIT)]
```

## Package Layout

```
secmsg/
├── schema/         # Shared types: method constants, notification params, party/attachment structs
├── client/         # Go client library: Dial, Subscribe, Send, StealthSet, ...
└── cmd/secmsg/     # Reference CLI
```

`schema/` contains only plain Go structs and string constants — no Signal code, no CGO, no network dependencies. Import it in any project that needs to work with sigd notifications.

## License

MIT. The `schema/` and `client/` packages may be imported into projects of any license. The sigd daemon itself is AGPL-3.0.

## Building

```bash
go build ./...
go test ./...

# Install the secmsg CLI
go install ./cmd/secmsg
```

## CLI Quick Start

```bash
# Link sigd to your Signal account (scan the QR code with your phone)
secmsg link myaccount MyDevice
secmsg poll-link myaccount

# Check status
secmsg status

# Send a message
secmsg send myaccount <aci-uuid> "Hello from sigd"

# Stream incoming messages
secmsg subscribe

# Stealth mode
secmsg stealth-enable myaccount
secmsg stealth-status myaccount
secmsg stealth-disable myaccount
```

Default sigd address: `127.0.0.1:9801`. Override with `-addr`:

```bash
secmsg -addr 10.0.0.1:9801 status
```

## CLI Commands

```
secmsg [flags] <command> [args...]

Flags:
  -addr string   sigd address (default "127.0.0.1:9801")
  -json          output as JSON
  -debug         enable debug logging to /tmp/secmsg.log

Commands:
  send            <account> <to> <message>
  send-group      <account> <groupId> <message>
  contacts        <account>
  groups          <account>
  receipt-read    <account> <to> <timestamp> [<timestamp>...]
  typing          <account> <to> <true|false>
  link            <account> <name>
  link-status     <account>
  poll-link       <account>
  status          [account]
  unlink          <account>
  subscribe
  stealth-enable  <account>
  stealth-disable <account>
  stealth-status  <account>
  version
  help
```

## Go Client Library

```go
import "github.com/tenebris-tech/secmsg/client"
import "github.com/tenebris-tech/secmsg/schema"

c, err := client.Dial("127.0.0.1:9801")
if err != nil { ... }
defer c.Close()

ctx := context.Background()

// Send a message
err = c.SendMessage(ctx, schema.ServiceSignal, "myaccount", recipientACI, "Hello")

// Subscribe to incoming notifications
ch, cancel, err := c.Subscribe(ctx)
if err != nil { ... }
defer cancel()

for env := range ch {
    fmt.Printf("method=%s params=%s\n", env.Method, env.Params)
}

// Stealth mode
result, err := c.StealthSet(ctx, "myaccount", true)
status, err := c.StealthStatus(ctx, "myaccount")
```

---

## Protocol Reference

sigd uses [JSON-RPC 2.0](https://www.jsonrpc.org/specification) over a persistent TCP connection. All messages are newline-delimited JSON — one complete JSON object per line.

### Transport

- **Default address:** `127.0.0.1:9801`
- **Framing:** `\n`-terminated JSON lines
- **Direction:** request/response for commands; server-push for notifications on subscribe connections
- **TLS:** optional; recommended for non-loopback deployments

### Connection handshake

Immediately after a client connects, before the client sends any request, the server sends a `hello` notification:

```json
{
  "jsonrpc": "2.0",
  "method": "hello",
  "params": {
    "proto": 1,
    "daemon": "sigd",
    "version": "0.3.0",
    "service": "signal",
    "link_method": "qr",
    "accounts": 1,
    "account_id_schema": [
      {"key": "aci",   "label": "Account ID"},
      {"key": "phone", "label": "Phone Number"}
    ],
    "capabilities": [
      "send", "send.group", "receive", "subscribe",
      "contacts.list", "groups.list", "receipt.read", "typing",
      "attachments.metadata", "attachments.download"
    ]
  }
}
```

`capabilities` lists the features this daemon instance supports. Clients should check capabilities before calling optional methods. `attachments.download` is only advertised when `--attachment-dir` is configured.

---

## Methods

### `info`

Returns the same payload as the `hello` notification. Works even when no account is linked.

```json
// Request
{"jsonrpc":"2.0","id":1,"method":"info"}

// Response
{"jsonrpc":"2.0","id":1,"result":{
  "proto":1,"daemon":"sigd","version":"0.3.0","service":"signal",
  "link_method":"qr","accounts":1,
  "account_id_schema":[{"key":"aci","label":"Account ID"},{"key":"phone","label":"Phone Number"}],
  "capabilities":["send","receive","subscribe",...]
}}
```

---

### `accounts.list`

Lists all linked accounts. Returns an empty list when no accounts are linked.

```json
// Request
{"jsonrpc":"2.0","id":1,"method":"accounts.list"}

// Response
{"jsonrpc":"2.0","id":1,"result":{
  "accounts":[{
    "account":"myaccount",
    "identifiers":{"aci":"3c3b4200-...","phone":"+15551234567"},
    "linked":true,
    "connected":true
  }]
}}
```

---

### `status`

Returns link and connection state. When `account` is omitted, returns all accounts.

```json
// Single account
{"jsonrpc":"2.0","id":1,"method":"status","params":{"account":"myaccount"}}
{"jsonrpc":"2.0","id":1,"result":{
  "service":"signal","account":"myaccount",
  "linked":true,"connected":true,
  "identifiers":{"aci":"3c3b4200-...","phone":"+15551234567"}
}}

// All accounts (omit params or pass {})
{"jsonrpc":"2.0","id":1,"method":"status"}
{"jsonrpc":"2.0","id":1,"result":{
  "service":"signal",
  "accounts":[{"account":"myaccount","linked":true,"connected":true,"identifiers":{...}}]
}}
```

---

### `link.request`

Starts the QR code device linking flow. Returns the `sgnl://linkdevice?...` URI immediately with `status: "pending"`. Poll `link.status` to detect completion.

```json
// Request
{"jsonrpc":"2.0","id":1,"method":"link.request","params":{"account":"myaccount","name":"MyDevice"}}

// Response — QR URI ready
{"jsonrpc":"2.0","id":1,"result":{"status":"pending","uri":"sgnl://linkdevice?uuid=..."}}
```

---

### `link.status`

Polls for completion of a pending `link.request`. Call repeatedly (e.g. once per second) until status is `"complete"` or `"error"`.

```json
// Request
{"jsonrpc":"2.0","id":2,"method":"link.status","params":{"account":"myaccount"}}

// Still waiting
{"jsonrpc":"2.0","id":2,"result":{"status":"pending"}}

// Linked successfully
{"jsonrpc":"2.0","id":2,"result":{
  "status":"complete","service":"signal","account":"myaccount",
  "aci":"3c3b4200-...","phone":"+15551234567"
}}

// Failed
{"jsonrpc":"2.0","id":2,"result":{"status":"error","error":"linking failed"}}
```

---

### `unlink`

Removes the linked account. The daemon must be restarted to reconnect.

```json
{"jsonrpc":"2.0","id":1,"method":"unlink","params":{"account":"myaccount"}}
{"jsonrpc":"2.0","id":1,"result":{}}
```

---

### `send`

Sends a 1:1 text message. `to` must be an ACI UUID or E.164 phone number (E.164 resolved via local contact store).

```json
// Request
{"jsonrpc":"2.0","id":1,"method":"send","params":{
  "account":"myaccount","to":"3c3b4200-...","body":"Hello"
}}

// Response
{"jsonrpc":"2.0","id":1,"result":{"timestamp":1744000000000}}
```

---

### `send.group`

Sends a text message to a Signal group.

```json
{"jsonrpc":"2.0","id":1,"method":"send.group","params":{
  "groupId":"base64groupid==","body":"Hello group"
}}
{"jsonrpc":"2.0","id":1,"result":{"timestamp":1744000000000}}
```

---

### `receive`

Polls for queued messages without subscribing. Blocks up to `timeout` seconds (1–300; default 30).

```json
{"jsonrpc":"2.0","id":1,"method":"receive","params":{"timeout":10}}
{"jsonrpc":"2.0","id":1,"result":{
  "messages":[{
    "from":"3c3b4200-...","body":"Hello","timestamp":1744000000000,"type":"text"
  }]
}}
```

---

### `subscribe`

Registers the connection for push notifications. After the ack, the server pushes unsolicited notification lines whenever events occur. The connection becomes a one-way notification stream — no further request/response framing is expected.

```json
// Request — subscribe to all accounts
{"jsonrpc":"2.0","id":1,"method":"subscribe"}

// Request — subscribe to specific accounts
{"jsonrpc":"2.0","id":1,"method":"subscribe","params":{"accounts":["myaccount"]}}

// Ack
{"jsonrpc":"2.0","id":1,"result":{"subscribed":true,"accounts":["myaccount"]}}

// Subsequent lines are push notifications (see Notifications section)
```

---

### `contacts.list`

Returns all contacts with full profile data (name, about, avatar CDN path).

```json
{"jsonrpc":"2.0","id":1,"method":"contacts.list"}
{"jsonrpc":"2.0","id":1,"result":{
  "contacts":[{
    "aci":"3c3b4200-...","phone":"+15551234567","name":"Alice",
    "about":"Hello there","about_emoji":"👋","avatar":"profiles/..."
  }]
}}
```

---

### `groups.list`

Returns known groups.

```json
{"jsonrpc":"2.0","id":1,"method":"groups.list"}
{"jsonrpc":"2.0","id":1,"result":{
  "groups":[{"id":"base64groupid==","memberCount":5}]
}}
```

---

### `receipt.read`

Sends read receipts for one or more message timestamps. Behaviour is controlled by the `send_read_receipts` daemon setting:

- `send_read_receipts: false` (default) — succeeds silently without sending to Signal
- `send_read_receipts: true` — actually sends to Signal
- Stealth mode active — returns error `-32005`

```json
{"jsonrpc":"2.0","id":1,"method":"receipt.read","params":{
  "to":"3c3b4200-...","ref":[1744000000000,1744000001000]
}}
{"jsonrpc":"2.0","id":1,"result":{}}
```

---

### `typing`

Sends a typing started or stopped indicator.

```json
{"jsonrpc":"2.0","id":1,"method":"typing","params":{
  "account":"myaccount","to":"3c3b4200-...","typing":true
}}
{"jsonrpc":"2.0","id":1,"result":{}}
```

---

### `stealth.set`

Enables or disables account-level stealth mode at runtime. When active, automatic delivery receipts are suppressed and all client-initiated sends, receipts, and typing indicators return error `-32005`.

Global stealth (set at daemon startup via `--stealth`) always overrides account-level stealth.

```json
// Enable stealth
{"jsonrpc":"2.0","id":1,"method":"stealth.set","params":{"account":"myaccount","enabled":true}}
{"jsonrpc":"2.0","id":1,"result":{"account":"myaccount","stealth":true}}

// Disable stealth
{"jsonrpc":"2.0","id":1,"method":"stealth.set","params":{"account":"myaccount","enabled":false}}
{"jsonrpc":"2.0","id":1,"result":{"account":"myaccount","stealth":false}}
```

---

### `stealth.status`

Queries the current stealth state for an account.

```json
{"jsonrpc":"2.0","id":1,"method":"stealth.status","params":{"account":"myaccount"}}
{"jsonrpc":"2.0","id":1,"result":{
  "account":"myaccount",
  "global_stealth":false,
  "account_stealth":true,
  "active":true
}}
```

Fields:

| Field | Description |
|-------|-------------|
| `global_stealth` | Whether the daemon was started with `--stealth` |
| `account_stealth` | Whether `stealth.set` was called for this account |
| `active` | Whether stealth is currently in effect (`global_stealth \|\| account_stealth`) |

---

## Notifications

After `subscribe`, the server pushes unsolicited JSON-RPC 2.0 notifications. No `id` field is present on notifications.

### `hello`

Sent immediately on every new TCP connection, before any client request.

```json
{
  "jsonrpc":"2.0","method":"hello",
  "params":{"proto":1,"daemon":"sigd","version":"0.3.0","service":"signal",...}
}
```

---

### `message`

Incoming text message, edit, retract, or outbound-sync copy (message sent from another linked device).

```json
{
  "jsonrpc":"2.0","method":"message",
  "params":{
    "service":"signal","account":"myaccount",
    "from":{"id":"sender-aci","name":"Alice","device":1,"about":"Hello","avatar":"profiles/..."},
    "to":  {"id":"our-aci","name":"Me"},
    "type":"text",
    "body":"Hello",
    "timestamp":1744000000000,
    "attachments":[{
      "content_type":"image/jpeg","file_name":"photo.jpg","size":45231,
      "local_path":"/home/user/.sigd/files/1744000000000-photo.jpg"
    }]
  }
}
```

**`type` values:**

| Value | Meaning |
|-------|---------|
| `text` | New message |
| `edit` | Edit of a previous message; `ref` contains the original timestamp |
| `retract` | Delete-for-everyone; `ref` contains the original timestamp |

**Party fields (`from` / `to`):**

| Field | Description |
|-------|-------------|
| `id` | ACI UUID |
| `name` | Display name (from profile cache or contact store) |
| `device` | Sender device ID (present on `from` only for inbound messages) |
| `about` | Bio text |
| `about_emoji` | Profile emoji |
| `avatar` | CDN path or local path when avatar download is enabled |

**Outbound sync** (sent from another linked device): `from.id` is your own ACI, `to` is the recipient.

---

### `receipt`

Delivery, read, or viewed receipt.

```json
{
  "jsonrpc":"2.0","method":"receipt",
  "params":{
    "service":"signal","account":"myaccount",
    "from":{"id":"sender-aci"},
    "to":  {"id":"our-aci"},
    "type":"delivery",
    "ref":[1744000000000]
  }
}
```

**`type` values:** `delivery`, `read`, `viewed`

---

### `typing`

Typing indicator from a contact.

```json
{
  "jsonrpc":"2.0","method":"typing",
  "params":{
    "service":"signal","account":"myaccount",
    "from":{"id":"sender-aci","name":"Alice"},
    "to":  {"id":"our-aci"},
    "action":"started"
  }
}
```

**`action` values:** `started`, `stopped`

---

### `conversation.deleted`

Fired when the user deletes or clears a conversation on another device (DeleteForMe sync).

```json
{
  "jsonrpc":"2.0","method":"conversation.deleted",
  "params":{
    "service":"signal","account":"myaccount",
    "peer":"other-party-aci-or-group-id",
    "is_full_delete":true
  }
}
```

---

### `attachment.ready`

Fired when an attachment has been downloaded and stored locally. Only emitted when `--attachment-dir` is configured.

```json
{
  "jsonrpc":"2.0","method":"attachment.ready",
  "params":{
    "service":"signal","account":"myaccount",
    "timestamp":1744000000000,
    "index":0,
    "local_path":"/home/user/.sigd/files/1744000000000-photo.jpg"
  }
}
```

---

### `status`

Connection state change (Signal WebSocket connected or disconnected).

```json
{"jsonrpc":"2.0","method":"status","params":{"connected":true}}
```

---

## Error Codes

| Code | Name | Description |
|------|------|-------------|
| `-32700` | Parse error | Request is not valid JSON |
| `-32600` | Invalid request | JSON-RPC envelope is malformed |
| `-32601` | Method not found | Unknown method name |
| `-32602` | Invalid params | Missing or invalid parameter |
| `-32000` | Not linked | Operation requires a linked account |
| `-32001` | Not connected | Operation requires an active Signal connection |
| `-32002` | Recipient not found | ACI or phone number could not be resolved |
| `-32003` | Rate limited | Too many requests |
| `-32004` | Internal error | Unexpected daemon error |
| `-32005` | Stealth mode | Operation blocked because stealth mode is active |

---

## Copyright and License

Copyright (c) 2026 Tenebris Technologies Inc.

secmsg is licensed under the [MIT License](LICENSE).

The `schema/` and `client/` packages are intentionally kept free of Signal and cryptographic code so they can be imported by projects of any license. The AGPL-3.0 license of the sigd daemon applies only to sigd itself — not to clients that communicate with it over TCP.
