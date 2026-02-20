package auth

import (
	"encoding/json"
	"errors"
	"fmt"
)

var errInvalidCredentialsJSON = errors.New("invalid credentials JSON (expected installed/web client_id and client_secret)")

type googleOAuthCredentials struct {
	Installed *struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	} `json:"installed"`
	Web *struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	} `json:"web"`
}

func ExtractClientIDSecretFromCredentialsJSON(b []byte) (clientID string, clientSecret string, err error) {
	var c googleOAuthCredentials
	if err := json.Unmarshal(b, &c); err != nil {
		return "", "", fmt.Errorf("%w: %v", errInvalidCredentialsJSON, err)
	}

	switch {
	case c.Installed != nil:
		clientID = c.Installed.ClientID
		clientSecret = c.Installed.ClientSecret
	case c.Web != nil:
		clientID = c.Web.ClientID
		clientSecret = c.Web.ClientSecret
	default:
		return "", "", errInvalidCredentialsJSON
	}

	if clientID == "" || clientSecret == "" {
		return "", "", errInvalidCredentialsJSON
	}
	return clientID, clientSecret, nil
}
