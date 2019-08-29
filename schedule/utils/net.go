package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// Gets unmarshals the response from a url into the given interface
func Gets(c *http.Client, s interface{}, url string) error {
	r, err := c.Get(url)
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
