package openshift

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/bitly/go-simplejson"
	"github.com/fabric8-services/fabric8-tenant/keycloak"
)

func KubeConnected(kcConfig keycloak.Config, config Config, username string) (string, error) {
	if KubernetesMode() {
		name := createName(username)
		jenkinsNS := fmt.Sprintf("%v-jenkins", name)
		return EnsureKeyCloakHasJenkinsRedirectURL(config, kcConfig, jenkinsNS)
	}
	return "not required for OpenShift", nil
}

// HasJenkinsNamespace returns true if the tenant namespace has been created
func HasJenkinsNamespace(config Config, username string) bool {
	if KubernetesMode() {
		name := createName(username)
		jenkinsNS := fmt.Sprintf("%v-jenkins", name)
		namespaceURL := fmt.Sprintf("/api/v1/namespaces/%s", jenkinsNS)
		ns, err := getResource(config, namespaceURL)
		if ns != nil && err == nil {
			return true
		}
	}
	return false
}

// EnsureKeyCloakHasJenkinsRedirectURL checks that the client has a redirect URI for the jenkins URL
func EnsureKeyCloakHasJenkinsRedirectURL(config Config, kcConfig keycloak.Config, jenkinsNS string) (string, error) {
	fabric8Namespace := os.Getenv("KUBERNETES_NAMESPACE")
	if fabric8Namespace == "" {
		fabric8Namespace = "fabric8"
	}

	token, err := GetKeyCloakAdminToken(config, kcConfig, fabric8Namespace)
	if err != nil {
		return "No admin token for KeyCloak", err
	}
	jenkinsUrl, err := FindServiceURL(config, jenkinsNS, "jenkins")
	if err != nil {
		return "Waiting for your external Jenkins URL to become available", err
	}
	clientID := "fabric8-online-platform"
	realm := kcConfig.Realm
	clientsURL := strings.TrimSuffix(kcConfig.BaseURL, "/") + "/auth/admin/realms/" + realm + "/clients"
	clientQueryURL := clientsURL + "?clientId=" + clientID

	status, jsonText, err := doGet(config, clientQueryURL, token)
	if err != nil {
		return fmt.Sprintf("Cannot query the keycloak realm %s for client %s", realm, clientID), err
	}
	if status < 200 || status >= 400 {
		return fmt.Sprintf("Cannot query the keycloak realm %s for client %s", realm, clientID), fmt.Errorf("Failed to load KeyCloak client at %s status code %d", clientsURL, status)
	}
	redirectURL := strings.TrimSuffix(jenkinsUrl, "/") + "/securityRealm/finishLogin"
	id, jsonText, err := addRedirectUrl(jsonText, redirectURL)
	if err != nil {
		return "Failed to add redirectURL for Jenkins into KeyCLoak JSON", err
	}
	if len(jsonText) > 0 {
		clientURL := clientsURL + "/" + id
		_, err = postJson(config, "PUT", clientURL, token, jsonText)
		if err != nil {
			return "Failed to register redirectURL for Jenkins into KeyCloak", err
		}
	}
	return "Connected", nil
}

func FindKeyCloakUserPassword(config Config, namespace string) (string, string, error) {
	secretName := "keycloak"

	url := fmt.Sprintf("/api/v1/namespaces/%s/secrets/%s", namespace, secretName)

	cm, err := getResource(config, url)
	if err != nil {
		return "", "", fmt.Errorf("Failed to load %s secret in namespace %s due to %v", secretName, namespace, err)
	}
	data, ok := cm["data"].(map[interface{}]interface{})
	if ok {
		userName, err := mandatorySecretProperty(data, namespace, secretName, "kc.user")
		if err != nil {
			return "", "", err
		}
		password, err := mandatorySecretProperty(data, namespace, secretName, "kc.password")
		if err != nil {
			return "", "", err
		}
		return userName, password, nil
	}
	return "", "", fmt.Errorf("Could not find the  data in Secret %s in namespace %s", secretName, namespace)
}

