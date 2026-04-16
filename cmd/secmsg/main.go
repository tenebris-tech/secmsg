// Command secmsg is a CLI client for sigd, the signal daemon.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/mdp/qrterminal/v3"
	"github.com/tenebris-tech/mlogger"
	"github.com/tenebris-tech/secmsg/client"
	"github.com/tenebris-tech/secmsg/global"
	"github.com/tenebris-tech/secmsg/schema"
)

func main() {
	addr := flag.String("addr", client.DefaultAddr, "sigd address")
	asJSON := flag.Bool("json", false, "output as JSON")
	debug := flag.Bool("debug", false, "enable debug logging to /tmp/secmsg.log")
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		usage()
		os.Exit(1)
	}

	cmd := args[0]
	rest := args[1:]

	// Commands that don't need a persistent connection.
	switch cmd {
	case "help":
		usage()
		return
	case "version":
		fmt.Printf("%s %s\n", global.AppName, global.AppVersion)
		return
	}

	var log mlogger.Logger
	if *debug {
		var err error
		log, err = mlogger.New(
			mlogger.WithDebug(true),
			mlogger.WithLogFile(os.TempDir()+"/secmsg.log"),
			mlogger.WithLogStdout(false),
		)
		if err != nil {
			fatalf("init logger: %v", err)
		}
		defer log.Close()
	} else {
		log = mlogger.NewNullLogger()
	}

	c, err := client.Dial(*addr, client.WithLogger(log))
	if err != nil {
		fatalf("connect: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	switch cmd {
	case "send":
		if len(rest) < 3 {
			fatalf("usage: send <account> <to> <message>")
		}
		if err := c.SendMessage(ctx, schema.ServiceSignal, rest[0], rest[1], rest[2]); err != nil {
			fatalf("send: %v", err)
		}

	case "send-group":
		if len(rest) < 3 {
			fatalf("usage: send-group <account> <groupId> <message>")
		}
		if err := c.SendGroupMessage(ctx, schema.ServiceSignal, rest[0], rest[1], rest[2]); err != nil {
			fatalf("send-group: %v", err)
		}

	case "contacts":
		if len(rest) < 1 {
			fatalf("usage: contacts <account>")
		}
		contacts, err := c.Contacts(ctx, schema.ServiceSignal, rest[0])
		if err != nil {
			fatalf("contacts: %v", err)
		}
		printResult(*asJSON, contacts)

	case "groups":
		if len(rest) < 1 {
			fatalf("usage: groups <account>")
		}
		groups, err := c.Groups(ctx, schema.ServiceSignal, rest[0])
		if err != nil {
			fatalf("groups: %v", err)
		}
		printResult(*asJSON, groups)

	case "receipt-read":
		// receipt-read <account> <to> <timestamp> [<timestamp>...]
		if len(rest) < 3 {
			fatalf("usage: receipt-read <account> <to> <timestamp> [<timestamp>...]")
		}
		account := rest[0]
		to := rest[1]
		timestamps, err := parseTimestamps(rest[2:])
		if err != nil {
			fatalf("receipt-read: %v", err)
		}
		if err := c.SendReceiptRead(ctx, schema.ServiceSignal, account, to, timestamps); err != nil {
			fatalf("receipt-read: %v", err)
		}

	case "typing":
		if len(rest) < 3 {
			fatalf("usage: typing <account> <to> <true|false>")
		}
		typing, err := strconv.ParseBool(rest[2])
		if err != nil {
			fatalf("typing: invalid bool %q: %v", rest[2], err)
		}
		if err := c.SendTyping(ctx, schema.ServiceSignal, rest[0], rest[1], typing); err != nil {
			fatalf("typing: %v", err)
		}

	case "link":
		if len(rest) < 2 {
			fatalf("usage: link <account> <name>")
		}
		reply, err := c.LinkRequest(ctx, rest[0], rest[1])
		if err != nil {
			fatalf("link: %v", err)
		}
		if *asJSON {
			printJSON(reply)
		} else {
			fmt.Printf("status: %s\n", reply.Status)
			if reply.URI != "" {
				qrterminal.GenerateWithConfig(reply.URI, qrterminal.Config{
					Level:          qrterminal.L,
					Writer:         os.Stdout,
					HalfBlocks:     true,
					BlackChar:      "\033[97;40m \033[0m",
					WhiteBlackChar: "\033[97;40m▀\033[0m",
					WhiteChar:      "\033[97;40m█\033[0m",
					BlackWhiteChar: "\033[97;40m▄\033[0m",
					QuietZone:      1,
				})
				fmt.Println("Scan the QR code above, or use this URI:")
				fmt.Println(reply.URI)
				fmt.Println()
			}
		}
		if reply.Status == schema.LinkStatusPending {
			fmt.Println("Waiting for link to complete...")
			pollCtx, pollCancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer pollCancel()
			if err := pollLinkStatus(pollCtx, c, rest[0], *asJSON); err != nil {
				fatalf("link: %v", err)
			}
		}

	case "link-status":
		if len(rest) < 1 {
			fatalf("usage: link-status <account>")
		}
		reply, err := c.LinkStatus(ctx, rest[0])
		if err != nil {
			fatalf("link-status: %v", err)
		}
		if *asJSON {
			printJSON(reply)
		} else {
			fmt.Printf("status: %s\n", reply.Status)
			if reply.ACI != "" {
				fmt.Printf("aci: %s\n", reply.ACI)
			}
			if reply.Phone != "" {
				fmt.Printf("phone: %s\n", reply.Phone)
			}
			if reply.Error != "" {
				fmt.Printf("error: %s\n", reply.Error)
			}
		}

	case "poll-link":
		// poll-link <account> — polls link.status until complete or error.
		if len(rest) < 1 {
			fatalf("usage: poll-link <account>")
		}
		if err := pollLinkStatus(ctx, c, rest[0], *asJSON); err != nil {
			fatalf("poll-link: %v", err)
		}

	case "status":
		// status [account] — optional account name
		account := ""
		if len(rest) > 0 {
			account = rest[0]
		}
		if account != "" {
			result, err := c.Status(ctx, account)
			if err != nil {
				fatalf("status: %v", err)
			}
			if *asJSON {
				printJSON(result)
			} else {
				printStatusRow(*result)
			}
		} else {
			result, err := c.StatusAll(ctx)
			if err != nil {
				fatalf("status: %v", err)
			}
			if *asJSON {
				printJSON(result)
			} else {
				if len(result.Accounts) == 0 {
					fmt.Println("No accounts configured.")
				} else {
					for _, s := range result.Accounts {
						printStatusRow(s)
					}
				}
			}
		}

	case "unlink":
		if len(rest) < 1 {
			fatalf("usage: unlink <account>")
		}
		if err := c.Unlink(ctx, rest[0]); err != nil {
			fatalf("unlink: %v", err)
		}
		fmt.Println("Account unlinked.")

	case "receive":
		// receive [account] — poll for queued messages.
		account := ""
		if len(rest) > 0 {
			account = rest[0]
		}
		// Use a longer context for the long-poll receive call.
		recvCtx, recvCancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer recvCancel()
		messages, err := c.Receive(recvCtx, account, 30)
		if err != nil {
			fatalf("receive: %v", err)
		}
		for _, env := range messages {
			if *asJSON {
				printJSON(env)
			} else {
				printEnvelope(env)
			}
		}

	case "subscribe":
		// subscribe [account ...] — subscribe to notifications and print them.
		ch, cancel, err := c.Subscribe(ctx, rest...)
		if err != nil {
			fatalf("subscribe: %v", err)
		}
		defer cancel()
		fmt.Printf("%s %s — waiting for messages\n\n", global.AppName, global.AppVersion)
		for env := range ch {
			if *asJSON {
				printJSON(env)
			} else {
				printEnvelope(*env)
			}
		}
		fmt.Fprintln(os.Stderr, "Connection closed.")

	case "stealth-enable":
		// stealth-enable <account>
		if len(rest) < 1 {
			fatalf("usage: stealth-enable <account>")
		}
		result, err := c.StealthSet(ctx, rest[0], true)
		if err != nil {
			fatalf("stealth-enable: %v", err)
		}
		if *asJSON {
			printJSON(result)
		} else {
			fmt.Printf("stealth: %v\n", result.Stealth)
		}

	case "stealth-disable":
		// stealth-disable <account>
		if len(rest) < 1 {
			fatalf("usage: stealth-disable <account>")
		}
		result, err := c.StealthSet(ctx, rest[0], false)
		if err != nil {
			fatalf("stealth-disable: %v", err)
		}
		if *asJSON {
			printJSON(result)
		} else {
			fmt.Printf("stealth: %v\n", result.Stealth)
		}

	case "stealth-status":
		// stealth-status <account>
		if len(rest) < 1 {
			fatalf("usage: stealth-status <account>")
		}
		result, err := c.StealthStatus(ctx, rest[0])
		if err != nil {
			fatalf("stealth-status: %v", err)
		}
		if *asJSON {
			printJSON(result)
		} else {
			fmt.Printf("account: %s  global_stealth: %v  account_stealth: %v  active: %v\n",
				result.Account, result.GlobalStealth, result.AccountStealth, result.Active)
		}

	default:
		fatalf("unknown command %q (use -help for usage)", cmd)
	}
}

