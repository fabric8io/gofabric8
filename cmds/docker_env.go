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
	"os"
	"os/exec"
	"strings"

	"github.com/fabric8io/gofabric8/util"
	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

const (
	rhelcdk = "default/10-1-2-2:8443/admin" // seems like an odd context name, lets try it for now in the absence of anything else
)

// NewCmdDockerEnv sets the current
func NewCmdDockerEnv(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docker-env",
		Short: "Sets up docker env variables; Usage 'eval $(gofabric8 docker-env)'",
		Long:  `Sets up docker env variables; Usage 'eval $(gofabric8 docker-env)'`,

		Run: func(cmd *cobra.Command, args []string) {

			out, err := exec.Command("kubectl config current-context").Output()
			if err != nil {
				util.Fatalf("Error getting current context %v", err)
			}
			context := strings.TrimSpace(string(out))

			var command string
			var cargs []string

			if context == minikubeNodeName {
				command = "minikube"
				cargs = []string{"docker-env"}

			} else if context == minishiftNodeName {
				command = "minishift"
				cargs = []string{"docker-env"}

			} else if context == rhelcdk {
				command = "vagrant"
				cargs = []string{"service-manager", "env", "docker"}
			}

			if command == "" {
				util.Fatalf("Context %s not supported.  Currently only CDK, Minishift and Minikube are supported\n", context)
			}

			e := exec.Command(command, cargs...)
			e.Stdout = os.Stdout
			e.Stderr = os.Stderr
			err = e.Run()
			if err != nil {
				util.Fatalf("Unable to set the docker environment %v", err)
			}

		},
	}

	return cmd
}
