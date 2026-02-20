package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jpreagan/gsc-cli/internal/auth"
	"golang.org/x/oauth2"
)

func TestAuthListCommand_JSONDoesNotLeakTokenFieldsOrValues(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	store, err := auth.NewStore()
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	account := "default"
	accessTok := "ACCESS_TOKEN_SHOULD_NOT_LEAK"
	refreshTok := "REFRESH_TOKEN_SHOULD_NOT_LEAK"
	exp := time.Date(2026, 2, 14, 12, 34, 56, 0, time.UTC)

	if err := store.WriteClientCredentials(account, "id", "secret"); err != nil {
		t.Fatalf("WriteClientCredentials: %v", err)
	}
	if err := store.WriteToken(account, &oauth2.Token{
		AccessToken:  accessTok,
		RefreshToken: refreshTok,
		Expiry:       exp,
		TokenType:    "Bearer",
	}); err != nil {
		t.Fatalf("WriteToken: %v", err)
	}

	root := NewRootCmd()
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"--json", "--account", account, "auth", "list"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v (stderr: %s)", err, stderr.String())
	}

	got := stdout.String()
	if strings.Contains(got, accessTok) || strings.Contains(got, refreshTok) {
		t.Fatalf("stdout leaked token values: %s", got)
	}
	for _, bad := range []string{`"access_token"`, `"refresh_token"`, `"id_token"`} {
		if strings.Contains(got, bad) {
			t.Fatalf("stdout leaked token field %s: %s", bad, got)
		}
	}

	var payload struct {
		Accounts []struct {
			Name        string `json:"name"`
			HasClient   bool   `json:"has_client"`
			HasToken    bool   `json:"has_token"`
			TokenExpiry string `json:"token_expiry"`
		} `json:"accounts"`
	}
	if err := json.Unmarshal([]byte(got), &payload); err != nil {
		t.Fatalf("Unmarshal stdout JSON: %v (stdout: %s)", err, got)
	}
	if len(payload.Accounts) != 1 {
		t.Fatalf("accounts len=%d want 1 (stdout: %s)", len(payload.Accounts), got)
	}
	if payload.Accounts[0].Name != account {
		t.Fatalf("accounts[0].name=%q want %q", payload.Accounts[0].Name, account)
	}
	if !payload.Accounts[0].HasClient || !payload.Accounts[0].HasToken {
		t.Fatalf("accounts[0] has_client=%v has_token=%v want true/true", payload.Accounts[0].HasClient, payload.Accounts[0].HasToken)
	}
	if payload.Accounts[0].TokenExpiry == "" {
		t.Fatalf("accounts[0].token_expiry empty; want non-empty")
	}
}

func TestCarryRefreshTokenPreservesOldWhenNewEmpty(t *testing.T) {
	oldTok := &oauth2.Token{RefreshToken: "OLD_REFRESH"}
	newTok := &oauth2.Token{RefreshToken: ""}

	carryRefreshToken(newTok, oldTok)

	if newTok.RefreshToken != "OLD_REFRESH" {
		t.Fatalf("RefreshToken=%q want %q", newTok.RefreshToken, "OLD_REFRESH")
	}
}

func TestCarryRefreshTokenDoesNotOverrideNonEmptyNew(t *testing.T) {
	oldTok := &oauth2.Token{RefreshToken: "OLD_REFRESH"}
	newTok := &oauth2.Token{RefreshToken: "NEW_REFRESH"}

	carryRefreshToken(newTok, oldTok)

	if newTok.RefreshToken != "NEW_REFRESH" {
		t.Fatalf("RefreshToken=%q want %q", newTok.RefreshToken, "NEW_REFRESH")
	}
}
