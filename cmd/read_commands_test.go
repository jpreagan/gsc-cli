package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jpreagan/gsc-cli/internal/auth"
	"google.golang.org/api/option"
	"google.golang.org/api/searchconsole/v1"
)

func TestSitesListCommand_JSON(t *testing.T) {
	withMockSearchConsoleService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.EscapedPath() != "/webmasters/v3/sites" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"siteEntry":[{"siteUrl":"sc-domain:jpreagan.com","permissionLevel":"siteOwner"}]}`)
	})

	stdout, stderr, err := executeRoot(t, "--json", "--account", "personal", "sites")
	if err != nil {
		t.Fatalf("Execute: %v (stderr: %s)", err, stderr)
	}

	var out struct {
		Sites []struct {
			SiteURL         string `json:"site_url"`
			PermissionLevel string `json:"permission_level"`
		} `json:"sites"`
	}
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("Unmarshal stdout JSON: %v (stdout: %s)", err, stdout)
	}
	if len(out.Sites) != 1 {
		t.Fatalf("len(out.Sites)=%d want 1 (stdout: %s)", len(out.Sites), stdout)
	}
	if out.Sites[0].SiteURL != "sc-domain:jpreagan.com" || out.Sites[0].PermissionLevel != "siteOwner" {
		t.Fatalf("sites[0]=%+v want site_url=%q permission_level=%q", out.Sites[0], "sc-domain:jpreagan.com", "siteOwner")
	}
}

func TestSitesGetCommand_JSON(t *testing.T) {
	withMockSearchConsoleService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.HasPrefix(r.URL.EscapedPath(), "/webmasters/v3/sites/") {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"siteUrl":"sc-domain:jpreagan.com","permissionLevel":"siteFullUser"}`)
	})

	stdout, stderr, err := executeRoot(t, "--json", "--account", "personal", "sites", "get", "sc-domain:jpreagan.com")
	if err != nil {
		t.Fatalf("Execute: %v (stderr: %s)", err, stderr)
	}

	var out struct {
		SiteURL         string `json:"site_url"`
		PermissionLevel string `json:"permission_level"`
	}
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("Unmarshal stdout JSON: %v (stdout: %s)", err, stdout)
	}
	if out.SiteURL != "sc-domain:jpreagan.com" || out.PermissionLevel != "siteFullUser" {
		t.Fatalf("out=%+v want site_url=%q permission_level=%q", out, "sc-domain:jpreagan.com", "siteFullUser")
	}
}

func TestSitesListCommand_PropagatesAPIError(t *testing.T) {
	withMockSearchConsoleService(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"code":403,"message":"forbidden"}}`, http.StatusForbidden)
	})

	_, _, err := executeRoot(t, "--account", "personal", "sites")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestSitemapsListCommand_JSON(t *testing.T) {
	withMockSearchConsoleService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.HasSuffix(r.URL.EscapedPath(), "/sitemaps") {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"sitemap":[{"path":"https://jpreagan.com/sitemap.xml","type":"sitemap","lastSubmitted":"2026-02-11","lastDownloaded":"2026-02-12","warnings":"1","errors":"2","isPending":false}]}`)
	})

	stdout, stderr, err := executeRoot(t, "--json", "--account", "personal", "sitemaps", "--site", "sc-domain:jpreagan.com")
	if err != nil {
		t.Fatalf("Execute: %v (stderr: %s)", err, stderr)
	}

	var out struct {
		Site     string `json:"site"`
		Sitemaps []struct {
			Path           string `json:"path"`
			Type           string `json:"type"`
			LastSubmitted  string `json:"last_submitted"`
			LastDownloaded string `json:"last_downloaded"`
			Warnings       int64  `json:"warnings"`
			Errors         int64  `json:"errors"`
			IsPending      bool   `json:"is_pending"`
		} `json:"sitemaps"`
	}
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("Unmarshal stdout JSON: %v (stdout: %s)", err, stdout)
	}
	if out.Site != "sc-domain:jpreagan.com" {
		t.Fatalf("site=%q want %q", out.Site, "sc-domain:jpreagan.com")
	}
	if len(out.Sitemaps) != 1 {
		t.Fatalf("len(out.Sitemaps)=%d want 1 (stdout: %s)", len(out.Sitemaps), stdout)
	}
	if out.Sitemaps[0].Path != "https://jpreagan.com/sitemap.xml" || out.Sitemaps[0].Errors != 2 {
		t.Fatalf("sitemaps[0]=%+v unexpected", out.Sitemaps[0])
	}
}

