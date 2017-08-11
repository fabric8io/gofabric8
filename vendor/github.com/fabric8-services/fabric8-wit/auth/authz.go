package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/rest"
	errs "github.com/pkg/errors"
)

const (
	// PermissionTypeResource is to used in a Keycloak Permission payload: {"type":"resource"}
	PermissionTypeResource = "resource"
	// PolicyTypeUser is to used in a Keycloak Policy payload: {"type":"user"}
	PolicyTypeUser = "user"
	// PolicyLogicPossitive is to used in a Keycloak Policy payload: {"logic":""POSITIVE"}
	PolicyLogicPossitive = "POSITIVE"
	// PolicyDecisionStrategyUnanimous is to used in a Keycloak Policy payload: {"decisionStrategy":""UNANIMOUS"}
	PolicyDecisionStrategyUnanimous = "UNANIMOUS"
)

// KeycloakResource represents a keycloak resource payload
type KeycloakResource struct {
	Name   string    `json:"name"`
	Owner  *string   `json:"owner,omitempty"`
	Type   string    `json:"type"`
	Scopes *[]string `json:"scopes,omitempty"`
	URI    *string   `json:"uri,omitempty"`
}

type createResourceRequestResultPayload struct {
	ID string `json:"_id"`
}

type createPolicyRequestResultPayload struct {
	ID string `json:"id"`
}

type clientData struct {
	ID       string `json:"id"`
	ClientID string `json:"clientID"`
}

// KeycloakPolicy represents a keycloak policy payload
type KeycloakPolicy struct {
	ID               *string          `json:"id,omitempty"`
	Name             string           `json:"name"`
	Type             string           `json:"type"`
	Logic            string           `json:"logic"`
	DecisionStrategy string           `json:"decisionStrategy"`
	Config           PolicyConfigData `json:"config"`
}

// PolicyConfigData represents a config in the keycloak policy payload
type PolicyConfigData struct {
	//"users":"[\"<ID>\",\"<ID>\"]"
	UserIDs string `json:"users"`
}

// Token represents a Keycloak token response
type Token struct {
	AccessToken      *string `json:"access_token,omitempty"`
	ExpiresIn        *int64  `json:"expires_in,omitempty"`
	NotBeforePolicy  *int64  `json:"not-before-policy,omitempty"`
	RefreshExpiresIn *int64  `json:"refresh_expires_in,omitempty"`
	RefreshToken     *string `json:"refresh_token,omitempty"`
	TokenType        *string `json:"token_type,omitempty"`
}

// AddUserToPolicy adds the user ID to the policy
func (p *KeycloakPolicy) AddUserToPolicy(userID string) bool {
	currentUsers := p.Config.UserIDs
	if strings.Contains(currentUsers, userID) {
		return false
	}
	s := strings.Split(currentUsers, "]")
	if len(s) > 1 {
		p.Config.UserIDs = fmt.Sprintf("%s,\"%s\"]", s[0], userID)
	} else {
		p.Config.UserIDs = fmt.Sprintf("[\"%s\"]", userID)
	}
	return true
}

// RemoveUserFromPolicy removes the user ID from the policy
func (p *KeycloakPolicy) RemoveUserFromPolicy(userID string) bool {
	currentUsers := p.Config.UserIDs
	s := strings.Split(currentUsers, ",")
	var found bool
	var i int
	newUsers := make([]string, len(s))
	for _, id := range s {
		newUsers[i] = strings.Trim(id, "[]")
		if strings.Trim(newUsers[i], "\"") == userID {
			found = true
		} else {
			i++
		}
	}
	if !found {
		return false
	}
	newUsers = newUsers[0 : len(newUsers)-1]
	p.Config.UserIDs = fmt.Sprintf("[%s]", strings.Join(newUsers, ","))
	return true
}

// KeycloakPermission represents a keycloak permission payload
type KeycloakPermission struct {
	ID               *string              `json:"id,omitempty"`
	Name             string               `json:"name"`
	Type             string               `json:"type"`
	Logic            string               `json:"logic"`
	DecisionStrategy string               `json:"decisionStrategy"`
	Config           PermissionConfigData `json:"config"`
}

