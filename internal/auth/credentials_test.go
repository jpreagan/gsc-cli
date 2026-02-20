package auth

import "testing"

func TestExtractClientIDSecretFromCredentialsJSON_Installed(t *testing.T) {
	b := []byte(`{"installed":{"client_id":"id","client_secret":"sec"}}`)
	id, sec, err := ExtractClientIDSecretFromCredentialsJSON(b)
	if err != nil {
		t.Fatalf("ExtractClientIDSecretFromCredentialsJSON: %v", err)
	}
	if id != "id" || sec != "sec" {
		t.Fatalf("got id=%q sec=%q want id=%q sec=%q", id, sec, "id", "sec")
	}
}

func TestExtractClientIDSecretFromCredentialsJSON_Web(t *testing.T) {
	b := []byte(`{"web":{"client_id":"id","client_secret":"sec"}}`)
	id, sec, err := ExtractClientIDSecretFromCredentialsJSON(b)
	if err != nil {
		t.Fatalf("ExtractClientIDSecretFromCredentialsJSON: %v", err)
	}
	if id != "id" || sec != "sec" {
		t.Fatalf("got id=%q sec=%q want id=%q sec=%q", id, sec, "id", "sec")
	}
}

func TestExtractClientIDSecretFromCredentialsJSON_Invalid(t *testing.T) {
	for _, in := range [][]byte{
		[]byte(`{}`),
		[]byte(`{"installed":{"client_id":"","client_secret":"sec"}}`),
		[]byte(`{"installed":{"client_id":"id","client_secret":""}}`),
		[]byte(`not json`),
	} {
		_, _, err := ExtractClientIDSecretFromCredentialsJSON(in)
		if err == nil {
			t.Fatalf("expected error for input=%q", string(in))
		}
	}
}
