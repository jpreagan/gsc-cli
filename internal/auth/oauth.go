package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/searchconsole/v1"
)

const (
	ScopeReadWrite = "https://www.googleapis.com/auth/webmasters"
	ScopeReadOnly  = "https://www.googleapis.com/auth/webmasters.readonly"
)

func OAuthConfigFromClientJSON(clientJSON []byte, redirectURL string, readonly bool) (*oauth2.Config, error) {
	scope := ScopeReadWrite
	if readonly {
		scope = ScopeReadOnly
	}
	cfg, err := google.ConfigFromJSON(clientJSON, scope)
	if err != nil {
		return nil, err
	}
	if redirectURL != "" {
		cfg.RedirectURL = redirectURL
	}
	return cfg, nil
}

func NewSearchConsoleService(ctx context.Context, store *Store, account string) (*searchconsole.Service, error) {
	clientJSON, err := store.ReadClientJSON(account)
	if err != nil {
		return nil, err
	}
	tok, err := store.ReadToken(account)
	if err != nil {
		return nil, err
	}
	cfg, err := OAuthConfigFromClientJSON(clientJSON, "", false)
	if err != nil {
		return nil, err
	}
	ts := cfg.TokenSource(ctx, tok)
	return searchconsole.NewService(ctx, option.WithTokenSource(ts))
}

func RunInstalledAppFlow(ctx context.Context, cfg *oauth2.Config, noInput bool, openBrowser bool, stderrWriter func(format string, args ...any)) (*oauth2.Token, error) {
	if cfg == nil {
		return nil, errors.New("oauth config is nil")
	}

	cfgCopy := *cfg
	cfg = &cfgCopy

	if stderrWriter == nil {
		stderrWriter = func(string, ...any) {}
	}

	flowCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port
	redirectURL := loopbackRedirectURL(port)
	cfg.RedirectURL = redirectURL

	state, err := randomState(32)
	if err != nil {
		return nil, err
	}

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	var once sync.Once

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("state") != state {
			stderrWriter("Ignoring OAuth callback with invalid state parameter.\n")
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			return
		}
		if oauthErr := q.Get("error"); oauthErr != "" {
			http.Error(w, "Authorization error: "+oauthErr, http.StatusBadRequest)
			once.Do(func() { errCh <- fmt.Errorf("authorization error: %s", oauthErr) })
			return
		}
		code := q.Get("code")
		if code == "" {
			http.Error(w, "Missing code", http.StatusBadRequest)
			once.Do(func() { errCh <- fmt.Errorf("missing oauth code parameter") })
			return
		}
		if _, err := w.Write([]byte("Authorization received. You can close this tab.\n")); err != nil {
			_ = err
		}
		once.Do(func() { codeCh <- code })
	})

	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			once.Do(func() { errCh <- err })
		}
	}()

	authURL := cfg.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "consent"),
	)
	if noInput || !openBrowser {
		stderrWriter("Open this URL in a browser to continue:\n%s\n", authURL)
	} else {
		if err := OpenBrowser(authURL); err != nil {
			stderrWriter("Failed to open browser; open this URL manually:\n%s\n", authURL)
		}
	}

	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return nil, err
	case <-flowCtx.Done():
		return nil, flowCtx.Err()
	}

	tok, err := cfg.Exchange(flowCtx, code)
	if err != nil {
		return nil, err
	}
	return tok, nil
}

func loopbackRedirectURL(port int) string {
	return fmt.Sprintf("http://127.0.0.1:%d/callback", port)
}

type AuthCodeRequest struct {
	AuthURL     string
	RedirectURL string
	State       string
}

func NewAuthCodeRequest(cfg *oauth2.Config) (*AuthCodeRequest, error) {
	if cfg == nil {
		return nil, errors.New("oauth config is nil")
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()

	redirectURL := loopbackRedirectURL(port)
	state, err := randomState(32)
	if err != nil {
		return nil, err
	}

	cfgCopy := *cfg
	cfgCopy.RedirectURL = redirectURL
	authURL := cfgCopy.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "consent"),
	)
	return &AuthCodeRequest{AuthURL: authURL, RedirectURL: redirectURL, State: state}, nil
}

func ExchangeAuthCodeFromRedirectURL(ctx context.Context, cfg *oauth2.Config, expectedRedirectURL, expectedState, redirectWithCode string) (*oauth2.Token, error) {
	if cfg == nil {
		return nil, errors.New("oauth config is nil")
	}
	code, state, base, err := parseOAuthRedirectURL(redirectWithCode)
	if err != nil {
		return nil, err
	}
	if state != expectedState {
		return nil, errors.New("invalid state parameter")
	}
	if expectedRedirectURL != "" && base != expectedRedirectURL {
		return nil, fmt.Errorf("redirect URL mismatch (got %q, want %q)", base, expectedRedirectURL)
	}

	cfgCopy := *cfg
	if expectedRedirectURL != "" {
		cfgCopy.RedirectURL = expectedRedirectURL
	} else {
		cfgCopy.RedirectURL = base
	}
	return cfgCopy.Exchange(ctx, code)
}

func parseOAuthRedirectURL(raw string) (code string, state string, base string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", "", errors.New("missing redirect URL")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", "", "", err
	}
	if u.Scheme == "" || u.Host == "" {
		return "", "", "", errors.New("redirect URL must be a full URL (e.g. http://127.0.0.1:12345/callback?code=...&state=...)")
	}

	q := u.Query()
	if oauthErr := q.Get("error"); oauthErr != "" {
		return "", "", "", fmt.Errorf("authorization error: %s", oauthErr)
	}
	code = q.Get("code")
	if code == "" {
		return "", "", "", errors.New("missing oauth code parameter")
	}
	state = q.Get("state")
	if state == "" {
		return "", "", "", errors.New("missing state parameter")
	}

	uu := *u
	uu.RawQuery = ""
	uu.Fragment = ""
	base = uu.String()
	return code, state, base, nil
}

func OpenBrowser(url string) error {
	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		cmd = exec.Command("open", url)
	} else {
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

func randomState(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
