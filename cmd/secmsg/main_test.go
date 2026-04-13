package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/tenebris-tech/secmsg/global"
	"github.com/tenebris-tech/secmsg/schema"
)

// TestHelperProcess is not a real test. When GO_WANT_HELPER_PROCESS=1 is set
// in the environment, it acts as the secmsg binary so that other tests can
// invoke CLI commands via exec.Command without requiring a pre-built binary.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	var args []string
	if err := json.Unmarshal([]byte(os.Getenv("HELPER_ARGS")), &args); err != nil {
		fmt.Fprintln(os.Stderr, "secmsg test: parse args:", err)
		os.Exit(2)
	}
	os.Args = append([]string{"secmsg"}, args...)
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	main()
	os.Exit(0)
}

// helperCmd returns an exec.Cmd that runs the current test binary acting as
// the secmsg binary with the given arguments.
func helperCmd(t *testing.T, args ...string) *exec.Cmd {
	t.Helper()
	argsJSON, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("helperCmd marshal: %v", err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestHelperProcess$")
	cmd.Env = append(os.Environ(),
		"GO_WANT_HELPER_PROCESS=1",
		"HELPER_ARGS="+string(argsJSON),
	)
	return cmd
}

// mockReq is the minimal JSON-RPC 2.0 request shape needed by the test server.
type mockReq struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	ID      uint64          `json:"id"`
	Params  json.RawMessage `json:"params"`
}

// mockResp is the JSON-RPC 2.0 response shape written by the test server.
type mockResp struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      uint64          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *mockRPCErr     `json:"error,omitempty"`
}

type mockRPCErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// startMockServer starts a minimal sigd stub and returns its address. The
// handler is called for each incoming RPC request and returns the full response.
// The server accepts exactly one connection.
func startMockServer(t *testing.T, handler func(req mockReq) mockResp) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		// Send the hello notification that the client expects on connect.
		fmt.Fprintf(conn, `{"jsonrpc":"2.0","method":"%s"}`+"\n", schema.MethodHello)
		r := bufio.NewReader(conn)
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			var req mockReq
			if err := json.Unmarshal([]byte(strings.TrimRight(line, "\n")), &req); err != nil {
				continue
			}
			resp := handler(req)
			data, _ := json.Marshal(resp)
			fmt.Fprintf(conn, "%s\n", data)
		}
	}()

	return ln.Addr().String()
}

// okResult returns a mockResp with a JSON null result (success, no payload).
func okResult(id uint64) mockResp {
	return mockResp{JSONRPC: "2.0", ID: id, Result: json.RawMessage(`null`)}
}

// errResult returns a mockResp carrying an RPC error.
func errResult(id uint64, code int, msg string) mockResp {
	return mockResp{JSONRPC: "2.0", ID: id, Error: &mockRPCErr{Code: code, Message: msg}}
}

// --- version ---

func TestVersion(t *testing.T) {
	cmd := helperCmd(t, "version")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("version: %v", err)
	}
	want := global.AppName + " " + global.AppVersion
	if !strings.Contains(string(out), want) {
		t.Errorf("version: got %q, want output to contain %q", string(out), want)
	}
}

// --- status ---

func TestStatusEmpty(t *testing.T) {
	addr := startMockServer(t, func(req mockReq) mockResp {
		result, _ := json.Marshal(schema.StatusAllReply{Accounts: []schema.StatusReply{}})
		return mockResp{JSONRPC: "2.0", ID: req.ID, Result: result}
	})

	cmd := helperCmd(t, "-addr", addr, "status")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("status empty: %v (output: %s)", err, out)
	}
	if !strings.Contains(string(out), "No accounts configured.") {
		t.Errorf("status empty: got %q, want 'No accounts configured.'", string(out))
	}
}

func TestStatusWithAccounts(t *testing.T) {
	accounts := []schema.StatusReply{
		{
			Account:     "alice",
			Linked:      true,
			Connected:   true,
			Identifiers: map[string]string{"phone": "+15550001111"},
		},
		{
			Account:   "bob",
			Linked:    false,
			Connected: false,
		},
	}
	addr := startMockServer(t, func(req mockReq) mockResp {
		result, _ := json.Marshal(schema.StatusAllReply{Accounts: accounts})
		return mockResp{JSONRPC: "2.0", ID: req.ID, Result: result}
	})

	cmd := helperCmd(t, "-addr", addr, "status")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("status accounts: %v (output: %s)", err, out)
	}
	output := string(out)
	for _, want := range []string{"alice", "bob", "+15550001111"} {
		if !strings.Contains(output, want) {
			t.Errorf("status accounts: output %q missing %q", output, want)
		}
	}
}

// --- unlink ---

func TestUnlinkSuccess(t *testing.T) {
	addr := startMockServer(t, func(req mockReq) mockResp {
		return okResult(req.ID)
	})

	cmd := helperCmd(t, "-addr", addr, "unlink", "myaccount")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("unlink success: %v (output: %s)", err, out)
	}
	if !strings.Contains(string(out), "Account unlinked.") {
		t.Errorf("unlink success: got %q, want 'Account unlinked.'", string(out))
	}
}

func TestUnlinkError(t *testing.T) {
	addr := startMockServer(t, func(req mockReq) mockResp {
		return errResult(req.ID, -32600, "account not found")
	})

	cmd := helperCmd(t, "-addr", addr, "unlink", "myaccount")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("unlink error: expected non-zero exit; output: %q", out)
	}
	combined := string(out)
	if !strings.Contains(combined, "account not found") {
		t.Errorf("unlink error: output %q does not contain error message", combined)
	}
}
