package messageref

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantChannel string
		wantTS      string
		wantErr     bool
	}{
		{"api form", "C02DF3BEUGN/1777469221.721439", "C02DF3BEUGN", "1777469221.721439", false},
		{"private channel api form", "G01234ABCDE/1234567890.123456", "G01234ABCDE", "1234567890.123456", false},
		{"DM api form", "D01234ABCDE/1234567890.123456", "D01234ABCDE", "1234567890.123456", false},
		{"DM permalink", "https://example.slack.com/archives/D01234ABCDE/p1234567890123456", "D01234ABCDE", "1234567890.123456", false},
		{"p-prefixed ts", "C02DF3BEUGN/p1777469221721439", "C02DF3BEUGN", "1777469221.721439", false},
		{"permalink", "https://example.slack.com/archives/C02DF3BEUGN/p1777469221721439", "C02DF3BEUGN", "1777469221.721439", false},
		{"permalink with thread_ts query", "https://example.slack.com/archives/C02DF3BEUGN/p1777469221721439?thread_ts=1777469200.000000&cid=C02DF3BEUGN", "C02DF3BEUGN", "1777469221.721439", false},
		{"trims whitespace", "  C02DF3BEUGN/1777469221.721439  ", "C02DF3BEUGN", "1777469221.721439", false},

		{"empty", "", "", "", true},
		{"missing slash", "C02DF3BEUGN", "", "", true},
		{"empty channel", "/1234567890.123456", "", "", true},
		{"empty ts", "C02DF3BEUGN/", "", "", true},
		{"invalid channel", "X02DF3BEUGN/1234567890.123456", "", "", true},
		{"user id as channel", "U02DF3BEUGN/1234567890.123456", "", "", true},
		{"invalid ts", "C02DF3BEUGN/not-a-ts", "", "", true},
		{"non-permalink http", "https://example.com/foo", "", "", true},
		{"permalink missing p prefix", "https://example.slack.com/archives/C123/1234567890123456", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse(%q) err = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got.ChannelID != tt.wantChannel || got.TS != tt.wantTS {
				t.Errorf("Parse(%q) = %+v, want {%s %s}", tt.input, got, tt.wantChannel, tt.wantTS)
			}
		})
	}
}

func TestRefString(t *testing.T) {
	r := Ref{ChannelID: "C02DF3BEUGN", TS: "1777469221.721439"}
	want := "C02DF3BEUGN/1777469221.721439"
	if got := r.String(); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestParseRoundTrip(t *testing.T) {
	inputs := []string{
		"C02DF3BEUGN/1777469221.721439",
		"https://example.slack.com/archives/C02DF3BEUGN/p1777469221721439",
		"C02DF3BEUGN/p1777469221721439",
	}
	for _, in := range inputs {
		r, err := Parse(in)
		if err != nil {
			t.Fatalf("Parse(%q) failed: %v", in, err)
		}
		r2, err := Parse(r.String())
		if err != nil {
			t.Fatalf("Parse(%q.String() = %q) failed: %v", in, r.String(), err)
		}
		if r != r2 {
			t.Errorf("round-trip mismatch: %+v -> %+v", r, r2)
		}
	}
}
