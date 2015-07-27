/**
 * Copyright (C) 2015 Red Hat, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *         http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package cmds

import (
	"strings"

	cmdutil "github.com/GoogleCloudPlatform/kubernetes/pkg/kubectl/cmd/util"
	"github.com/fabric8io/gofabric8/client"
	"github.com/fabric8io/gofabric8/util"
	"github.com/spf13/cobra"
)

type Result string

const (
	Success Result = "✔"
	Failure Result = "✘"
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
		util.Failuref("%s", r)
	} else {
		util.Successf("%s", r)
	}
	util.Blank()
}
