package auth

import (
	"context"

	"fmt"

	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/goadesign/goa"
	"github.com/satori/go.uuid"
)

// AuthzResourceManager represents a space resource manager
type AuthzResourceManager interface {
	CreateResource(ctx context.Context, request *goa.RequestData, name string, rType string, uri *string, scopes *[]string, userID string) (*Resource, error)
	DeleteResource(ctx context.Context, request *goa.RequestData, resource Resource) error
}

// KeycloakResourceManager implements AuthzResourceManager interface
type KeycloakResourceManager struct {
	configuration KeycloakConfiguration
}

// Resource represents a Keycloak resource and associated permission and policy
type Resource struct {
	ResourceID   string
	PolicyID     string
	PermissionID string
}

// KeycloakConfiguration represents a keycloak configuration
type KeycloakConfiguration interface {
	GetKeycloakEndpointAuthzResourceset(*goa.RequestData) (string, error)
	GetKeycloakEndpointToken(*goa.RequestData) (string, error)
	GetKeycloakEndpointClients(*goa.RequestData) (string, error)
	GetKeycloakEndpointAdmin(*goa.RequestData) (string, error)
	GetKeycloakEndpointEntitlement(*goa.RequestData) (string, error)
	GetKeycloakClientID() string
	GetKeycloakSecret() string
}

// NewKeycloakResourceManager constructs KeycloakResourceManager
func NewKeycloakResourceManager(config KeycloakConfiguration) *KeycloakResourceManager {
	return &KeycloakResourceManager{config}
}

// CreateResource creates a keycloak resource and associated permission and policy
func (m *KeycloakResourceManager) CreateResource(ctx context.Context, request *goa.RequestData, name string, rType string, uri *string, scopes *[]string, userID string) (*Resource, error) {
	pat, err := getPat(ctx, request, m.configuration)
	if err != nil {
		return nil, err
	}
	publicClientID := m.configuration.GetKeycloakClientID()
	clientsEndpoint, err := m.configuration.GetKeycloakEndpointClients(request)
	if err != nil {
		return nil, err
	}
	clientID, err := GetClientID(context.Background(), clientsEndpoint, publicClientID, pat)
	if err != nil {
		return nil, err
	}
	authzEndpoint, err := m.configuration.GetKeycloakEndpointAuthzResourceset(request)
	if err != nil {
		return nil, err
	}
	adminEndpoint, err := m.configuration.GetKeycloakEndpointAdmin(request)
	// Create resource
	kcResource := KeycloakResource{
		Name:   name,
		Type:   rType,
		URI:    uri,
		Scopes: scopes,
	}
	resourceID, err := CreateResource(ctx, kcResource, authzEndpoint, pat)
	if err != nil {
		return nil, err
	}

	// Create policy
	found, err := ValidateKeycloakUser(ctx, adminEndpoint, userID, pat)
	if err != nil {
		return nil, err
	}
	if !found {
		log.Error(ctx, map[string]interface{}{
			"user_id": userID,
		}, "User not found in Keycloak")
		return nil, errors.NewNotFoundError("keycloak user", userID) // The user is not found in the Keycloak user base
	}
	userIDs := "[\"" + userID + "\"]"
	policy := KeycloakPolicy{
		Name:             fmt.Sprintf("%s-%s", name, uuid.NewV4().String()),
		Type:             PolicyTypeUser,
		Logic:            PolicyLogicPossitive,
		DecisionStrategy: PolicyDecisionStrategyUnanimous,
		Config: PolicyConfigData{
			UserIDs: userIDs,
		},
	}
	policyID, err := CreatePolicy(ctx, clientsEndpoint, clientID, policy, pat)
	if err != nil {
		return nil, err
	}

	// Create permission
	permission := KeycloakPermission{
		Name:             fmt.Sprintf("%s-%s", name, uuid.NewV4().String()),
		Type:             PermissionTypeResource,
		Logic:            PolicyLogicPossitive,
		DecisionStrategy: PolicyDecisionStrategyUnanimous,
		Config: PermissionConfigData{
			Resources:     "[\"" + resourceID + "\"]",
			ApplyPolicies: "[\"" + policyID + "\"]",
		},
	}
	permissionID, err := CreatePermission(ctx, clientsEndpoint, clientID, permission, pat)
	if err != nil {
		return nil, err
	}

	newResource := &Resource{
		ResourceID:   resourceID,
		PolicyID:     policyID,
		PermissionID: permissionID,
	}

	return newResource, nil
}

func getPat(ctx context.Context, requestData *goa.RequestData, config KeycloakConfiguration) (string, error) {
	endpoint, err := config.GetKeycloakEndpointToken(requestData)
	if err != nil {
		return "", err
	}
	token, err := GetProtectedAPIToken(ctx, endpoint, config.GetKeycloakClientID(), config.GetKeycloakSecret())
	if err != nil {
		return "", err
	}
	return token, nil
}

// DeleteResource deletes the keycloak resource and associated permission and policy
func (m *KeycloakResourceManager) DeleteResource(ctx context.Context, request *goa.RequestData, resource Resource) error {
	authzEndpoint, err := m.configuration.GetKeycloakEndpointAuthzResourceset(request)
	if err != nil {
		return err
	}
	clientsEndpoint, err := m.configuration.GetKeycloakEndpointClients(request)
	if err != nil {
		return err
	}
	pat, err := getPat(ctx, request, m.configuration)
	if err != nil {
		return err
	}
	publicClientID := m.configuration.GetKeycloakClientID()
	clientID, err := GetClientID(context.Background(), clientsEndpoint, publicClientID, pat)
	if err != nil {
		return err
	}

	// Delete resource
	err = DeleteResource(ctx, resource.ResourceID, authzEndpoint, pat)
	if err != nil {
		return err
	}
	// Delete permission
	err = DeletePermission(ctx, clientsEndpoint, clientID, resource.PermissionID, pat)
	if err != nil {
		return err
	}
	// Delete policy
	err = DeletePolicy(ctx, clientsEndpoint, clientID, resource.PolicyID, pat)
	return err
}
