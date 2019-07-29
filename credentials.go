package main

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"io/ioutil"
	"net/http"
)

// GoogleClient returns an authenticated HTTP client for accessing Google APIs
func GoogleClient(scope ...string) (*http.Client, error) {
	b, err := ioutil.ReadFile("client_secret.json")
	if err != nil {
		return nil, err
	}
	c, err := google.ConfigFromJSON(b, scope...)
	if err != nil {
		return nil, err
	}

	var t *oauth2.Token
	ctx := context.Background()
	if b, err = ioutil.ReadFile("cache/token.json"); err == nil {
		creds, err := google.CredentialsFromJSON(ctx, b, scope...)
		if err != nil {
			return nil, err
		}
		t, err = creds.TokenSource.Token()
		if err != nil {
			return nil, err
		}
	} else {
		fmt.Printf("Paste code from %s\n", c.AuthCodeURL("state"))
		var code string
		n, err := fmt.Scan(&code)
		if n != 1 {
			return nil, fmt.Errorf("error: code not given / too many given")
		} else if err != nil {
			return nil, err
		}
		t, err = c.Exchange(ctx, code)
		if err != nil {
			return nil, err
		}
		b, err = json.Marshal(t)
		if err != nil {
			return nil, err
		}
		err = ioutil.WriteFile("cache/token.json", b, 0644)
		if err != nil {
			return nil, err
		}
	}

	return c.Client(ctx, t), nil
}
