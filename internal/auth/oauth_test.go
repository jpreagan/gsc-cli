package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestLoopbackRedirectURL_UsesIPv4Literal(t *testing.T) {
	got := loopbackRedirectURL(12345)
	want := "http://127.0.0.1:12345/callback"
	if got != want {
		t.Fatalf("loopbackRedirectURL() = %q, want %q", got, want)
	}
}

func TestRunInstalledAppFlow_NilStderrWriterDoesNotPanic(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	cfg := &oauth2.Config{
		ClientID: "id",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://example.com/auth",
			TokenURL: "https://example.com/token",
		},
	}

	_, err := RunInstalledAppFlow(ctx, cfg, true, false, nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestRunInstalledAppFlow_AuthURLRequestsOfflineAndConsentPrompt(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := &oauth2.Config{
		ClientID: "id",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://example.com/auth",
			TokenURL: "https://example.com/token",
		},
	}

	urlCh := make(chan string, 1)
	doneCh := make(chan error, 1)
	go func() {
		_, err := RunInstalledAppFlow(ctx, cfg, true, false, func(format string, args ...any) {
			msg := fmt.Sprintf(format, args...)
			lines := strings.Split(strings.TrimSpace(msg), "\n")
			if len(lines) > 0 {
				u := lines[len(lines)-1]
				select {
				case urlCh <- u:
				default:
				}
			}
		})
		doneCh <- err
	}()

	var authURL string
	select {
	case authURL = <-urlCh:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for auth URL")
	}

	cancel()
	select {
	case <-doneCh:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for flow to exit")
	}

	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("url.Parse: %v (url=%q)", err, authURL)
	}
	q := parsed.Query()
	if q.Get("access_type") != "offline" {
		t.Fatalf("access_type=%q want %q (url=%q)", q.Get("access_type"), "offline", authURL)
	}
	if q.Get("prompt") != "consent" {
		t.Fatalf("prompt=%q want %q (url=%q)", q.Get("prompt"), "consent", authURL)
	}
	ru := q.Get("redirect_uri")
	if !strings.HasPrefix(ru, "http://127.0.0.1:") || !strings.HasSuffix(ru, "/callback") {
		t.Fatalf("redirect_uri=%q want prefix %q and suffix %q (url=%q)", ru, "http://127.0.0.1:", "/callback", authURL)
	}
}

