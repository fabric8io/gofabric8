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
package main

import (
	commands "github.com/fabric8io/gofabric8/cmds"
	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

func runHelp(cmd *cobra.Command, args []string) {
	cmd.Help()
}

func main() {
	cmds := &cobra.Command{
		Use:   "gofabric8",
		Short: "gofabric8 is used to validate & deploy fabric8 components on to your Kubernetes or OpenShift environment",
		Long: `gofabric8 is used to validate & deploy fabric8 components on to your Kubernetes or OpenShift environment
								Find more information at http://fabric8.io.`,
		Run: runHelp,
	}

	cmds.PersistentFlags().String("fabric8-version", "latest", "fabric8 version")
	cmds.PersistentFlags().BoolP("yes", "y", false, "assume yes")

	f := cmdutil.NewFactory(nil)
	f.BindFlags(cmds.PersistentFlags())

	cmds.AddCommand(commands.NewCmdValidate(f))
	cmds.AddCommand(commands.NewCmdDeploy(f))
	cmds.AddCommand(commands.NewCmdPull(f))
	cmds.AddCommand(commands.NewCmdIngress(f))
	cmds.AddCommand(commands.NewCmdRoutes(f))
	cmds.AddCommand(commands.NewCmdSecrets(f))
	cmds.AddCommand(commands.NewCmdVolume(f))
	cmds.AddCommand(commands.NewCmdVolumes(f))
	cmds.AddCommand(commands.NewCmdVersion())

	cmds.Execute()
}
