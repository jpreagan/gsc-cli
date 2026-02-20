package auth

import (
	"encoding/json"
	"testing"
)

func TestStore_WriteClientCredentials_RedirectURIsUseIPv4Literal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	store, err := NewStore()
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if err := store.WriteClientCredentials("default", "id", "secret"); err != nil {
		t.Fatalf("WriteClientCredentials: %v", err)
	}

	b, err := store.ReadClientJSON("default")
	if err != nil {
		t.Fatalf("ReadClientJSON: %v", err)
	}

	var decoded struct {
		Installed struct {
			RedirectURIs []string `json:"redirect_uris"`
		} `json:"installed"`
	}
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if len(decoded.Installed.RedirectURIs) != 1 || decoded.Installed.RedirectURIs[0] != "http://127.0.0.1" {
		t.Fatalf("redirect_uris=%v want [%q]", decoded.Installed.RedirectURIs, "http://127.0.0.1")
	}
}