// printStatusRow renders one account row from a status reply.
func printStatusRow(s schema.StatusReply) {
	fmt.Printf("account: %s  linked: %v  connected: %v", s.Account, s.Linked, s.Connected)
	if aci := s.Identifiers["aci"]; aci != "" {
		fmt.Printf("  aci: %s", aci)
	}
	if phone := s.Identifiers["phone"]; phone != "" {
		fmt.Printf("  phone: %s", phone)
	}
	fmt.Println()
}

// printEnvelope renders a notification envelope to stdout with an optional
// account prefix. For message notifications the output includes sender and body;
// other types fall back to method=... params=... format.
func printEnvelope(env schema.Envelope) {
	// Try to extract the account field from the raw params for the prefix.
	var acct struct {
		Account string `json:"account"`
	}
	_ = json.Unmarshal(env.Params, &acct)
	prefix := ""
	if acct.Account != "" {
		prefix = "[" + acct.Account + "] "
	}

	switch env.Method {
	case schema.MethodMessage:
		var msg schema.MessageParams
		if err := json.Unmarshal(env.Params, &msg); err != nil {
			fmt.Printf("%smethod=%s params=%s\n", prefix, env.Method, env.Params)
			return
		}
		fmt.Printf("%s%s from:%s body:%s\n", prefix, msg.Type, msg.From.ID, msg.Body)
	case schema.MethodReceipt:
		var r schema.ReceiptParams
		if err := json.Unmarshal(env.Params, &r); err != nil {
			fmt.Printf("%smethod=%s params=%s\n", prefix, env.Method, env.Params)
			return
		}
		fmt.Printf("%sreceipt %s from:%s\n", prefix, r.Type, r.From.ID)
	case schema.MethodTyping:
		var t schema.TypingParams
		if err := json.Unmarshal(env.Params, &t); err != nil {
			fmt.Printf("%smethod=%s params=%s\n", prefix, env.Method, env.Params)
			return
		}
		fmt.Printf("%styping %s from:%s\n", prefix, t.Action, t.From.ID)
	default:
		fmt.Printf("%smethod=%s params=%s\n", prefix, env.Method, env.Params)
	}
}

