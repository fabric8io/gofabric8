package main

import (
	cmdutil "github.com/GoogleCloudPlatform/kubernetes/pkg/kubectl/cmd/util"
	commands "github.com/fabric8io/gofabric8/cmds"
	"github.com/spf13/cobra"
)

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

	cmds.AddCommand(commands.NewCmdValidate(f))

	cmds.Execute()
}