func mandatorySecretProperty(data map[interface{}]interface{}, namespace string, secretName string, property string) (string, error) {
	text := stringValue(data, property)
	if len(text) > 0 {
		bytes, err := base64.StdEncoding.DecodeString(text)
		if err != nil {
			return "", fmt.Errorf("Failed to base64 decode the %s property for secret %s in namespace %s due to %v", property, secretName, namespace, err)
		}
		return strings.TrimSpace(string(bytes)), nil
	}
	return "", fmt.Errorf("No property %s found secret %s in namespace %s", property, secretName, namespace)
}

func addRedirectUrl(jsonText string, url string) (string, string, error) {
	js, err := simplejson.NewJson([]byte(jsonText))
	if err != nil {
		return "", "", err
	}
	obj := js.GetIndex(0)
	if obj.Interface() == nil {
		return "", "", fmt.Errorf("No Client could be found from KeyCloak!")
	}

	id, err := obj.Get("id").String()
	if err != nil {
		return "", "", err
	}
	if len(id) == 0 {
		return "", "", fmt.Errorf("No id property found in the KeyCloak client JSON")
	}
	redirectUris, err := obj.Get("redirectUris").StringArray()
	if err != nil {
		return "", "", err
	}
	for _, text := range redirectUris {
		if text == url {
			return "", "", nil
		}
	}
	redirectUris = append(redirectUris, url)
	obj.Set("redirectUris", redirectUris)

	data, err := obj.MarshalJSON()
	if err != nil {
		return "", "", err
	}
	return id, string(data), nil
}

type usertoken struct {
	AccessToken string `json:"access_token"`
}

func GetKeyCloakAdminToken(config Config, kcConfig keycloak.Config, namespace string) (string, error) {
	user, pwd, err := FindKeyCloakUserPassword(config, namespace)
	if err != nil {
		return "", err
	}
	data := url.Values{}
	data.Add("username", user)
	data.Add("password", pwd)
	data.Add("grant_type", "password")
	data.Add("client_id", "admin-cli")

	url := strings.TrimSuffix(kcConfig.BaseURL, "/") + "/auth/realms/master/protocol/openid-connect/token"

	opts := ApplyOptions{Config: config}
	client := opts.CreateHttpClient()

	body := []byte(data.Encode())
	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(body)))
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	b := buf.Bytes()
	status := resp.StatusCode
	if status < 200 || status >= 400 {
		return "", fmt.Errorf("Failed to get the KeyCloak openid-connect token. Status %d from POST to %s", status, url)
	}
	var u usertoken
	err = json.Unmarshal(b, &u)
	if err != nil {
		return "", err
	}
	token := u.AccessToken
	if len(token) == 0 {
		return "", fmt.Errorf("Missing `access_token` property from KeyCloak openid-connect response %s", string(b))
	}
	return token, nil
}

func postJson(config Config, method string, url string, token string, json string) (string, error) {
	req, err := http.NewRequest(method, url, strings.NewReader(json))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	opts := ApplyOptions{Config: config}
	client := opts.CreateHttpClient()
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	b := buf.Bytes()
	result := string(b)
	status := resp.StatusCode

	if status < 200 || status > 300 {
		return "", fmt.Errorf("Failed to %s url %s due to status code %d", method, url, status)
	}
	return result, nil
}

func doGet(config Config, url string, token string) (int, string, error) {
	var body []byte
	req, err := http.NewRequest("GET", url, bytes.NewReader(body))
	if err != nil {
		return 500, "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	opts := ApplyOptions{Config: config}

	client := opts.CreateHttpClient()
	resp, err := client.Do(req)
	if err != nil {
		return 500, "", err
	}

	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	b := buf.Bytes()
	result := string(b)
	status := resp.StatusCode

	return status, result, nil
}
