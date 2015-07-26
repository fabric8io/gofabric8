package cmds

import (
	"strings"

	cmdutil "github.com/GoogleCloudPlatform/kubernetes/pkg/kubectl/cmd/util"
	"github.com/fabric8io/gofabric8/client"
	"github.com/fabric8io/gofabric8/util"
	"github.com/spf13/cobra"
)

type Result bool

const (
	Success Result = true
	Failure Result = false
)

func NewCmdValidate(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate your Kubernetes or OpenShift environment",
		Long:  `validate your Kubernetes or OpenShift environment`,
		Run: func(cmd *cobra.Command, args []string) {
			c, cfg := client.NewClient(f)
			util.Infof("Validating your ")
			util.Success(string(util.TypeOfMaster(c)))
			util.Infof(" installation at ")
			util.Success(cfg.Host)
			util.Blank()
			util.Blank()
			validateResult("Hello", Success)
			validateResult("Goodbye", Failure)
			util.Blank()
		},
	}

	return cmd
}

func validateResult(check string, r Result) {
	util.Infof("%s%s", check, strings.Repeat(".", 24-len(check)))
	if r == Failure {
		util.Failuref("%t", r)
	} else {
		util.Successf("%t", r)
	}
	util.Blank()
}
