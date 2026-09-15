# secmsg

> [!WARNING]
> **This is a work in progress - use at your own risk**

A general-purpose secure messaging interface, Go client library, and CLI. It speaks a compact JSON-RPC 2.0 protocol to a messaging daemon, letting any application send and receive messages without carrying any service-specific or cryptographic code.

```
[messaging backend] <--native protocol--> [daemon] <--tcp/json-rpc--> [your app (MIT)]
```

secmsg is not tied to any single messaging service. Any daemon that implements the protocol below can be driven by this library — the daemon advertises its service identifier, capabilities, and account identifier schema in the connection handshake, and the client adapts to whatever it connects to.

## Compatible Daemons

| Daemon | Notes |
|--------|-------|
| [sigd](https://github.com/tenebris-tech/sigd) | Signal Messenger daemon (AGPL-3.0). |

Any daemon that implements the JSON-RPC 2.0 protocol described below is compatible.

## Package Layout

```
secmsg/
├── schema/         # Shared types: method constants, notification params, party/attachment structs
├── client/         # Go client library: Dial, Subscribe, Send, StealthSet, ...
└── cmd/secmsg/     # Reference CLI
```

`schema/` contains only plain Go structs and string constants — no service-specific code, no CGO, no network dependencies. Import it in any project that needs to work with a compatible daemon's notifications.

## License

MIT. The `schema/` and `client/` packages may be imported into projects of any license. Daemon licensing is independent of this library.

## Building

```bash
go build ./...
go test ./...

# Install the secmsg CLI
go install ./cmd/secmsg
```

## CLI Quick Start

```bash
# Link the daemon to your messaging account (scan the QR code with your phone)
secmsg link myaccount MyDevice
secmsg poll-link myaccount

# Check status
secmsg status

# Send a message
secmsg send myaccount <recipient-id> "Hello"

# Stream incoming messages
secmsg subscribe

# Stealth mode
secmsg stealth-enable myaccount
secmsg stealth-status myaccount
secmsg stealth-disable myaccount
```

Default daemon address: `127.0.0.1:9801`. Override with `-addr`:

```bash
secmsg -addr 10.0.0.1:9801 status
```

## CLI Commands

```
secmsg [flags] <command> [args...]

Flags:
  -addr string   daemon address (default "127.0.0.1:9801")
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

c, err := client.Dial("127.0.0.1:9801")
if err != nil { ... }
defer c.Close()

ctx := context.Background()

// The daemon advertises its service identifier in the connection handshake.
// Pass it to the send/receive methods; no service name is hardcoded.
service := c.Service()

// Send a message
err = c.SendMessage(ctx, service, "myaccount", recipientID, "Hello")

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

### Multiple Instances

A `Client` holds no package-level or global state — each one owns its own TCP
connection, request/response multiplexer, and notification dispatch. An
application may create as many independent clients as it needs and connect each
to a different daemon, so a single process can talk to an arbitrary number of
services that support the protocol concurrently.

```go
// Connect to several daemons at once; each client is fully independent.
work, err := client.Dial("127.0.0.1:9801")   // one service
if err != nil { ... }
defer work.Close()

personal, err := client.Dial("127.0.0.1:9802") // another service
if err != nil { ... }
defer personal.Close()

// Each client reports the service its daemon advertises.
_ = work.SendMessage(ctx, work.Service(), "acct", to, "from work daemon")
_ = personal.SendMessage(ctx, personal.Service(), "acct", to, "from personal daemon")
```

Each client's `Subscribe` returns its own notification channel, so an
application can fan messages from every connected service into one event loop.

---

## Protocol Reference

The protocol is [JSON-RPC 2.0](https://www.jsonrpc.org/specification) over a persistent TCP connection. All messages are newline-delimited JSON — one complete JSON object per line. Concrete values in the examples below (service name, identifiers, URIs) are illustrative placeholders; the actual values are reported by the daemon in the handshake.

### Transport

- **Default address:** `127.0.0.1:9801`
- **Framing:** `\n`-terminated JSON lines
- **Direction:** request/response for commands; server-push for notifications on subscribe connections
- **TLS:** optional; recommended for non-loopback deployments

### Connection handshake

Immediately after a client connects, before the client sends any request, the server sends a `hello` notification. The client captures this and exposes it via `Client.Info()` / `Client.Service()`:

```json
{
  "jsonrpc": "2.0",
  "method": "hello",
  "params": {
    "proto": 1,
    "daemon": "example-daemon",
    "version": "0.3.0",
    "service": "example",
    "link_method": "qr",
    "accounts": 1,
    "account_id_schema": [
      {"key": "user_id", "label": "User ID"},
      {"key": "phone",   "label": "Phone Number"}
    ],
    "capabilities": [
      "send", "send.group", "receive", "subscribe",
      "contacts.list", "groups.list", "receipt.read", "typing",
      "attachments.metadata", "attachments.download"
    ]
  }
}
```

`service` is the daemon-defined service identifier. `account_id_schema` describes the identifier keys this daemon uses, so clients can render them without hardcoding service-specific field names. `capabilities` lists the features this daemon instance supports; clients should check capabilities before calling optional methods. `attachments.download` is only advertised when `--attachment-dir` is configured.

---

## Methods

### `info`

Returns the same payload as the `hello` notification. Works even when no account is linked.

```json
// Request
{"jsonrpc":"2.0","id":1,"method":"info"}

// Response
{"jsonrpc":"2.0","id":1,"result":{
  "proto":1,"daemon":"example-daemon","version":"0.3.0","service":"example",
  "link_method":"qr","accounts":1,
  "account_id_schema":[{"key":"user_id","label":"User ID"},{"key":"phone","label":"Phone Number"}],
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
    "identifiers":{"user_id":"3c3b4200-...","phone":"+15551234567"},
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
  "service":"example","account":"myaccount",
  "linked":true,"connected":true,
  "identifiers":{"user_id":"3c3b4200-...","phone":"+15551234567"}
}}

// All accounts (omit params or pass {})
{"jsonrpc":"2.0","id":1,"method":"status"}
{"jsonrpc":"2.0","id":1,"result":{
  "service":"example",
  "accounts":[{"account":"myaccount","linked":true,"connected":true,"identifiers":{...}}]
}}
```

---

### `link.request`

Starts the QR code device linking flow. Returns a daemon-specific device-linking URI immediately with `status: "pending"`. Poll `link.status` to detect completion.

```json
// Request
{"jsonrpc":"2.0","id":1,"method":"link.request","params":{"account":"myaccount","name":"MyDevice"}}

// Response — link URI ready
{"jsonrpc":"2.0","id":1,"result":{"status":"pending","uri":"linkdevice://..."}}
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
  "status":"complete","service":"example","account":"myaccount",
  "user_id":"3c3b4200-...","phone":"+15551234567"
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

Sends a 1:1 text message. `to` is a user identifier — one of the keys advertised in `account_id_schema` (e.g. a `user_id` UUID or an E.164 phone number resolved via the local contact store).

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

Sends a text message to a group.

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
    "user_id":"3c3b4200-...","phone":"+15551234567","name":"Alice",
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

- `send_read_receipts: false` (default) — succeeds silently without sending the receipt upstream
- `send_read_receipts: true` — actually sends the receipt upstream
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
  "params":{"proto":1,"daemon":"example-daemon","version":"0.3.0","service":"example",...}
}
```

---

### `message`

Incoming text message, edit, retract, or outbound-sync copy (message sent from another linked device).

```json
{
  "jsonrpc":"2.0","method":"message",
  "params":{
    "service":"example","account":"myaccount",
    "from":{"id":"sender-id","name":"Alice","device":1,"about":"Hello","avatar":"profiles/..."},
    "to":  {"id":"self-id","name":"Me"},
    "type":"text",
    "body":"Hello",
    "timestamp":1744000000000,
    "attachments":[{
      "content_type":"image/jpeg","file_name":"photo.jpg","size":45231,
      "local_path":"/home/user/.local/files/1744000000000-photo.jpg"
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
| `reaction.add` | Emoji reaction added; `body` is the emoji, `ref` is the target message timestamp, `ref_author` is the target message author identifier |
| `reaction.remove` | Emoji reaction removed; same fields as `reaction.add` |
| `sticker` | Sticker message; `body` contains the sticker pack identifier |

**Party fields (`from` / `to`):**

| Field | Description |
|-------|-------------|
| `id` | User identifier |
| `name` | Display name (from profile cache or contact store) |
| `device` | Sender device ID (present on `from` only for inbound messages) |
| `about` | Bio text |
| `about_emoji` | Profile emoji |
| `avatar` | CDN path or local path when avatar download is enabled |
| `self` | `true` when this party is the local account; omitted otherwise |

**Outbound sync** (sent from another linked device): `from.self` is `true`, `to` is the recipient.

---

### `receipt`

Delivery, read, or viewed receipt.

```json
{
  "jsonrpc":"2.0","method":"receipt",
  "params":{
    "service":"example","account":"myaccount",
    "from":{"id":"sender-id"},
    "to":  {"id":"self-id"},
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
    "service":"example","account":"myaccount",
    "from":{"id":"sender-id","name":"Alice"},
    "to":  {"id":"self-id"},
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
    "service":"example","account":"myaccount",
    "peer":"other-party-id-or-group-id",
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
    "service":"example","account":"myaccount",
    "timestamp":1744000000000,
    "index":0,
    "local_path":"/home/user/.local/files/1744000000000-photo.jpg"
  }
}
```

---

### `status`

Connection state change (upstream service connected or disconnected).

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
| `-32001` | Not connected | Operation requires an active upstream connection |
| `-32002` | Recipient not found | Identifier could not be resolved |
| `-32003` | Rate limited | Too many requests |
| `-32004` | Internal error | Unexpected daemon error |
| `-32005` | Stealth mode | Operation blocked because stealth mode is active |

---

## Copyright and License

Copyright (c) 2026 Tenebris Technologies Inc.

secmsg is licensed under the [MIT License](LICENSE).

The `schema/` and `client/` packages are intentionally kept free of service-specific and cryptographic code so they can be imported by projects of any license. Daemons that communicate with this library over TCP are licensed independently.
