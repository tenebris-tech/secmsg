// Command secmsg is a CLI client for the sigd Signal daemon.
// It connects to sigd over TCP and exposes all RPC methods as subcommands.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/securityguy/secmsg/client"
	"github.com/securityguy/secmsg/schema"
)

const (
	defaultAddr    = "127.0.0.1:9905"
	defaultTimeout = 30
)

func main() {
	// Global flags.
	addr := flag.String("addr", defaultAddr, "sigd TCP address (host:port)")
	jsonOutput := flag.Bool("json", false, "output results as JSON")
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		usage()
		os.Exit(1)
	}

	cmd := args[0]
	cmdArgs := args[1:]

	switch cmd {
	case "link":
		runLink(*addr, *jsonOutput, cmdArgs)
	case "status":
		runStatus(*addr, *jsonOutput, cmdArgs)
	case "send":
		runSend(*addr, *jsonOutput, cmdArgs)
	case "receive":
		runReceive(*addr, *jsonOutput, cmdArgs)
	case "subscribe":
		runSubscribe(*addr, *jsonOutput, cmdArgs)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: secmsg [flags] <command> [args]

Flags:
  --addr string   sigd TCP address (default %q)
  -json           output results as JSON

Commands:
  link <name> <account>         link to Signal (displays QR URI)
  status [account]              show link/connection status
  send <account> <to> <body>    send a 1:1 message
  receive [timeout]             poll once for pending messages (default %ds)
  subscribe                     stream push notifications to stdout
`, defaultAddr, defaultTimeout)
}

// dial connects to sigd or exits with an error.
func dial(addr string) *client.Client {
	c, err := client.Dial(addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot connect to sigd at %s: %v\n", addr, err)
		os.Exit(1)
	}
	return c
}

// printJSON marshals v to JSON and writes it to stdout.
func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintf(os.Stderr, "error: encode JSON: %v\n", err)
		os.Exit(1)
	}
}

// ---------------------------------------------------------------------------
// link
// ---------------------------------------------------------------------------

// runLink starts device linking and prints the QR URI.
// Usage: link <name> <account>
func runLink(addr string, jsonOut bool, args []string) {
	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: secmsg link <name> <account>\n")
		os.Exit(1)
	}
	name, account := args[0], args[1]

	c := dial(addr)
	defer c.Close() //nolint:errcheck

	reply, err := c.Link(name, account)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if jsonOut {
		printJSON(reply)
		return
	}

	switch reply.Status {
	case schema.LinkStatusPending:
		fmt.Printf("Scan this URI with Signal on your phone:\n%s\n", reply.URI)
		fmt.Println("Polling for completion...")
		pollLinkStatus(c, jsonOut)
	case schema.LinkStatusComplete:
		fmt.Printf("Linked successfully.\n  Account: %s\n  ACI:     %s\n  Phone:   %s\n",
			reply.Account, reply.ACI, reply.Phone)
	case schema.LinkStatusError:
		fmt.Fprintf(os.Stderr, "link failed: %s\n", reply.Error)
		os.Exit(1)
	}
}

// pollLinkStatus polls link.status until complete or error.
func pollLinkStatus(c *client.Client, jsonOut bool) {
	for {
		reply, err := c.LinkStatus()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error polling link status: %v\n", err)
			os.Exit(1)
		}

		switch reply.Status {
		case schema.LinkStatusPending:
			// Continue polling.
		case schema.LinkStatusComplete:
			if jsonOut {
				printJSON(reply)
				return
			}
			fmt.Printf("Linked successfully.\n  Account: %s\n  ACI:     %s\n  Phone:   %s\n",
				reply.Account, reply.ACI, reply.Phone)
			return
		case schema.LinkStatusError:
			fmt.Fprintf(os.Stderr, "link failed: %s\n", reply.Error)
			os.Exit(1)
		}
	}
}

// ---------------------------------------------------------------------------
// status
// ---------------------------------------------------------------------------

// runStatus prints the current link/connection status.
// Usage: status [account]
func runStatus(addr string, jsonOut bool, args []string) {
	var account string
	if len(args) > 0 {
		account = args[0]
	}

	c := dial(addr)
	defer c.Close() //nolint:errcheck

	result, err := c.Status(account)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if jsonOut {
		printJSON(result)
		return
	}

	if !result.Linked {
		fmt.Println("Not linked.")
		return
	}
	fmt.Printf("Linked:    %v\nConnected: %v\nAccount:   %s\nACI:       %s\nPhone:     %s\n",
		result.Linked, result.Connected, result.Account, result.ACI, result.Phone)
}

// ---------------------------------------------------------------------------
// send
// ---------------------------------------------------------------------------

// runSend sends a 1:1 message.
// Usage: send <account> <to> <body>
func runSend(addr string, jsonOut bool, args []string) {
	if len(args) < 3 {
		fmt.Fprintf(os.Stderr, "usage: secmsg send <account> <to> <body>\n")
		os.Exit(1)
	}
	account, to, body := args[0], args[1], args[2]

	c := dial(addr)
	defer c.Close() //nolint:errcheck

	ts, err := c.Send(to, body, account)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if jsonOut {
		printJSON(map[string]any{"timestamp": ts})
		return
	}
	fmt.Printf("Sent. Timestamp: %d\n", ts)
}

// ---------------------------------------------------------------------------
// receive
// ---------------------------------------------------------------------------

// runReceive polls once for pending messages.
// Usage: receive [timeout]
func runReceive(addr string, jsonOut bool, args []string) {
	timeout := defaultTimeout
	if len(args) > 0 {
		n, err := strconv.Atoi(args[0])
		if err != nil || n <= 0 {
			fmt.Fprintf(os.Stderr, "error: invalid timeout %q\n", args[0])
			os.Exit(1)
		}
		timeout = n
	}

	c := dial(addr)
	defer c.Close() //nolint:errcheck

	msgs, err := c.Receive(timeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if jsonOut {
		printJSON(map[string]any{"messages": msgs})
		return
	}

	if len(msgs) == 0 {
		fmt.Println("No messages.")
		return
	}
	for _, m := range msgs {
		fmt.Printf("[%d] from=%s type=%s body=%s\n", m.Timestamp, m.From, m.Type, m.Body)
	}
}

// ---------------------------------------------------------------------------
// subscribe
// ---------------------------------------------------------------------------

// runSubscribe subscribes to push notifications and streams them to stdout.
// Usage: subscribe
func runSubscribe(addr string, jsonOut bool, _ []string) {
	c := dial(addr)
	defer c.Close() //nolint:errcheck

	ch, err := c.Subscribe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if !jsonOut {
		fmt.Println("Subscribed. Waiting for notifications (Ctrl+C to stop)...")
	}

	for env := range ch {
		if jsonOut {
			printJSON(env)
			continue
		}
		printEnvelope(env)
	}
}

// printEnvelope pretty-prints a notification envelope.
func printEnvelope(env *schema.Envelope) {
	switch env.Method {
	case schema.MethodMessage:
		var p schema.MessageParams
		if err := json.Unmarshal(env.Params, &p); err != nil {
			fmt.Printf("[message] (decode error: %v)\n", err)
			return
		}
		fmt.Printf("[message] from=%s to=%s type=%s ts=%d body=%q\n",
			p.From.ID, p.To.ID, p.Type, p.Timestamp, p.Body)

	case schema.MethodReceipt:
		var p schema.ReceiptParams
		if err := json.Unmarshal(env.Params, &p); err != nil {
			fmt.Printf("[receipt] (decode error: %v)\n", err)
			return
		}
		fmt.Printf("[receipt] from=%s type=%s\n", p.From.ID, p.Type)

	case schema.MethodTyping:
		var p schema.TypingParams
		if err := json.Unmarshal(env.Params, &p); err != nil {
			fmt.Printf("[typing] (decode error: %v)\n", err)
			return
		}
		fmt.Printf("[typing] from=%s action=%s\n", p.From.ID, p.Action)

	case schema.MethodStatus:
		var p schema.StatusParams
		if err := json.Unmarshal(env.Params, &p); err != nil {
			fmt.Printf("[status] (decode error: %v)\n", err)
			return
		}
		fmt.Printf("[status] connected=%v\n", p.Connected)

	case schema.MethodConversationCleared:
		var p schema.ConversationClearedParams
		if err := json.Unmarshal(env.Params, &p); err != nil {
			fmt.Printf("[conversation.cleared] (decode error: %v)\n", err)
			return
		}
		fmt.Printf("[conversation.cleared] peer=%s full=%v\n", p.Peer, p.IsFullDelete)

	default:
		// Unknown notification — print as raw JSON.
		fmt.Printf("[%s] %s\n", env.Method, env.Params)
	}
}
