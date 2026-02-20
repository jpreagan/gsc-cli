package cli

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

func TestPrinter_PlainNoHeadersTabSeparated(t *testing.T) {
	var out bytes.Buffer
	p := NewPrinter(&out, false, true, false)

	rows := [][]string{
		{"a", "b"},
		{"c", "d"},
	}
	if got := p.Print([]string{"H1", "H2"}, rows, map[string]any{"x": 1}); got != nil {
		t.Fatalf("Print: %v", got)
	}

	want := "a\tb\nc\td\n"
	if out.String() != want {
		t.Fatalf("out=%q want %q", out.String(), want)
	}
}

func TestPrinter_TableHasHeaders(t *testing.T) {
	var out bytes.Buffer
	p := NewPrinter(&out, false, false, false)

	headers := []string{"H1", "H2"}
	rows := [][]string{{"a", "b"}}
	if got := p.Print(headers, rows, map[string]any{"x": 1}); got != nil {
		t.Fatalf("Print: %v", got)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected >=2 lines, got %d: %q", len(lines), out.String())
	}
	if !strings.Contains(lines[0], "H1") || !strings.Contains(lines[0], "H2") {
		t.Fatalf("header line missing headers: %q", lines[0])
	}
	if !strings.Contains(lines[1], "a") || !strings.Contains(lines[1], "b") {
		t.Fatalf("row line missing values: %q", lines[1])
	}
}

func TestPrinter_JSONIsValid(t *testing.T) {
	var out bytes.Buffer
	p := NewPrinter(&out, true, false, false)

	v := struct {
		A string `json:"a"`
		B int    `json:"b"`
	}{
		A: "<tag>",
		B: 1,
	}
	if got := p.Print(nil, nil, v); got != nil {
		t.Fatalf("Print: %v", got)
	}
	if !strings.HasSuffix(out.String(), "\n") {
		t.Fatalf("expected trailing newline, got %q", out.String())
	}

	var decoded map[string]any
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("Unmarshal: %v (%q)", err, out.String())
	}
}

func TestPrinter_TableStripsControlChars(t *testing.T) {
	var out bytes.Buffer
	p := NewPrinter(&out, false, false, false)

	headers := []string{"H\x07", "H2"}
	rows := [][]string{
		{"A\x00B\x1fC", "D\rE"},
	}
	if got := p.Print(headers, rows, map[string]any{"x": 1}); got != nil {
		t.Fatalf("Print: %v", got)
	}

	for i, b := range out.Bytes() {
		if b < 0x20 && b != '\t' && b != '\n' {
			t.Fatalf("unexpected control byte 0x%02x at %d in %q", b, i, out.String())
		}
	}
	if strings.Contains(out.String(), "A\x00B") || strings.Contains(out.String(), "\r") {
		t.Fatalf("output still contains control chars: %q", out.String())
	}
	if !strings.Contains(out.String(), "ABC") {
		t.Fatalf("expected sanitized cell value to contain %q; got %q", "ABC", out.String())
	}
}

func TestPrinter_TableColorMatchesUncoloredAlignment(t *testing.T) {
	headers := []string{"QUERY", "PAGE", "CLICKS", "IMPRESSIONS", "CTR", "POSITION"}
	rows := [][]string{
		{
			"best ai for technology questions",
			"https://jpreagan.com/p/is-ai-eating-your-coding-skills",
			"0",
			"1",
			"0",
			"1",
		},
	}

	var plain bytes.Buffer
	if err := NewPrinter(&plain, false, false, false).Print(headers, rows, nil); err != nil {
		t.Fatalf("plain Print: %v", err)
	}

	var colored bytes.Buffer
	if err := NewPrinter(&colored, false, false, true).Print(headers, rows, nil); err != nil {
		t.Fatalf("colored Print: %v", err)
	}

	ansi := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	stripped := ansi.ReplaceAllString(colored.String(), "")
	if stripped != plain.String() {
		t.Fatalf("color output changed table layout:\nplain:   %q\ncolored: %q", plain.String(), colored.String())
	}
}
