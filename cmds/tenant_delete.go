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
	"fmt"

	"github.com/fabric8io/gofabric8/client"
	"github.com/fabric8io/gofabric8/util"
	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

type tenantDeleteFlags struct {
	cmd  *cobra.Command
	args []string

	confirm bool
	tenant  string
}

func NewCmdTenantDelete(f cmdutil.Factory) *cobra.Command {
	p := &tenantDeleteFlags{}
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Deletes all your tenant resources",
		Run: func(cmd *cobra.Command, args []string) {
			p.cmd = cmd
			p.args = args
			if cmd.Flags().Lookup(yesFlag).Value.String() == "true" {
				p.confirm = true
			}
			handleError(p.tenantDelete(f))
		},
	}
	flags := cmd.Flags()
	flags.StringVarP(&p.tenant, "tenant", "t", "", "the name of the tenant to delete. If blank it will be discovered")
	return cmd
}

func (p *tenantDeleteFlags) tenantDelete(f cmdutil.Factory) error {
	c, cfg := client.NewClient(f)
	ns, _, _ := f.DefaultNamespace()
	oc, _ := client.NewOpenShiftClient(cfg)
	initSchema()

	var err error
	userNS := p.tenant
	if len(userNS) == 0 {
		userNS, err = detectCurrentUserNamespace(ns, c, oc)
		if err != nil {
			return err
		}
	}
	if !p.confirm {
		confirm := ""
		util.Warn("WARNING this command will delete all of your tenant resource, pipelines, jobs, and apps\n")
		util.Info("for your tenant: ")
		util.Successf("%s", userNS)
		util.Warn("\nContinue [y/N]: ")
		fmt.Scanln(&confirm)
		if confirm != "y" {
			util.Warn("Aborted\n")
			return nil
		}
	}

	util.Info("Deleting tenant: ")
	util.Successf("%s\n\n", userNS)

	ocCLI := "oc"
	stageNS := fmt.Sprintf("%s-stage", userNS)
	runNS := fmt.Sprintf("%s-run", userNS)
	cheNS := fmt.Sprintf("%s-che", userNS)
	jenkinsNS := fmt.Sprintf("%s-jenkins", userNS)

	// zap jenkins resources
	util.Infof("Removing jenkins namespace resources in %s\n", jenkinsNS)
	err = runCommand(ocCLI, "delete", "all", "--all", "-n", jenkinsNS, "--cascade=true", "--grace-period=-50")
	if err != nil {
		return nil
	}
	err = runCommand(ocCLI, "delete", "pvc", "--all", "-n", jenkinsNS, "--cascade=true", "--grace-period=-50")
	if err != nil {
		return nil
	}
	err = runCommand(ocCLI, "delete", "cm", "--all", "-n", jenkinsNS, "--cascade=true", "--grace-period=-50")
	if err != nil {
		return nil
	}
	err = runCommand(ocCLI, "delete", "sa", "--all", "-n", jenkinsNS, "--cascade=true", "--grace-period=-50")
	if err != nil {
		return nil
	}
	err = runCommand(ocCLI, "delete", "secret", "--all", "-n", jenkinsNS, "--cascade=true", "--grace-period=-50")
	if err != nil {
		return nil
	}

	// zap other projects
	projectsToRemove := []string{stageNS, runNS, cheNS, userNS}
	for _, ns := range projectsToRemove {
		util.Infof("Removing project %s\n", ns)
		err = runCommand(ocCLI, "delete", "project", ns, "--cascade=true", "--grace-period=-50")
		if err != nil {
			return nil
		}
	}
	util.Infof("Tenant %s now deleted.\n", userNS)
	util.Infof("Now please Update your Tenant via: https://github.com/openshiftio/openshift.io/wiki/FAQ#how-do-i-update-my-tenant-\n\n")
	return nil
}
