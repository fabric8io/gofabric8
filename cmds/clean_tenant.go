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
	"fmt"

	"github.com/fabric8io/gofabric8/client"
	"github.com/fabric8io/gofabric8/util"
	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

type cleanUpTenantFlags struct {
	confirm bool
}

// NewCmdCleanUpTenant delete files in the tenants content repository
func NewCmdCleanUpTenant(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tenant",
		Short: "Hard delete of your tenant pipelines, apps, jobs and releases",
		Long:  `Hard delete of your tenant pipelines, apps, jobs and releases`,

		Run: func(cmd *cobra.Command, args []string) {
			p := cleanUpTenantFlags{}
			if cmd.Flags().Lookup(yesFlag).Value.String() == "true" {
				p.confirm = true
			}
			err := p.cleanTenant(f)
			if err != nil {
				util.Fatalf("%s\n", err)
			}
			return
		},
	}
	return cmd
}

func (p *cleanUpTenantFlags) cleanTenant(f *cmdutil.Factory) error {
	c, cfg := client.NewClient(f)
	ns, _, _ := f.DefaultNamespace()
	oc, _ := client.NewOpenShiftClient(cfg)
	initSchema()

	userNS, err := detectCurrentUserNamespace(ns, c, oc)
	if err != nil {
		return err
	}
	jenkinsNS := fmt.Sprintf("%s-jenkins", userNS)

	if !p.confirm {
		confirm := ""
		util.Warn("WARNING this is destructive and will remove all of your tenant pipelines, apps and releases!\n")
		util.Info("for your tenant: ")
		util.Successf("%s", userNS)
		util.Info(" running in namespace: ")
		util.Successf("%s\n", jenkinsNS)
		util.Warn("\nContinue [y/N]: ")
		fmt.Scanln(&confirm)
		if confirm != "y" {
			util.Warn("Aborted\n")
			return nil
		}
	}

	err = (&cleanUpAppsFlags{
		confirm: true,
	}).cleanApps(f)
	if err != nil {
		return err
	}
	err = (&cleanUpMavenLocalRepoFlags{
		confirm: true,
	}).cleanMavenLocalRepo(f)
	if err != nil {
		return err
	}
	fmt.Println("")

	err = (&cleanUpContentRepoFlags{
		confirm: true,
	}).cleanContentRepo(f)
	if err != nil {
		return err
	}
	fmt.Println("")

	err = (&cleanUpJenkinsFlags{
		confirm: true,
	}).cleanUpJenkins(f)
	if err != nil {
		return err
	}
	fmt.Println("")

	util.Info("\n\nCompleted cleaning the tenant resource\n")
	return nil
}
