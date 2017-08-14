/*
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
	"github.com/fabric8io/gofabric8/util"
	"github.com/spf13/cobra"

	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

func NewCmdTenantUpdate(f cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Updates a Tenant (a user/team) with a set of namespaces along with Jenkins and Che",
		Run: func(cmd *cobra.Command, args []string) {
			util.Error("Not implemented yet\n")
		},
	}
	return cmd
}
