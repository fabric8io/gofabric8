package che

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"

	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/goadesign/goa/middleware"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
)

// NewStarterClient is a helper function to create a new CheStarter client
// Uses http.DefaultClient
func NewStarterClient(cheStarterURL, openshiftMasterURL string, namespace string) *StarterClient {
	return &StarterClient{cheStarterURL: cheStarterURL, openshiftMasterURL: openshiftMasterURL, namespace: namespace, client: http.DefaultClient}
}

// StarterClient describes the REST interface between Platform and Che Starter
type StarterClient struct {
	cheStarterURL      string
	openshiftMasterURL string
	namespace          string
	client             *http.Client
}

func (cs *StarterClient) targetURL(resource string) string {
	return fmt.Sprintf("%v/%v?masterUrl=%v&namespace=%v", cs.cheStarterURL, resource, cs.openshiftMasterURL, cs.namespace)
}

func (cs *StarterClient) setHeaders(ctx context.Context, req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+goajwt.ContextJWT(ctx).Raw)
	req.Header.Set(middleware.RequestIDHeader, middleware.ContextRequestID(ctx))
}

// ListWorkspaces lists the available workspaces for a given user
func (cs *StarterClient) ListWorkspaces(ctx context.Context, repository string) ([]*WorkspaceResponse, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf(cs.targetURL("workspace")+"&repository=%v", repository), nil)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"repository": repository,
			"err":        err,
		}, "failed to create request object")
		return nil, err
	}
	cs.setHeaders(ctx, req)

	if log.IsDebug() {
		b, _ := httputil.DumpRequest(req, true)
		log.Debug(ctx, map[string]interface{}{
			"request": string(b),
		}, "request object")
	}

	resp, err := cs.client.Do(req)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"repository": repository,
			"err":        err,
		}, "failed to list workspace for repository")
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		workspaceErr := WorkspaceError{}
		err = json.NewDecoder(resp.Body).Decode(&workspaceErr)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"repository": repository,
				"err":        err,
			}, "failed to decode error response from list workspace for repository")
			return nil, err
		}
		log.Error(ctx, map[string]interface{}{
			"repository": repository,
			"err":        workspaceErr.String(),
		}, "failed to execute list workspace for repository")
		return nil, &workspaceErr
	}

	workspaceResp := []*WorkspaceResponse{}
	err = json.NewDecoder(resp.Body).Decode(&workspaceResp)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"repository": repository,
			"err":        err,
		}, "failed to decode response from list workspace for repository")
		return nil, err
	}
	return workspaceResp, nil
}

// CreateWorkspace creates a new Che Workspace based on a repository
func (cs *StarterClient) CreateWorkspace(ctx context.Context, workspace WorkspaceRequest) (*WorkspaceResponse, error) {
	body, err := json.Marshal(&workspace)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"workspace_id":       workspace.Name,
			"workspace_stack_id": workspace.StackID,
			"workspace":          workspace,
			"err":                err,
		}, "failed to create request object")
		return nil, err
	}

	req, err := http.NewRequest("POST", cs.targetURL("workspace"), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	cs.setHeaders(ctx, req)

	if log.IsDebug() {
		b, _ := httputil.DumpRequest(req, true)
		log.Debug(ctx, map[string]interface{}{
			"request": string(b),
		}, "request object")
	}

	resp, err := cs.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		workspaceErr := WorkspaceError{}
		err = json.NewDecoder(resp.Body).Decode(&workspaceErr)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"workspace_id":       workspace.Name,
				"workspace_stack_id": workspace.StackID,
				"workspace":          workspace,
				"err":                err,
			}, "failed to decode error response from create workspace for repository")
			return nil, err
		}
		log.Error(ctx, map[string]interface{}{
			"workspace_id":       workspace.Name,
			"workspace_stack_id": workspace.StackID,
			"workspace":          workspace,
			"err":                workspaceErr.String(),
		}, "failed to execute create workspace for repository")
		return nil, &workspaceErr
	}

	workspaceResp := WorkspaceResponse{}
	err = json.NewDecoder(resp.Body).Decode(&workspaceResp)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"workspace_id":       workspace.Name,
			"workspace_stack_id": workspace.StackID,
			"workspace":          workspace,
			"err":                err,
		}, "failed to decode response from create workspace for repository")
		return nil, err
	}
	return &workspaceResp, nil
}

