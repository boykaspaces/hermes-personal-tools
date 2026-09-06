package googleauth

import "testing"

func TestParse(t *testing.T) {
	credentials, err := Parse(`{
		"client_id":"client",
		"client_secret":"secret",
		"refresh_token":"refresh"
	}`)
	if err != nil {
		t.Fatal(err)
	}
	if credentials.ClientID != "client" || credentials.RefreshToken != "refresh" {
		t.Fatalf("unexpected credentials: %#v", credentials)
	}

	if _, err := Parse(`{"client_id":"client"}`); err == nil {
		t.Fatal("expected missing field validation error")
	}
}
