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

type cleanUpJenkinsParams struct {
	confirm bool
}

// NewCmdCleanUpJenkins delete files in the tenants content repository
func NewCmdCleanUpJenkins(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "jenkins",
		Short:   "Deletes all the jenkins jobs in your tenant Jenkins service",
		Long:    `Deletes all the jenkins jobs in your tenant Jenkins service`,
		Aliases: []string{"content-repository"},

		Run: func(cmd *cobra.Command, args []string) {
			p := cleanUpJenkinsParams{}
			if cmd.Flags().Lookup(yesFlag).Value.String() == "true" {
				p.confirm = true
			}
			err := p.cleanUpJenkins(f)
			if err != nil {
				util.Fatalf("%s", err)
			}
			return
		},
	}
	return cmd
}

func (p *cleanUpJenkinsParams) cleanUpJenkins(f *cmdutil.Factory) error {
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
		util.Warn("WARNING this is destructive and will remove ALL of the jenkins jobs\n")
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
	util.Info("Cleaning jenkins for tenant: ")
	util.Successf("%s", userNS)
	util.Info(" running in namespace: ")
	util.Successf("%s\n", jenkinsNS)

	err = ensureDeploymentOrDCHasReplicas(c, oc, jenkinsNS, "jenkins", 1)
	if err != nil {
		return err
	}
	pod, err := waitForReadyPodForDeploymentOrDC(c, oc, jenkinsNS, "jenkins")
	if err != nil {
		return err
	}
	util.Infof("Found running jenkins pod %s\n", pod)

	kubeCLI := "kubectl"
	err = runCommand(kubeCLI, "exec", "-it", pod, "-n", jenkinsNS, "--", "bash", "-c", "rm -rf /var/lib/jenkins/jobs/*")
	if err != nil {
		return err
	}
	err = runCommand(kubeCLI, "delete", "pod", pod, "-n", jenkinsNS)
	if err != nil {
		return err
	}
	if err == nil {
		util.Info("Completed!\n")
	}
	return err
}
