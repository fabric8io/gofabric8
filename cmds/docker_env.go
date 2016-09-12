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

	"github.com/fabric8io/gofabric8/client"
	"github.com/fabric8io/gofabric8/util"
	"github.com/spf13/cobra"
	"k8s.io/kubernetes/pkg/api"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

const (
	rhelcdk = "rhel-cdk"
)

// NewCmdDockerEnv sets the current
func NewCmdDockerEnv(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docker-env",
		Short: "Sets up docker env variables; Usage 'eval $(gofabric8 docker-env)'",
		Long:  `Sets up docker env variables; Usage 'eval $(gofabric8 docker-env)'`,

		Run: func(cmd *cobra.Command, args []string) {
			c, _ := client.NewClient(f)

			nodes, err := c.Nodes().List(api.ListOptions{})
			if err != nil {
				util.Errorf("Unable to find any nodes: %s\n", err)
			}
			if len(nodes.Items) == 1 {
				node := nodes.Items[0]
				var command string
				var args []string

				if node.Name == minikubeNodeName {
					command = "minikube"
					args = []string{"docker-env"}

				} else if node.Name == minishiftNodeName {
					command = "minishift"
					args = []string{"docker-env"}

				} else if node.Name == rhelcdk {
					command = "vagrant"
					args = []string{"service-manager", "env", "docker"}
				}

				if command == "" {
					util.Fatalf("Unrecognised cluster environment for node %s\n", node.Name)
					util.Fatalf("docker-env support is currently only for CDK, Minishift and Minikube\n")

				}

				e := exec.Command(command, args...)
				e.Stdout = os.Stdout
				e.Stderr = os.Stderr
				err = e.Run()
				if err != nil {
					util.Fatalf("Unable to set the docker environment %v", err)
				}
			} else {
				util.Fatalf("docker-env is only available to run on clusters of 1 node")
			}

		},
	}

	return cmd
}
