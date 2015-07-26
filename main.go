package main

import (
	"encoding/json"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	cmdutil "github.com/GoogleCloudPlatform/kubernetes/pkg/kubectl/cmd/util"
	"github.com/spf13/cobra"
)

const (
	openshift  masterType = "OpenShift"
	kubernetes masterType = "Kubernetes"

	successResult result = "✔"
	failureResult result = "✘"
)

var (
	k8sClient    *client.Client
	k8sConfig    *client.Config
	typeOfMaster masterType = kubernetes
)

func discoverInstallationType() {
	res, err := k8sClient.Get().AbsPath("").DoRaw()
	if err != nil {
		fatalf("Could not discover the type of your installation: %v", err)
	}

	var rp api.RootPaths
	err = json.Unmarshal(res, &rp)
	if err != nil {
		fatalf("Could not discover the type of your installation: %v", err)
	}
	for _, p := range rp.Paths {
		if p == "/oapi" {
			typeOfMaster = openshift
			return
		}
	}
}

func runHelp(cmd *cobra.Command, args []string) {
	cmd.Help()
}

func main() {
	cmds := &cobra.Command{
		Use:   "fabric8",
		Short: "fabric8 is used to validate & deploy fabric8 components on to your Kubernetes or OpenShift environment",
		Long: `fabric8 is used to validate & deploy fabric8 components on to your Kubernetes or OpenShift environment
								Find more information at http://fabric8.io.`,
		Run: runHelp,
	}

	f := cmdutil.NewFactory(nil)
	f.BindFlags(cmds.PersistentFlags())

	cmds.AddCommand(newCmdValidate(f))

	cmds.Execute()
}