// PermissionConfigData represents a config in the keycloak permission payload
type PermissionConfigData struct {
	Resources     string `json:"resources"`
	ApplyPolicies string `json:"applyPolicies"`
}

// UserInfo represents a user info Keycloak payload
type UserInfo struct {
	Sub               string `json:"sub"`
	Name              string `json:"name"`
	PreferredUsername string `json:"preferred_username"`
	GivenName         string `json:"given_name"`
	FamilyName        string `json:"family_name"`
	Email             string `json:"email"`
}

// EntitlementResource represents a payload for obtaining entitlement for specific resource
type EntitlementResource struct {
	Permissions []ResourceSet `json:"permissions"`
}

// ResourceSet represents a resource set for Entitlement payload
type ResourceSet struct {
	Name string  `json:"resource_set_name"`
	ID   *string `json:"resource_set_id,omitempty"`
}

type entitlementResult struct {
	Rpt string `json:"rpt"`
}

// TokenPayload represents an rpt token
type TokenPayload struct {
	jwt.StandardClaims
	Authorization *AuthorizationPayload `json:"authorization"`
}

// AuthorizationPayload represents an authz payload in the rpt token
type AuthorizationPayload struct {
	Permissions []Permissions `json:"permissions"`
}

// Permissions represents a "permissions" in the AuthorizationPayload
type Permissions struct {
	ResourceSetName *string `json:"resource_set_name"`
	ResourceSetID   *string `json:"resource_set_id"`
}

// VerifyResourceUser returns true if the user among the resource collaborators
func VerifyResourceUser(ctx context.Context, token string, resourceName string, entitlementEndpoint string) (bool, error) {
	resource := EntitlementResource{
		Permissions: []ResourceSet{{Name: resourceName}},
	}
	ent, err := GetEntitlement(ctx, entitlementEndpoint, &resource, token)
	if err != nil {
		return false, err
	}
	return ent != nil, nil
}

// CreateResource creates a Keycloak resource
func CreateResource(ctx context.Context, resource KeycloakResource, authzEndpoint string, protectionAPIToken string) (string, error) {
	log.Debug(ctx, map[string]interface{}{
		"resource": resource,
	}, "Creating a new Keycloak resource")

	b, err := json.Marshal(resource)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"resource": resource,
			"err":      err.Error(),
		}, "unable to marshal keycloak resource struct")
		return "", errors.NewInternalError(ctx, errs.Wrap(err, "unable to marshal keycloak resource struct"))
	}

	req, err := http.NewRequest("POST", authzEndpoint, strings.NewReader(string(b)))
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err.Error(),
		}, "unable to create http request")
		return "", errors.NewInternalError(ctx, errs.Wrap(err, "unable to create http request"))
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+protectionAPIToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"resource": resource,
			"err":      err.Error(),
		}, "unable to create a Keycloak resource")
		return "", errors.NewInternalError(ctx, errs.Wrap(err, "unable to create a Keycloak resource"))
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		log.Error(ctx, map[string]interface{}{
			"resource":        resource,
			"response_status": res.Status,
			"response_body":   rest.ReadBody(res.Body),
		}, "unable to create a Keycloak resource")
		return "", errors.NewInternalError(ctx, errs.Errorf("unable to create a Keycloak resource. Response status: "+res.Status+". Responce body: "+rest.ReadBody(res.Body)))
	}
	jsonString := rest.ReadBody(res.Body)

	var r createResourceRequestResultPayload
	err = json.Unmarshal([]byte(jsonString), &r)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"resource":    resource,
			"json_string": jsonString,
		}, "unable to unmarshal json with the create keycloak resource request result")

		return "", errors.NewInternalError(ctx, errs.Wrapf(err, "unable to unmarshal json with the create keycloak resource request result %s ", jsonString))
	}

	log.Debug(ctx, map[string]interface{}{
		"resource_name": resource.Name,
		"resource_id":   r.ID,
	}, "Keycloak resource created")

	return r.ID, nil
}

