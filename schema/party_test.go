package schema

import "testing"

func TestPartyFormat(t *testing.T) {
	tests := []struct {
		p    Party
		want string
	}{
		{Party{ID: "abc123", Name: "Alice"}, "abc123[]Alice"},
		{Party{ID: "abc123", Name: "Alice", Device: 2}, "abc123[2]Alice"},
		{Party{ID: "c8916876-9a5a-409e-842a-df57fbb468b8"}, "c8916876-9a5a-409e-842a-df57fbb468b8[]"},
		{Party{ID: "c8916876-9a5a-409e-842a-df57fbb468b8", Device: 1}, "c8916876-9a5a-409e-842a-df57fbb468b8[1]"},
		{Party{ID: "15a636b9-1c74-4305-bb62-f92c14bbb0a5", Name: "Eric", Device: 3}, "15a636b9-1c74-4305-bb62-f92c14bbb0a5[3]Eric"},
		{Party{ID: "d0944716-81b3-4d85-8e9d-b4ca041220da", Device: 1}, "d0944716-81b3-4d85-8e9d-b4ca041220da[1]"},
		{Party{Name: "Eric"}, "[]Eric"},
		{Party{Name: "Eric", Device: 4}, "[4]Eric"},
		{Party{}, ""},
		{Party{Self: true}, "me"},
		{Party{Self: true, Device: 3}, "me[3]"},
		{Party{Self: true, ID: "15a636b9-1c74-4305-bb62-f92c14bbb0a5", Device: 2}, "me[2]"},
	}
	for _, tc := range tests {
		got := tc.p.Format()
		if got != tc.want {
			t.Errorf("Party%+v.Format() = %q, want %q", tc.p, got, tc.want)
		}
	}
}
