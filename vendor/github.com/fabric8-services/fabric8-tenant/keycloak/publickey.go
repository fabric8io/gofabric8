package keycloak

import (
	"bytes"
	"crypto/rsa"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"unsafe"

	jwt "github.com/dgrijalva/jwt-go"
	yaml "gopkg.in/yaml.v2"
)

// GetPublicKey return the rsa.PublicKey parsed key from the Keycloak instance that can be used
// to verify tokens
func GetPublicKey(config Config) (*rsa.PublicKey, error) {
	resp, err := getPublicKey(config.RealmAuthURL())
	if err != nil {
		return nil, err
	}
	pk, err := jwt.ParseRSAPublicKeyFromPEM([]byte(formatPublicKey(resp.PublicKey)))
	if err != nil {
		return nil, err
	}
	return pk, nil
}

func formatPublicKey(data string) string {
	return fmt.Sprintf("-----BEGIN PUBLIC KEY-----\n%v\n-----END PUBLIC KEY-----", data)
}

type kcEnv struct {
	PublicKey string `yaml:"public_key"`
}

func getPublicKey(url string) (*kcEnv, error) {

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	// for debug only
	rb, _ := httputil.DumpRequest(req, true)
	if false {
		fmt.Println(string(rb))
	}

	client := createHttpClient()
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

	var u kcEnv
	err = yaml.Unmarshal(b, &u)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func createHttpClient() *http.Client {
	// when running on minishift there is usually no certs on the HTTPS endpoint for KeyCloak
	// so lets allow host verification to be disabled
	flag := os.Getenv("KEYCLOAK_SKIP_HOST_VERIFY")
	if strings.ToLower(flag) == "true" {
		return &http.Client{
			Transport: &http.Transport{
				// we need to disable TLS verify on minishift
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
	}
	return http.DefaultClient
}