// GetClientID obtains the internal client ID associated with keycloak client
func GetClientID(ctx context.Context, clientsEndpoint string, publicClientID string, protectionAPIToken string) (string, error) {
	req, err := http.NewRequest("GET", clientsEndpoint, nil)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err.Error(),
		}, "unable to create http request")
		return "", errors.NewInternalError(ctx, errs.Wrap(err, "unable to create http request"))
	}
	req.Header.Add("Authorization", "Bearer "+protectionAPIToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"public_client_id": publicClientID,
			"err":              err.Error(),
		}, "unable to obtain keycloak client ID")
		return "", errors.NewInternalError(ctx, errs.Wrap(err, "unable to obtain keycloak client ID"))
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		log.Error(ctx, map[string]interface{}{
			"public_client_id": publicClientID,
			"response_status":  res.Status,
			"response_body":    rest.ReadBody(res.Body),
		}, "unable to obtain keycloak client ID")
		return "", errors.NewInternalError(ctx, errs.New("unable to obtain keycloak client ID. Response status: "+res.Status+". Responce body: "+rest.ReadBody(res.Body)))
	}
	jsonString := rest.ReadBody(res.Body)

	var r []clientData
	err = json.Unmarshal([]byte(jsonString), &r)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"public_client_id": publicClientID,
			"err":              err.Error(),
		}, "unable to unmarshal json with client ID")
		return "", errors.NewInternalError(ctx, errs.Wrapf(err, "error when unmarshal json with client ID result %s ", jsonString))
	}
	for _, client := range r {
		if publicClientID == client.ClientID {
			return client.ID, nil
		}
	}
	log.Error(ctx, map[string]interface{}{
		"public_client_id": publicClientID,
		"json":             jsonString,
	}, "unable to find client ID '"+publicClientID+"' among available IDs: "+jsonString)
	return "", errors.NewInternalError(ctx, errs.New("unable to find keycloak client ID"))
}

// CreatePolicy creates a Keycloak policy
func CreatePolicy(ctx context.Context, clientsEndpoint string, clientID string, policy KeycloakPolicy, protectionAPIToken string) (string, error) {
	b, err := json.Marshal(policy)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"policy": policy,
			"err":    err.Error(),
		}, "unable to marshal keycloak policy struct")
		return "", errors.NewInternalError(ctx, errs.Wrap(err, "unable to marshal keycloak policy struct"))
	}

	req, err := http.NewRequest("POST", clientsEndpoint+"/"+clientID+"/authz/resource-server/policy", strings.NewReader(string(b)))
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err.Error(),
		}, "unable to create http request")
		return "", errors.NewInternalError(ctx, errs.Wrap(err, "unable to create http request"))
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+protectionAPIToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"client_id": clientID,
			"policy":    policy,
			"err":       err.Error(),
		}, "unable to create the Keycloak policy")
		return "", errors.NewInternalError(ctx, errs.Wrap(err, "unable to create the Keycloak policy"))
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		log.Error(ctx, map[string]interface{}{
			"client_id":       clientID,
			"policy":          policy,
			"response_status": res.Status,
			"response_body":   rest.ReadBody(res.Body),
		}, "unable to update the Keycloak policy")
		return "", errors.NewInternalError(ctx, errs.New("unable to create the Keycloak policy. Response status: "+res.Status+". Responce body: "+rest.ReadBody(res.Body)))
	}
	jsonString := rest.ReadBody(res.Body)

	var r createPolicyRequestResultPayload
	err = json.Unmarshal([]byte(jsonString), &r)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"client_id":   clientID,
			"policy":      policy,
			"json_string": jsonString,
		}, "unable to unmarshal json with the create keycloak policy request result")
		return "", errors.NewInternalError(ctx, errs.Wrapf(err, "error when unmarshal json with the create keycloak policy request result %s ", jsonString))
	}

	return r.ID, nil
}

