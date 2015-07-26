package util

import (
	"encoding/json"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/client"
)

type MasterType string

const (
	OpenShift  MasterType = "OpenShift"
	Kubernetes MasterType = "Kubernetes"
)

func TypeOfMaster(c *client.Client) MasterType {
	res, err := c.Get().AbsPath("").DoRaw()
	if err != nil {
		Fatalf("Could not discover the type of your installation: %v", err)
	}

	var rp api.RootPaths
	err = json.Unmarshal(res, &rp)
	if err != nil {
		Fatalf("Could not discover the type of your installation: %v", err)
	}
	for _, p := range rp.Paths {
		if p == "/oapi" {
			return OpenShift
		}
	}
	return Kubernetes
}
