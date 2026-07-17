package client

import (
	"errors"
	"fmt"
	"testing"

	"github.com/tenebris-tech/secmsg/schema"
)

func TestRPCErrorCode(t *testing.T) {
	stealth := &rpcError{Code: schema.ErrCodeStealth, Message: "operation not permitted in stealth mode"}

	tests := []struct {
		name     string
		err      error
		wantCode int
		wantOK   bool
	}{
		{"direct rpc error", stealth, schema.ErrCodeStealth, true},
		{"wrapped rpc error", fmt.Errorf("send failed: %w", stealth), schema.ErrCodeStealth, true},
		{"other rpc code", &rpcError{Code: -32000}, -32000, true},
		{"plain error", errors.New("boom"), 0, false},
		{"nil error", nil, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, ok := RPCErrorCode(tt.err)
			if ok != tt.wantOK || code != tt.wantCode {
				t.Errorf("RPCErrorCode() = (%d, %v), want (%d, %v)", code, ok, tt.wantCode, tt.wantOK)
			}
		})
	}
}