// CreatePermission creates a Keycloak permission
func CreatePermission(ctx context.Context, clientsEndpoint string, clientID string, permission KeycloakPermission, protectionAPIToken string) (string, error) {
	b, err := json.Marshal(permission)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"permission": permission,
			"err":        err.Error(),
		}, "unable to marshal keycloak permission struct")
		return "", errors.NewInternalError(ctx, errs.Wrap(err, "unable to marshal keycloak permission struct"))
	}

	req, err := http.NewRequest("POST", clientsEndpoint+"/"+clientID+"/authz/resource-server/policy", strings.NewReader(string(b)))
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err.Error(),
		}, "unable to create http request")
		return "", errors.NewInternalError(ctx, errs.Wrap(err, "unable to create http request"))
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+protectionAPIToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"client_id":  clientID,
			"permission": permission,
			"err":        err.Error(),
		}, "unable to create the Keycloak permission")
		return "", errors.NewInternalError(ctx, errs.Wrap(err, "unable to create the Keycloak permission"))
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		log.Error(ctx, map[string]interface{}{
			"client_id":       clientID,
			"permission":      permission,
			"response_status": res.Status,
			"response_body":   rest.ReadBody(res.Body),
		}, "unable to update the Keycloak permission")
		return "", errors.NewInternalError(ctx, errs.New("unable to create the Keycloak permission. Response status: "+res.Status+". Responce body: "+rest.ReadBody(res.Body)))
	}
	jsonString := rest.ReadBody(res.Body)

	var r createPolicyRequestResultPayload
	err = json.Unmarshal([]byte(jsonString), &r)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"client_id":   clientID,
			"permission":  permission,
			"json_string": jsonString,
		}, "unable to unmarshal json with the create keycloak permission request result")
		return "", errors.NewInternalError(ctx, errs.Wrapf(err, "error when unmarshal json with the create keycloak permission request result %s ", jsonString))
	}

	return r.ID, nil
}

// DeleteResource deletes the Keycloak resource assosiated with the space
func DeleteResource(ctx context.Context, kcResourceID string, authzEndpoint string, protectionAPIToken string) error {
	if kcResourceID == "" {
		log.Error(ctx, map[string]interface{}{}, "kc-resource-id is emtpy")
		return errors.NewBadParameterError("kcResourceID", kcResourceID)
	}
	log.Debug(ctx, map[string]interface{}{
		"kc_resource_id": kcResourceID,
	}, "Deleting the Keycloak resource")

	req, err := http.NewRequest("DELETE", authzEndpoint+"/"+kcResourceID, nil)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err.Error(),
		}, "unable to create http request")
		return errors.NewInternalError(ctx, errs.Wrap(err, "unable to create http request"))
	}
	req.Header.Add("Authorization", "Bearer "+protectionAPIToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"kc_resource_id": kcResourceID,
			"err":            err.Error(),
		}, "unable to delete the Keycloak resource")
		return errors.NewInternalError(ctx, errs.Wrap(err, "unable to delete the Keycloak resource"))
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		log.Error(ctx, map[string]interface{}{
			"kc_resource_id":  kcResourceID,
			"response_status": res.Status,
			"response_body":   rest.ReadBody(res.Body),
		}, "unable to delete the Keycloak resource")
		return errors.NewInternalError(ctx, errs.New("unable to delete the Keycloak resource. Response status: "+res.Status+". Responce body: "+rest.ReadBody(res.Body)))
	}

	log.Debug(ctx, map[string]interface{}{
		"kc_resource_id": kcResourceID,
	}, "Keycloak resource deleted")

	return nil
}

// DeletePolicy deletes the Keycloak policy
func DeletePolicy(ctx context.Context, clientsEndpoint string, clientID string, policyID string, protectionAPIToken string) error {
	if policyID == "" {
		log.Error(ctx, map[string]interface{}{}, "policy-id is emtpy")
		return errors.NewBadParameterError("policyID", policyID)
	}
	req, err := http.NewRequest("DELETE", clientsEndpoint+"/"+clientID+"/authz/resource-server/policy/"+policyID, nil)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err.Error(),
		}, "unable to create http request")
		return errors.NewInternalError(ctx, errs.Wrap(err, "unable to create http request"))
	}
	req.Header.Add("Authorization", "Bearer "+protectionAPIToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"policy_id": policyID,
			"err":       err.Error(),
		}, "unable to delete the Keycloak policy")
		return errors.NewInternalError(ctx, errs.Wrap(err, "unable to delete the Keycloak policy"))
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		log.Error(ctx, map[string]interface{}{
			"policy_id":       policyID,
			"response_status": res.Status,
			"response_body":   rest.ReadBody(res.Body),
		}, "unable to delete the Keycloak policy")
		return errors.NewInternalError(ctx, errs.New("unable to delete the Keycloak policy. Response status: "+res.Status+". Responce body: "+rest.ReadBody(res.Body)))
	}

	log.Debug(ctx, map[string]interface{}{
		"policy_id": policyID,
	}, "Keycloak policy deleted")

	return nil
}

