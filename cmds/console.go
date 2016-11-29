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
	"github.com/fabric8io/gofabric8/client"
	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

// NewCmdConsole Open the fabric8 console
func NewCmdConsole(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "console",
		Short: "Open the fabric8 console",
		Long:  `Open the fabric8 console`,

		Run: func(cmd *cobra.Command, args []string) {
			c, _ := client.NewClient(f)
			ns, _, _ := f.DefaultNamespace()

			openService(ns, "fabric8", c, false, true)
		},
	}

	return cmd
}