// StartExistingWorkspace starts an existing Che Workspace based on a repository
func (cs *StarterClient) StartExistingWorkspace(ctx context.Context, workspaceName string) (*WorkspaceResponse, error) {
	log.Debug(ctx, map[string]interface{}{
		"workspace_id": workspaceName,
	}, "starting an existing workspace")

	req, err := http.NewRequest("PATCH", cs.targetURL(fmt.Sprintf("workspace/%s", workspaceName)), nil)
	if err != nil {
		return nil, err
	}
	cs.setHeaders(ctx, req)

	if log.IsDebug() {
		b, _ := httputil.DumpRequest(req, true)
		log.Debug(ctx, map[string]interface{}{
			"request": string(b),
		}, "request object")
	}

	resp, err := cs.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		workspaceErr := WorkspaceError{}
		err = json.NewDecoder(resp.Body).Decode(&workspaceErr)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"workspace_id": workspaceName,
				"err":          err,
			}, "failed to decode error response from starting an existing workspace for repository")
			return nil, err
		}
		log.Error(ctx, map[string]interface{}{
			"workspace_id": workspaceName,
			"err":          workspaceErr.String(),
		}, "failed to execute start existing workspace for repository")
		return nil, &workspaceErr
	}

	workspaceResp := WorkspaceResponse{}
	err = json.NewDecoder(resp.Body).Decode(&workspaceResp)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"workspace_id": workspaceName,
			"err":          err,
		}, "failed to decode response from starting an existing workspace for repository")
		return nil, err
	}
	return &workspaceResp, nil
}

// WorkspaceRequest represents a create workspace request body
type WorkspaceRequest struct {
	//ID          string `json:"id,omitempty"`
	Branch      string `json:"branch,omitempty"`
	Description string `json:"description,omitempty"`
	Name        string `json:"config.name,omitempty"`
	Repository  string `json:"repo,omitempty"`
	StackID     string `json:"stackId,omitempty"`
}

// WorkspaceResponse represents a create workspace response body
type WorkspaceResponse struct {
	//ID string `json:"id,omitempty"`
	//	Branch          string `json:"branch"`
	Description string `json:"description,omitempty"`
	//	Location        string `json:"location"`
	//	Login           string `json:"login"`
	Config WorkspaceConfig `json:"config,omitempty"`
	//	Repository      string `json:"repository"`
	Status string `json:"status,omitempty"`
	//	WorkspaceIDEURL string `json:"workspaceIdeUrl"`
	Links []WorkspaceLink `json:"links,omitempty"`
}

// WorkspaceConfig represents the workspace config
type WorkspaceConfig struct {
	Name string `json:"name"`
}

// GetIDEURL return the link with rel for ide url
func (w WorkspaceResponse) GetIDEURL() string {
	for _, l := range w.Links {
		if l.Rel == "ide url" {
			return l.HRef
		}
	}
	return ""
}

// WorkspaceLink represents a URL for the location of a workspace
type WorkspaceLink struct {
	HRef   string `json:"href"`
	Method string `json:"method"`
	Rel    string `json:"rel"`
}

// WorkspaceError represent an error comming from the che-starter service
type WorkspaceError struct {
	Status    int    `json:"status"`
	ErrorMsg  string `json:"error"`
	Message   string `json:"message"`
	Timestamp string `json:"timeStamp"`
	Trace     string `json:"trace"`
}

func (err *WorkspaceError) Error() string {
	return err.ErrorMsg
}

func (err *WorkspaceError) String() string {
	return fmt.Sprintf("Status %v Error %v Message %v Trace\n%v", err.Status, err.ErrorMsg, err.ErrorMsg, err.Trace)
}
