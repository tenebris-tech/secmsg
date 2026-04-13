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

	"github.com/tenebris-tech/secmsg/client"
	"github.com/tenebris-tech/secmsg/global"
	"github.com/tenebris-tech/secmsg/schema"
)

func main() {
	addr := flag.String("addr", client.DefaultAddr, "sigd address")
	asJSON := flag.Bool("json", false, "output as JSON")
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
		fmt.Printf("%s %s\n", global.AppName, global.Version)
		return
	}

	c, err := client.Dial(*addr)
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
				fmt.Printf("uri: %s\n", reply.URI)
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

	case "listen":
		// listen — subscribe to notifications and print them.
		ch, cancel := c.Subscribe()
		defer cancel()
		for env := range ch {
			if *asJSON {
				printJSON(env)
			} else {
				fmt.Printf("method=%s params=%s\n", env.Method, env.Params)
			}
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

// pollLinkStatus polls link.status once per second until the link is complete
// or has errored.
func pollLinkStatus(ctx context.Context, c *client.Client, account string, asJSON bool) error {
	for {
		reply, err := c.LinkStatus(ctx, account)
		if err != nil {
			return err
		}
		if asJSON {
			printJSON(reply)
		} else {
			fmt.Printf("status: %s", reply.Status)
			if reply.URI != "" {
				fmt.Printf("  uri: %s", reply.URI)
			}
			fmt.Println()
		}
		switch reply.Status {
		case schema.LinkStatusComplete, schema.LinkStatusError:
			return nil
		}
		time.Sleep(time.Second)
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

// printResult prints val as JSON when requested, otherwise uses %+v.
func printResult(asJSON bool, val any) {
	if asJSON {
		printJSON(val)
		return
	}
	fmt.Printf("%+v\n", val)
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

Commands:
  send          <account> <to> <message>
  send-group    <account> <groupId> <message>
  contacts      <account>
  groups        <account>
  receipt-read  <account> <to> <timestamp> [<timestamp>...]
  typing        <account> <to> <true|false>
  link          <account> <name>
  link-status   <account>
  poll-link     <account>
  status        [account]
  unlink        <account>
  listen
  version
  help
`, global.AppName, global.Version, client.DefaultAddr)
}
