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

type cleanUpContentRepoFlags struct {
	confirm bool
}

// NewCmdCleanUpContentRepository delete files in the tenants content repository
func NewCmdCleanUpContentRepository(f cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "content-repo",
		Short:   "Hard delete all fabric8 apps, environments and configurations",
		Long:    `Hard delete all fabric8 apps, environments and configurations`,
		Aliases: []string{"content-repository"},

		Run: func(cmd *cobra.Command, args []string) {
			p := cleanUpContentRepoFlags{}
			if cmd.Flags().Lookup(yesFlag).Value.String() == "true" {
				p.confirm = true
			}
			err := p.cleanContentRepo(f)
			if err != nil {
				util.Fatalf("%s\n", err)
			}
			return
		},
	}
	return cmd
}

func (p *cleanUpContentRepoFlags) cleanContentRepo(f cmdutil.Factory) error {
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
		util.Warn("WARNING this is destructive and will remove ALL of the releases in your content-repository\n")
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
	util.Info("Cleaning content-repository for tenant: ")
	util.Successf("%s", userNS)
	util.Info(" running in namespace: ")
	util.Successf("%s\n", jenkinsNS)

	err = ensureDeploymentOrDCHasReplicas(c, oc, jenkinsNS, "content-repository", 1)
	if err != nil {
		return err
	}
	pod, err := waitForReadyPodForDeploymentOrDC(c, oc, jenkinsNS, "content-repository")
	if err != nil {
		return err
	}
	util.Infof("Found running content-repository pod %s\n", pod)

	kubeCLI := "kubectl"
	err = runCommand(kubeCLI, "exec", "-it", pod, "-n", jenkinsNS, "--", "rm", "-rf", "/var/www/html/content/repositories")
	if err != nil {
		return err
	}
	err = runCommand(kubeCLI, "exec", "-it", pod, "-n", jenkinsNS, "--", "du", "-hc", "/var/www/html/content")
	if err != nil {
		return err
	}
	if err == nil {
		util.Info("Completed!\n")
	}
	return err
}