func TestRunInstalledAppFlow_InvalidStateDoesNotAbort(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"access_token":"x","token_type":"Bearer","refresh_token":"y","expires_in":3600}`)
	}))
	defer tokenSrv.Close()

	cfg := &oauth2.Config{
		ClientID:     "id",
		ClientSecret: "secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://example.com/auth",
			TokenURL: tokenSrv.URL,
		},
	}

	urlCh := make(chan string, 1)
	type done struct {
		tok *oauth2.Token
		err error
	}
	doneCh := make(chan done, 1)
	go func() {
		tok, err := RunInstalledAppFlow(ctx, cfg, true, false, func(format string, args ...any) {
			msg := fmt.Sprintf(format, args...)
			lines := strings.Split(strings.TrimSpace(msg), "\n")
			if len(lines) > 0 {
				u := lines[len(lines)-1]
				select {
				case urlCh <- u:
				default:
				}
			}
		})
		doneCh <- done{tok: tok, err: err}
	}()

	var authURL string
	select {
	case authURL = <-urlCh:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for auth URL")
	}

	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("url.Parse: %v (url=%q)", err, authURL)
	}
	q := parsed.Query()
	redirectURI := q.Get("redirect_uri")
	if redirectURI == "" {
		t.Fatalf("redirect_uri is empty (url=%q)", authURL)
	}
	state := q.Get("state")
	if state == "" {
		t.Fatalf("state is empty (url=%q)", authURL)
	}

	badCB := redirectURI + "?state=bad&code=fake"
	resp, err := http.Get(badCB)
	if err != nil {
		t.Fatalf("http.Get(%q): %v", badCB, err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d want %d", resp.StatusCode, http.StatusBadRequest)
	}

	select {
	case d := <-doneCh:
		t.Fatalf("flow aborted after bad state callback: tok=%v err=%v", d.tok, d.err)
	case <-time.After(200 * time.Millisecond):
	}

	goodCB := redirectURI + "?state=" + url.QueryEscape(state) + "&code=fake"
	resp, err = http.Get(goodCB)
	if err != nil {
		t.Fatalf("http.Get(%q): %v", goodCB, err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d want %d", resp.StatusCode, http.StatusOK)
	}

	select {
	case d := <-doneCh:
		if d.err != nil {
			t.Fatalf("expected nil error, got %v", d.err)
		}
		if d.tok == nil || d.tok.AccessToken == "" {
			t.Fatalf("expected token, got %+v", d.tok)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for flow to complete")
	}
}

func TestRunInstalledAppFlow_MissingCodeFailsFast(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := &oauth2.Config{
		ClientID: "id",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://example.com/auth",
			TokenURL: "https://example.com/token",
		},
	}

	urlCh := make(chan string, 1)
	doneCh := make(chan error, 1)
	go func() {
		_, err := RunInstalledAppFlow(ctx, cfg, true, false, func(format string, args ...any) {
			msg := fmt.Sprintf(format, args...)
			lines := strings.Split(strings.TrimSpace(msg), "\n")
			if len(lines) > 0 {
				u := lines[len(lines)-1]
				select {
				case urlCh <- u:
				default:
				}
			}
		})
		doneCh <- err
	}()

	var authURL string
	select {
	case authURL = <-urlCh:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for auth URL")
	}

	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("url.Parse: %v (url=%q)", err, authURL)
	}
	q := parsed.Query()
	redirectURI := q.Get("redirect_uri")
	if redirectURI == "" {
		t.Fatalf("redirect_uri is empty (url=%q)", authURL)
	}
	state := q.Get("state")
	if state == "" {
		t.Fatalf("state is empty (url=%q)", authURL)
	}

	cb := redirectURI + "?state=" + url.QueryEscape(state)
	resp, err := http.Get(cb)
	if err != nil {
		t.Fatalf("http.Get(%q): %v", cb, err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d want %d", resp.StatusCode, http.StatusBadRequest)
	}

	select {
	case err := <-doneCh:
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !strings.Contains(strings.ToLower(err.Error()), "code") {
			t.Fatalf("err=%q want mention of code", err.Error())
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for flow to fail fast")
	}
}

func TestNewAuthCodeRequest_AuthURLIncludesRedirectStateOfflineConsent(t *testing.T) {
	cfg := &oauth2.Config{
		ClientID: "id",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://example.com/auth",
			TokenURL: "https://example.com/token",
		},
	}

	req, err := NewAuthCodeRequest(cfg)
	if err != nil {
		t.Fatalf("NewAuthCodeRequest: %v", err)
	}
	if req.AuthURL == "" || req.RedirectURL == "" || req.State == "" {
		t.Fatalf("req=%+v; expected non-empty fields", req)
	}

	parsed, err := url.Parse(req.AuthURL)
	if err != nil {
		t.Fatalf("url.Parse: %v (url=%q)", err, req.AuthURL)
	}
	q := parsed.Query()
	if q.Get("access_type") != "offline" {
		t.Fatalf("access_type=%q want %q (url=%q)", q.Get("access_type"), "offline", req.AuthURL)
	}
	if q.Get("prompt") != "consent" {
		t.Fatalf("prompt=%q want %q (url=%q)", q.Get("prompt"), "consent", req.AuthURL)
	}
	if q.Get("redirect_uri") != req.RedirectURL {
		t.Fatalf("redirect_uri=%q want %q (url=%q)", q.Get("redirect_uri"), req.RedirectURL, req.AuthURL)
	}
	if q.Get("state") != req.State {
		t.Fatalf("state=%q want %q (url=%q)", q.Get("state"), req.State, req.AuthURL)
	}
}

func TestExchangeAuthCodeFromRedirectURL_ExchangesToken(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if got := r.Form.Get("code"); got != "fake" {
			http.Error(w, "bad code", http.StatusBadRequest)
			return
		}
		if got := r.Form.Get("redirect_uri"); got == "" {
			http.Error(w, "missing redirect_uri", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"access_token":"x","token_type":"Bearer","refresh_token":"y","expires_in":3600}`)
	}))
	defer tokenSrv.Close()

	cfg := &oauth2.Config{
		ClientID:     "id",
		ClientSecret: "secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://example.com/auth",
			TokenURL: tokenSrv.URL,
		},
	}

	req, err := NewAuthCodeRequest(cfg)
	if err != nil {
		t.Fatalf("NewAuthCodeRequest: %v", err)
	}
	pasted := req.RedirectURL + "?state=" + url.QueryEscape(req.State) + "&code=fake"

	tok, err := ExchangeAuthCodeFromRedirectURL(ctx, cfg, req.RedirectURL, req.State, pasted)
	if err != nil {
		t.Fatalf("ExchangeAuthCodeFromRedirectURL: %v", err)
	}
	if tok == nil || tok.AccessToken == "" || tok.RefreshToken == "" {
		t.Fatalf("expected token, got %+v", tok)
	}
}

func TestExchangeAuthCodeFromRedirectURL_StateMismatchFails(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := &oauth2.Config{
		ClientID: "id",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://example.com/auth",
			TokenURL: "https://example.com/token",
		},
	}
	req, err := NewAuthCodeRequest(cfg)
	if err != nil {
		t.Fatalf("NewAuthCodeRequest: %v", err)
	}
	pasted := req.RedirectURL + "?state=bad&code=fake"

	_, err = ExchangeAuthCodeFromRedirectURL(ctx, cfg, req.RedirectURL, req.State, pasted)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "state") {
		t.Fatalf("expected state error, got %v", err)
	}
}