// DeletePermission deletes the Keycloak permission
func DeletePermission(ctx context.Context, clientsEndpoint string, clientID string, permissionID string, protectionAPIToken string) error {
	if permissionID == "" {
		log.Error(ctx, map[string]interface{}{}, "permission-id is emtpy")
		return errors.NewBadParameterError("permissionID", permissionID)
	}
	req, err := http.NewRequest("DELETE", clientsEndpoint+"/"+clientID+"/authz/resource-server/policy/"+permissionID, nil)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err.Error(),
		}, "unable to create http request")
		return errors.NewInternalError(ctx, errs.Wrap(err, "unable to create http request"))
	}
	req.Header.Add("Authorization", "Bearer "+protectionAPIToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"permission_id": permissionID,
			"err":           err.Error(),
		}, "unable to delete the Keycloak permission")
		return errors.NewInternalError(ctx, errs.Wrap(err, "unable to delete the Keycloak permission"))
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		log.Error(ctx, map[string]interface{}{
			"permission_id":   permissionID,
			"response_status": res.Status,
			"response_body":   rest.ReadBody(res.Body),
		}, "unable to delete the Keycloak permission")
		return errors.NewInternalError(ctx, errs.New("unable to delete the Keycloak permission. Response status: "+res.Status+". Responce body: "+rest.ReadBody(res.Body)))
	}

	log.Debug(ctx, map[string]interface{}{
		"permission_id": permissionID,
	}, "Keycloak permission deleted")

	return nil
}

// GetPolicy obtains a policy from Keycloak
func GetPolicy(ctx context.Context, clientsEndpoint string, clientID string, policyID string, protectionAPIToken string) (*KeycloakPolicy, error) {
	if policyID == "" {
		log.Error(ctx, map[string]interface{}{}, "policy-id is emtpy")
		return nil, errors.NewBadParameterError("policyID", policyID)
	}
	req, err := http.NewRequest("GET", clientsEndpoint+"/"+clientID+"/authz/resource-server/policy/"+policyID, nil)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err.Error(),
		}, "unable to create http request")
		return nil, errors.NewInternalError(ctx, errs.Wrap(err, "unable to create http request"))
	}
	req.Header.Add("Authorization", "Bearer "+protectionAPIToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"client_id": clientID,
			"policy_id": policyID,
			"err":       err.Error(),
		}, "unable to obtain a Keycloak policy")
		return nil, errors.NewInternalError(ctx, errs.Wrap(err, "unable to obtain a Keycloak policy"))
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
		// OK
	case http.StatusNotFound:
		log.Error(ctx, map[string]interface{}{
			"client_id": clientID,
			"policy_id": policyID,
		}, "Keycloak policy is not found")
		return nil, errors.NewNotFoundError("policy", policyID)
	default:
		log.Error(ctx, map[string]interface{}{
			"client_id":       clientID,
			"policy_id":       policyID,
			"response_status": res.Status,
			"response_body":   rest.ReadBody(res.Body),
		}, "unable to obtain a Keycloak policy")
		return nil, errors.NewInternalError(ctx, errs.New("unable to obtain a Keycloak policy. Response status: "+res.Status+". Responce body: "+rest.ReadBody(res.Body)))
	}
	jsonString := rest.ReadBody(res.Body)

	var r KeycloakPolicy
	err = json.Unmarshal([]byte(jsonString), &r)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"client_id":   clientID,
			"policy_id":   policyID,
			"json_string": jsonString,
		}, "unable to unmarshal json with the get keycloak policy request result")
		return nil, errors.NewInternalError(ctx, errs.Wrapf(err, "error when unmarshal json with get the keycloak policy request result %s ", jsonString))
	}

	return &r, nil
}

