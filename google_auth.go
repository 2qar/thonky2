package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// authClient wraps an authenticated http client to make accessing Google APIs a little easier
type authClient struct {
	*http.Client
}

// GetStruct unmarshals the response from a Google API endpoint into the given interface
func (a *authClient) GetStruct(s interface{}, url string) error {
	r, err := a.Get(url)
	if err != nil {
		return err
	} else if r.StatusCode != 200 {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("%s", b)
	}
	defer r.Body.Close()

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, s)
}
