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

	"github.com/fabric8io/gofabric8/util"
	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

const ()

// NewCmdDelete deletes the current local cluster
func NewCmdDeleteCluster(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Delete a local cluster",
		Long:  `Delete a local cluster. This command deletes the VM and removes all`,

		Run: func(cmd *cobra.Command, args []string) {
			context, err := util.GetCurrentContext()
			if err != nil {
				util.Fatalf("Error getting current context %s", err)
			}
			var command string
			var cargs []string

			if context == util.Minikube {
				command = "minikube"
				cargs = []string{"delete"}

			} else if util.IsMiniShift(context) {
				command = "minishift"
				cargs = []string{"delete"}
			}

			if command == "" {
				util.Fatalf("Context %s not supported.  Currently only Minishift and Minikube are supported by this command\n", context)
			}

			e := exec.Command(command, cargs...)
			e.Stdout = os.Stdout
			e.Stderr = os.Stderr
			err = e.Run()
			if err != nil {
				util.Fatalf("Unable to delete the cluster %s", err)
			}

		},
	}

	return cmd
}
