package openshift

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httputil"
	"unsafe"

	yaml "gopkg.in/yaml.v2"
)

// WhoAmI checks with OSO who owns the current token.
// returns the username
func WhoAmI(config Config) (string, error) {
	whoamiURL := config.MasterURL + "/oapi/v1/users/~"
	user, err := get(whoamiURL, config.Token)
	if err != nil {
		return "", err
	}

	return user.Metadata.Name, nil
}

type user struct {
	Metadata struct {
		Name string
	}
}

func get(url, token string) (*user, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/yaml")
	req.Header.Set("Content-Type", "application/yaml")
	req.Header.Set("Authorization", "Bearer "+token)

	// debug only
	rb, _ := httputil.DumpRequest(req, true)
	if false {
		fmt.Println(string(rb))
	}

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	b := buf.Bytes()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Unknown response:\n%v\n%v", *(*string)(unsafe.Pointer(&b)), string(rb))
	}

	var u user
	err = yaml.Unmarshal(b, &u)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