func pollLinkStatus(ctx context.Context, c *client.Client, account string, asJSON bool) error {
	for {
		reply, err := c.LinkStatus(ctx, account)
		if err != nil {
			if ctx.Err() != nil {
				fmt.Fprintln(os.Stderr, "Timed out waiting for link to complete.")
				return nil
			}
			return err
		}
		if asJSON {
			printJSON(reply)
		} else {
			fmt.Printf("status: %s", reply.Status)
			if reply.ACI != "" {
				fmt.Printf("  aci: %s", reply.ACI)
			}
			if reply.Phone != "" {
				fmt.Printf("  phone: %s", reply.Phone)
			}
			if reply.Error != "" {
				fmt.Printf("  error: %s", reply.Error)
			}
			fmt.Println()
		}
		switch reply.Status {
		case schema.LinkStatusComplete:
			fmt.Println("Link successful.")
			return nil
		case schema.LinkStatusError:
			fmt.Fprintln(os.Stderr, "Link failed.")
			return nil
		}

		select {
		case <-ctx.Done():
			fmt.Fprintln(os.Stderr, "Timed out waiting for link to complete.")
			return nil
		case <-time.After(2 * time.Second):
		}
	}
}

// parseTimestamps converts a slice of decimal strings to uint64 values.
func parseTimestamps(ss []string) ([]uint64, error) {
	out := make([]uint64, 0, len(ss))
	for _, s := range ss {
		v, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid timestamp %q: %w", s, err)
		}
		out = append(out, v)
	}
	return out, nil
}

// printResult prints val as JSON when requested, otherwise prints each
// schema.Party field on its own line for contacts/groups results.
func printResult(asJSON bool, val any) {
	if asJSON {
		printJSON(val)
		return
	}
	switch v := val.(type) {
	case []schema.Party:
		for _, p := range v {
			fmt.Printf("id: %s\n", p.ID)
			if p.Name != "" {
				fmt.Printf("name: %s\n", p.Name)
			}
			if p.Device != 0 {
				fmt.Printf("device: %d\n", p.Device)
			}
			if p.About != "" {
				fmt.Printf("about: %s\n", p.About)
			}
			if p.AboutEmoji != "" {
				fmt.Printf("about_emoji: %s\n", p.AboutEmoji)
			}
			fmt.Println()
		}
	default:
		fmt.Printf("%+v\n", v)
	}
}

// printJSON marshals v to indented JSON and writes it to stdout.
func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fatalf("json encode: %v", err)
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "secmsg: "+format+"\n", args...)
	os.Exit(1)
}

func usage() {
	fmt.Fprintf(os.Stderr, `%s %s — CLI client for sigd

Usage:
  secmsg [flags] <command> [args...]

Flags:
  -addr string   sigd address (default %q)
  -json          output as JSON
  -debug         enable debug logging to /tmp/secmsg.log

Commands:
  send           <account> <to> <message>
  send-group     <account> <groupId> <message>
  contacts       <account>
  groups         <account>
  receipt-read   <account> <to> <timestamp> [<timestamp>...]
  typing         <account> <to> <true|false>
  link           <account> <name>
  link-status    <account>
  poll-link      <account>
  status         [account]
  unlink         <account>
  receive        [account]
  subscribe      [account ...]
  stealth-enable  <account>
  stealth-disable <account>
  stealth-status  <account>
  version
  help
`, global.AppName, global.AppVersion, client.DefaultAddr)
}