// UpdatePolicy updates the Keycloak policy
func UpdatePolicy(ctx context.Context, clientsEndpoint string, clientID string, policy KeycloakPolicy, protectionAPIToken string) error {
	if policy.ID == nil {
		log.Error(ctx, map[string]interface{}{}, "Policy ID is nil")
		return errors.NewBadParameterError("policy-id", "nil")
	}
	b, err := json.Marshal(policy)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"policy": policy,
			"err":    err.Error(),
		}, "unable to marshal keycloak policy struct")
		return errors.NewInternalError(ctx, errs.Wrap(err, "unable to marshal keycloak policy struct"))
	}

	req, err := http.NewRequest("PUT", clientsEndpoint+"/"+clientID+"/authz/resource-server/policy/"+*policy.ID, strings.NewReader(string(b)))
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err.Error(),
		}, "unable to create http request")
		return errors.NewInternalError(ctx, errs.Wrap(err, "unable to create http request"))
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+protectionAPIToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"client_id": clientID,
			"policy":    policy,
			"err":       err.Error(),
		}, "unable to update the Keycloak policy")
		return errors.NewInternalError(ctx, errs.Wrap(err, "unable to update the Keycloak policy"))
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		log.Error(ctx, map[string]interface{}{
			"client_id":       clientID,
			"policy":          policy,
			"response_status": res.Status,
			"response_body":   rest.ReadBody(res.Body),
		}, "unable to update the Keycloak policy")
		return errors.NewInternalError(ctx, errs.New("unable to update the Keycloak policy. Response status: "+res.Status+". Responce body: "+rest.ReadBody(res.Body)))
	}

	return nil
}

// GetEntitlement obtains Entitlement for specific resource.
// If entitlementResource == nil then Entitlement for all resources available to the user is returned.
// Returns (nil, nil) if response status == Forbiden which means the user doesn't have permissions to obtain Entitlement
func GetEntitlement(ctx context.Context, entitlementEndpoint string, entitlementResource *EntitlementResource, userAccesToken string) (*string, error) {
	var req *http.Request
	var reqErr error
	if entitlementResource != nil {
		b, err := json.Marshal(entitlementResource)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"entitlement_resource": entitlementResource,
				"err": err.Error(),
			}, "unable to marshal keycloak entitlement resource struct")
			return nil, errors.NewInternalError(ctx, errs.Wrap(err, "unable to marshal keycloak entitlement resource struct"))
		}

		req, reqErr = http.NewRequest("POST", entitlementEndpoint, strings.NewReader(string(b)))
		req.Header.Add("Content-Type", "application/json")
	} else {
		req, reqErr = http.NewRequest("GET", entitlementEndpoint, nil)
	}
	if reqErr != nil {
		log.Error(ctx, map[string]interface{}{
			"err": reqErr.Error(),
		}, "unable to create http request")
		return nil, errors.NewInternalError(ctx, errs.Wrap(reqErr, "unable to create http request"))
	}

	req.Header.Add("Authorization", "Bearer "+userAccesToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"entitlement_resource": entitlementResource,
			"err": err.Error(),
		}, "unable to obtain entitlement resource")
		return nil, errors.NewInternalError(ctx, errs.Wrap(err, "unable to obtain entitlement resource"))
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
		// OK
	case http.StatusForbidden:
		return nil, nil
	default:
		log.Error(ctx, map[string]interface{}{
			"entitlement_resource": entitlementResource,
			"response_status":      res.Status,
			"response_body":        rest.ReadBody(res.Body),
		}, "unable to update the Keycloak permission")
		return nil, errors.NewInternalError(ctx, errs.New("unable to obtain entitlement resource. Response status: "+res.Status+". Responce body: "+rest.ReadBody(res.Body)))
	}
	jsonString := rest.ReadBody(res.Body)

	var r entitlementResult
	err = json.Unmarshal([]byte(jsonString), &r)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"entitlement_resource": entitlementResource,
			"json_string":          jsonString,
		}, "unable to unmarshal json with the obtain entitlement request result")
		return nil, errors.NewInternalError(ctx, errs.Wrapf(err, "error when unmarshal json with the obtain entitlement request result %s ", jsonString))
	}

	return &r.Rpt, nil
}

