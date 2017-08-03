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
	"github.com/spf13/cobra"

	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

func NewCmdTenant(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tenant",
		Short: "Commands for working on your tenant",
	}
	cmd.AddCommand(NewCmdTenantCheck(f))
	cmd.AddCommand(NewCmdTenantDelete(f))
	cmd.AddCommand(NewCmdTenantUpdate(f))

	cleanCmd := NewCmdCleanUpTenant(f)
	cleanCmd.Use = "clean"
	cmd.AddCommand(cleanCmd)

	return cmd
}
