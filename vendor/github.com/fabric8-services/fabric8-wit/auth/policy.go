package auth

import (
	"context"

	"github.com/goadesign/goa"
)

// AuthzPolicyManager represents a space collaborators policy manager
type AuthzPolicyManager interface {
	GetPolicy(ctx context.Context, request *goa.RequestData, policyID string) (*KeycloakPolicy, *string, error)
	UpdatePolicy(ctx context.Context, request *goa.RequestData, policy KeycloakPolicy, pat string) error
	AddUserToPolicy(p *KeycloakPolicy, userID string) bool
	RemoveUserFromPolicy(p *KeycloakPolicy, userID string) bool
}

// KeycloakPolicyManager implements AuthzPolicyManager interface
type KeycloakPolicyManager struct {
	configuration KeycloakConfiguration
}

// NewKeycloakPolicyManager constructs KeycloakPolicyManager
func NewKeycloakPolicyManager(config KeycloakConfiguration) *KeycloakPolicyManager {
	return &KeycloakPolicyManager{config}
}

// AddUserToPolicy adds the user ID to the policy
func (m *KeycloakPolicyManager) AddUserToPolicy(p *KeycloakPolicy, userID string) bool {
	return p.AddUserToPolicy(userID)
}

// RemoveUserFromPolicy removes the user ID from the policy
func (m *KeycloakPolicyManager) RemoveUserFromPolicy(p *KeycloakPolicy, userID string) bool {
	return p.RemoveUserFromPolicy(userID)
}

// GetPolicy obtains the space collaborators policy
func (m *KeycloakPolicyManager) GetPolicy(ctx context.Context, request *goa.RequestData, policyID string) (*KeycloakPolicy, *string, error) {
	clientsEndpoint, err := m.configuration.GetKeycloakEndpointClients(request)
	if err != nil {
		return nil, nil, err
	}
	pat, err := getPat(ctx, request, m.configuration)
	if err != nil {
		return nil, nil, err
	}
	publicClientID := m.configuration.GetKeycloakClientID()
	clientID, err := GetClientID(context.Background(), clientsEndpoint, publicClientID, pat)
	if err != nil {
		return nil, nil, err
	}

	policy, err := GetPolicy(ctx, clientsEndpoint, clientID, policyID, pat)
	if err != nil {
		return nil, nil, err
	}

	return policy, &pat, nil
}

// UpdatePolicy updates the space collaborators policy
func (m *KeycloakPolicyManager) UpdatePolicy(ctx context.Context, request *goa.RequestData, policy KeycloakPolicy, pat string) error {
	clientsEndpoint, err := m.configuration.GetKeycloakEndpointClients(request)
	if err != nil {
		return err
	}
	publicClientID := m.configuration.GetKeycloakClientID()
	clientID, err := GetClientID(context.Background(), clientsEndpoint, publicClientID, pat)
	if err != nil {
		return err
	}

	return UpdatePolicy(ctx, clientsEndpoint, clientID, policy, pat)
}
