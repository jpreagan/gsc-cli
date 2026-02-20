package cmd

import (
	"testing"
	"time"
)

func TestDefaultAnalyticsDateRange_PacificTime(t *testing.T) {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatalf("LoadLocation: %v", err)
	}

	tests := []struct {
		name      string
		nowUTC    time.Time
		wantStart string
		wantEnd   string
	}{
		{
			name:      "UTC time still previous day in PT",
			nowUTC:    time.Date(2026, 2, 14, 7, 0, 0, 0, time.UTC),
			wantStart: "2026-01-15",
			wantEnd:   "2026-02-11",
		},
		{
			name:      "UTC time same day in PT",
			nowUTC:    time.Date(2026, 2, 14, 12, 0, 0, 0, time.UTC),
			wantStart: "2026-01-16",
			wantEnd:   "2026-02-12",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStart, gotEnd := defaultAnalyticsDateRange(tt.nowUTC, loc)
			if gotStart != tt.wantStart || gotEnd != tt.wantEnd {
				t.Fatalf("defaultAnalyticsDateRange() = (%q, %q), want (%q, %q)", gotStart, gotEnd, tt.wantStart, tt.wantEnd)
			}
			start, err := parseDate(gotStart)
			if err != nil {
				t.Fatalf("parseDate(start): %v", err)
			}
			end, err := parseDate(gotEnd)
			if err != nil {
				t.Fatalf("parseDate(end): %v", err)
			}
			if got := int(end.Sub(start).Hours()/24) + 1; got != 28 {
				t.Fatalf("inclusive range length=%d days, want 28", got)
			}
		})
	}
}

func TestParseFilters(t *testing.T) {
	tests := []struct {
		name        string
		in          []string
		wantFilters int
		wantErr     bool
	}{
		{name: "empty", in: nil, wantFilters: 0, wantErr: false},
		{name: "one", in: []string{"query=equals=foo"}, wantFilters: 1, wantErr: false},
		{name: "trim", in: []string{" query = contains = foo bar "}, wantFilters: 1, wantErr: false},
		{name: "two", in: []string{"query=equals=foo", "page=includingRegex=/blog/.*"}, wantFilters: 2, wantErr: false},
		{name: "skip empty items", in: []string{"", "query=equals=foo", " "}, wantFilters: 1, wantErr: false},
		{name: "bad format", in: []string{"query=equals"}, wantFilters: 0, wantErr: true},
		{name: "bad operator", in: []string{"query=wat=foo"}, wantFilters: 0, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, err := parseFilters(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseFilters() err=%v wantErr=%v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if tt.wantFilters == 0 {
				if g != nil {
					t.Fatalf("parseFilters() group=%v want nil", g)
				}
				return
			}
			if g == nil {
				t.Fatalf("parseFilters() group=nil want non-nil")
			}
			if g.GroupType != "and" {
				t.Fatalf("group.GroupType=%q want %q", g.GroupType, "and")
			}
			if len(g.Filters) != tt.wantFilters {
				t.Fatalf("len(group.Filters)=%d want %d", len(g.Filters), tt.wantFilters)
			}
		})
	}
}