func TestSitemapsGetCommand_JSON(t *testing.T) {
	withMockSearchConsoleService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.Contains(r.URL.EscapedPath(), "/sitemaps/") {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"path":"https://jpreagan.com/sitemap.xml","type":"sitemap","lastSubmitted":"2026-02-11","lastDownloaded":"2026-02-12","warnings":"0","errors":"0","isPending":false}`)
	})

	stdout, stderr, err := executeRoot(t, "--json", "--account", "personal", "sitemaps", "--site", "sc-domain:jpreagan.com", "get", "https://jpreagan.com/sitemap.xml")
	if err != nil {
		t.Fatalf("Execute: %v (stderr: %s)", err, stderr)
	}

	var out struct {
		Site string `json:"site"`
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("Unmarshal stdout JSON: %v (stdout: %s)", err, stdout)
	}
	if out.Site != "sc-domain:jpreagan.com" || out.Path != "https://jpreagan.com/sitemap.xml" {
		t.Fatalf("out=%+v unexpected", out)
	}
}

func TestInspectCommand_JSON_ConvertsLastCrawlTimeToUTC(t *testing.T) {
	withMockSearchConsoleService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.EscapedPath() != "/v1/urlInspection/index:inspect" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		var req struct {
			InspectionURL string `json:"inspectionUrl"`
			SiteURL       string `json:"siteUrl"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		if req.SiteURL != "sc-domain:jpreagan.com" || req.InspectionURL != "https://jpreagan.com/p/is-ai-eating-your-coding-skills" {
			http.Error(w, "unexpected request payload", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"inspectionResult":{"indexStatusResult":{"verdict":"PASS","coverageState":"Submitted and indexed","indexingState":"INDEXING_ALLOWED","robotsTxtState":"ALLOWED","lastCrawlTime":"2026-02-11T01:02:03-08:00"}}}`)
	})

	stdout, stderr, err := executeRoot(
		t,
		"--json",
		"--account", "personal",
		"inspect",
		"--site", "sc-domain:jpreagan.com",
		"--url", "https://jpreagan.com/p/is-ai-eating-your-coding-skills",
	)
	if err != nil {
		t.Fatalf("Execute: %v (stderr: %s)", err, stderr)
	}

	var out struct {
		Site        string `json:"site"`
		URL         string `json:"url"`
		Verdict     string `json:"verdict"`
		LastCrawlAt string `json:"last_crawl_time"`
	}
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("Unmarshal stdout JSON: %v (stdout: %s)", err, stdout)
	}
	if out.Site != "sc-domain:jpreagan.com" || out.URL != "https://jpreagan.com/p/is-ai-eating-your-coding-skills" || out.Verdict != "PASS" {
		t.Fatalf("out=%+v unexpected", out)
	}
	if out.LastCrawlAt != "2026-02-11T09:02:03Z" {
		t.Fatalf("last_crawl_time=%q want %q", out.LastCrawlAt, "2026-02-11T09:02:03Z")
	}
}

func TestInspectCommand_JSON_PreservesInvalidLastCrawlTime(t *testing.T) {
	withMockSearchConsoleService(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"inspectionResult":{"indexStatusResult":{"lastCrawlTime":"not-a-time"}}}`)
	})

	stdout, stderr, err := executeRoot(
		t,
		"--json",
		"--account", "personal",
		"inspect",
		"--site", "sc-domain:jpreagan.com",
		"--url", "https://jpreagan.com/p/is-ai-eating-your-coding-skills",
	)
	if err != nil {
		t.Fatalf("Execute: %v (stderr: %s)", err, stderr)
	}

	var out struct {
		LastCrawlAt string `json:"last_crawl_time"`
	}
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("Unmarshal stdout JSON: %v (stdout: %s)", err, stdout)
	}
	if out.LastCrawlAt != "not-a-time" {
		t.Fatalf("last_crawl_time=%q want %q", out.LastCrawlAt, "not-a-time")
	}
}

func withMockSearchConsoleService(t *testing.T, handler http.HandlerFunc) {
	t.Helper()

	srv := httptest.NewTLSServer(handler)
	t.Cleanup(srv.Close)

	svc, err := searchconsole.NewService(
		context.Background(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
		option.WithoutAuthentication(),
	)
	if err != nil {
		t.Fatalf("searchconsole.NewService: %v", err)
	}

	old := newSearchConsoleService
	newSearchConsoleService = func(_ context.Context, _ *auth.Store, _ string) (*searchconsole.Service, error) {
		return svc, nil
	}
	t.Cleanup(func() {
		newSearchConsoleService = old
	})
}

func executeRoot(t *testing.T, args ...string) (stdout string, stderr string, err error) {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)

	root := NewRootCmd()
	var out bytes.Buffer
	var errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetArgs(args)

	err = root.Execute()
	if errors.Is(err, ErrVersionRequested) {
		err = nil
	}
	return out.String(), errOut.String(), err
}
