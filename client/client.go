package client

import (
	"github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	cmdutil "github.com/GoogleCloudPlatform/kubernetes/pkg/kubectl/cmd/util"
	"github.com/fabric8io/gofabric8/util"
)

func NewClient(f *cmdutil.Factory) (*client.Client, *client.Config) {
	var err error
	cfg, err := f.ClientConfig()
	if err != nil {
		util.Fatalf("Could not initialise a client config: %v", err)
	}
	c, err := client.New(cfg)
	if err != nil {
		util.Fatalf("Could not initialise a client: %v", err)
	}

	return c, cfg
}
