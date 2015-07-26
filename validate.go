package main

import (
	"strings"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/kubectl/cmd/util"
	"github.com/spf13/cobra"
)

func newCmdValidate(f *util.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate your Kubernetes or OpenShift environment",
		Long:  `validate your Kubernetes or OpenShift environment`,
		Run: func(cmd *cobra.Command, args []string) {
			newClient(f)
			infof("Validating your ")
			success(string(typeOfMaster))
			infof(" installation at ")
			success(k8sConfig.Host)
			blank()
			blank()
			validateResult("Hello", successResult)
			validateResult("Goodbye", failureResult)
			blank()
			fatal("Failed to validate your environment")
		},
	}

	return cmd
}

func validateResult(check string, r result) {
	infof("%s%s", check, strings.Repeat(".", 24-len(check)))
	if r == failureResult {
		failuref("%s", r)
	} else {
		successf("%s", r)
	}
	blank()
}
