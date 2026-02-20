package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	"golang.org/x/oauth2"
)

var accountNameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)

type Store struct {
	BaseDir string
}

func NewStore() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	base := filepath.Join(home, ".gsc", "credentials")
	if err := ensureDir(base); err != nil {
		return nil, err
	}
	return &Store{BaseDir: base}, nil
}

func (s *Store) accountDir(account string) (string, error) {
	if !accountNameRe.MatchString(account) {
		return "", fmt.Errorf("invalid account name %q (use letters, digits, '_' or '-', max 64 chars)", account)
	}
	return filepath.Join(s.BaseDir, account), nil
}

func (s *Store) ClientPath(account string) (string, error) {
	dir, err := s.accountDir(account)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "client.json"), nil
}

func (s *Store) TokenPath(account string) (string, error) {
	dir, err := s.accountDir(account)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "token.json"), nil
}

func (s *Store) WriteClientCredentials(account, clientID, clientSecret string) error {
	if clientID == "" || clientSecret == "" {
		return errors.New("client_id and client_secret are required")
	}

	dir, err := s.accountDir(account)
	if err != nil {
		return err
	}
	if err := ensureDir(dir); err != nil {
		return err
	}

	payload := map[string]any{
		"installed": map[string]any{
			"client_id":     clientID,
			"client_secret": clientSecret,
			"auth_uri":      "https://accounts.google.com/o/oauth2/auth",
			"token_uri":     "https://oauth2.googleapis.com/token",
			"redirect_uris": []string{"http://127.0.0.1"},
		},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	p, err := s.ClientPath(account)
	if err != nil {
		return err
	}
	return writeFileAtomic(p, b, 0o600)
}

func (s *Store) ReadClientJSON(account string) ([]byte, error) {
	p, err := s.ClientPath(account)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(p)
}

func (s *Store) ReadToken(account string) (*oauth2.Token, error) {
	p, err := s.TokenPath(account)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var tok oauth2.Token
	if err := json.Unmarshal(b, &tok); err != nil {
		return nil, err
	}
	return &tok, nil
}

func (s *Store) WriteToken(account string, tok *oauth2.Token) error {
	if tok == nil {
		return errors.New("token is nil")
	}

	dir, err := s.accountDir(account)
	if err != nil {
		return err
	}
	if err := ensureDir(dir); err != nil {
		return err
	}

	b, err := json.Marshal(tok)
	if err != nil {
		return err
	}
	p, err := s.TokenPath(account)
	if err != nil {
		return err
	}
	return writeFileAtomic(p, b, 0o600)
}

type AccountInfo struct {
	Name        string
	HasClient   bool
	HasToken    bool
	TokenExpiry time.Time
}

func (s *Store) ListAccounts() ([]AccountInfo, error) {
	entries, err := os.ReadDir(s.BaseDir)
	if err != nil {
		return nil, err
	}
	var out []AccountInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if !accountNameRe.MatchString(name) {
			continue
		}

		dir := filepath.Join(s.BaseDir, name)
		clientPath := filepath.Join(dir, "client.json")
		tokenPath := filepath.Join(dir, "token.json")

		ai := AccountInfo{Name: name}
		if _, err := os.Stat(clientPath); err == nil {
			ai.HasClient = true
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		if b, err := os.ReadFile(tokenPath); err == nil {
			ai.HasToken = true
			var tok oauth2.Token
			if err := json.Unmarshal(b, &tok); err == nil {
				ai.TokenExpiry = tok.Expiry
			}
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		out = append(out, ai)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func ensureDir(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	_ = os.Chmod(dir, 0o700)
	return nil
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := ensureDir(dir); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}()

	if err := tmp.Chmod(perm); err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	_ = os.Chmod(path, perm)
	return nil
}
