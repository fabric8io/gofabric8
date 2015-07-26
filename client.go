package main

import (
	"github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/kubectl/cmd/util"
)

func newClient(f *util.Factory) *client.Client {
	var err error
	k8sConfig, err = f.ClientConfig()
	if err != nil {
		fatalf("Could not initialise a client config: %v", err)
	}
	k8sClient, err = client.New(k8sConfig)
	if err != nil {
		fatalf("Could not initialise a client: %v", err)
	}

	discoverInstallationType()

	return k8sClient
}