// GetUserInfo gets user info from Keycloak
func GetUserInfo(ctx context.Context, userInfoEndpoint string, userAccessToken string) (*UserInfo, error) {
	req, err := http.NewRequest("GET", userInfoEndpoint, nil)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err.Error(),
		}, "unable to create http request")
		return nil, errors.NewInternalError(ctx, errs.Wrap(err, "unable to create http request"))
	}
	req.Header.Add("Authorization", "Bearer "+userAccessToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err.Error(),
		}, "unable to get user info from Keycloak")
		return nil, errors.NewInternalError(ctx, errs.Wrap(err, "unable to get user info from Keycloak"))
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		log.Error(ctx, map[string]interface{}{}, "unable to get user info from Keycloak")
		return nil, errors.NewInternalError(ctx, errs.New("unable to get user info from Keycloak. Response status: "+res.Status+". Responce body: "+rest.ReadBody(res.Body)))
	}
	jsonString := rest.ReadBody(res.Body)

	var r UserInfo
	err = json.Unmarshal([]byte(jsonString), &r)
	if err != nil {
		return nil, errors.NewInternalError(ctx, errs.Wrapf(err, "error when unmarshal json with user info payload: \"%s\" ", jsonString))
	}

	return &r, nil
}

// ValidateKeycloakUser returns true if the user exists in Keycloak. Returns false if the user is not found
func ValidateKeycloakUser(ctx context.Context, adminEndpoint string, userID, protectionAPIToken string) (bool, error) {
	req, err := http.NewRequest("GET", adminEndpoint+"/users/"+userID, nil)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err.Error(),
		}, "unable to create http request")
		return false, errors.NewInternalError(ctx, errs.Wrap(err, "unable to create http request"))
	}
	req.Header.Add("Authorization", "Bearer "+protectionAPIToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"user_id": userID,
			"err":     err.Error(),
		}, "unable to get user from Keycloak")
		return false, errors.NewInternalError(ctx, errs.Wrap(err, "unable to get user from Keycloak"))
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		log.Error(ctx, map[string]interface{}{
			"user_id": userID,
		}, "unable to get user from Keycloak")
		return false, errors.NewInternalError(ctx, errs.New("unable to get user from Keycloak. Response status: "+res.Status+". Responce body: "+rest.ReadBody(res.Body)))
	}
}

// GetProtectedAPIToken obtains a Protected API Token (PAT) from Keycloak
func GetProtectedAPIToken(ctx context.Context, openidConnectTokenURL string, clientID string, clientSecret string) (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	res, err := client.PostForm(openidConnectTokenURL, url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"grant_type":    {"client_credentials"},
	})
	if err != nil {
		return "", errors.NewInternalError(ctx, errs.Wrap(err, "error when obtaining token"))
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
		// OK
	case http.StatusUnauthorized:
		return "", errors.NewUnauthorizedError(res.Status + " " + rest.ReadBody(res.Body))
	case http.StatusBadRequest:
		return "", errors.NewBadParameterError(rest.ReadBody(res.Body), nil)
	default:
		return "", errors.NewInternalError(ctx, errs.New(res.Status+" "+rest.ReadBody(res.Body)))
	}

	token, err := ReadToken(ctx, res)
	if err != nil {
		return "", err
	}
	return *token.AccessToken, nil
}

// ReadToken extracts json with token data from the response
func ReadToken(ctx context.Context, res *http.Response) (*Token, error) {
	// Read the json out of the response body
	buf := new(bytes.Buffer)
	io.Copy(buf, res.Body)
	jsonString := strings.TrimSpace(buf.String())

	var token Token
	err := json.Unmarshal([]byte(jsonString), &token)
	if err != nil {
		return nil, errors.NewInternalError(ctx, errs.Wrapf(err, "error when unmarshal json with access token %s ", jsonString))
	}
	return &token, nil
}
