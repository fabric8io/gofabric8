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
	oclient "github.com/openshift/origin/pkg/client"
	"github.com/spf13/cobra"
	k8sclient "k8s.io/kubernetes/pkg/client/unversioned"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

type cleanUpAppsFlags struct {
	confirm bool
}

// NewCmdCleanUpApps deletes all the tenant apps
func NewCmdCleanUpApps(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apps",
		Short: "Hard delete all of your tenant applications",
		Long:  `Hard delete all of your tenant applications"`,

		Run: func(cmd *cobra.Command, args []string) {
			p := cleanUpAppsFlags{}
			if cmd.Flags().Lookup(yesFlag).Value.String() == "true" {
				p.confirm = true
			}
			err := p.cleanApps(f)
			if err != nil {
				util.Fatalf("%s\n", err)
			}
			return
		},
	}
	return cmd
}

func (p *cleanUpAppsFlags) cleanApps(f *cmdutil.Factory) error {
	c, cfg := client.NewClient(f)
	ns, _, _ := f.DefaultNamespace()
	oc, _ := client.NewOpenShiftClient(cfg)
	initSchema()

	userNS, err := detectCurrentUserNamespace(ns, c, oc)
	if err != nil {
		return err
	}
	stageNS := fmt.Sprintf("%s-stage", userNS)
	runNS := fmt.Sprintf("%s-run", userNS)

	if !p.confirm {
		confirm := ""
		util.Warn("WARNING this is destructive and will remove all of your apps!\n")
		util.Info("for your tenant: ")
		util.Successf("%s", userNS)
		util.Info(" running in namespaces: ")
		util.Success(stageNS)
		util.Info(" & ")
		util.Successf("%s\n", runNS)
		util.Warn("\nContinue [y/N]: ")
		fmt.Scanln(&confirm)
		if confirm != "y" {
			util.Warn("Aborted\n")
			return nil
		}
	}

	typeOfMaster := util.TypeOfMaster(c)
	openshift := false
	namespaces := []string{stageNS, runNS, userNS}
	for _, ns := range namespaces {
		util.Info("Cleaning apps running in namespace: ")
		util.Successf("%s\n", ns)

		if typeOfMaster == util.OpenShift {
			openshift = true
			err = cleanUpAllOpenshiftResources(c, oc, ns)
			if err != nil {
				return err
			}
		}
		isUserNS := ns == userNS
		err = cleanUpAllKubernetesResources(c, ns, openshift, isUserNS)
		if err != nil {
			return err
		}
		fmt.Println("")
	}
	return nil
}

func cleanUpAllOpenshiftResources(c *k8sclient.Client, oc *oclient.Client, ns string) error {
	ocCmd := "oc"
	err := runCommand(ocCmd, "delete", "dc", "--all", "--ignore-not-found=true", "-n", ns)
	if err != nil {
		return err
	}
	err = runCommand(ocCmd, "delete", "bc", "--all", "--ignore-not-found=true", "-n", ns)
	if err != nil {
		return err
	}
	err = runCommand(ocCmd, "delete", "build", "--all", "--ignore-not-found=true", "-n", ns)
	if err != nil {
		return err
	}
	err = runCommand(ocCmd, "delete", "route", "--all", "--ignore-not-found=true", "-n", ns)
	return err
}

func cleanUpAllKubernetesResources(c *k8sclient.Client, ns string, openshift bool, isUserNS bool) error {
	ocCmd := "oc"
	err := runCommand(ocCmd, "delete", "deployment", "--all", "--ignore-not-found=true", "-n", ns)
	if err != nil {
		return err
	}
	err = runCommand(ocCmd, "delete", "rs", "--all", "--ignore-not-found=true", "-n", ns)
	if err != nil {
		return err
	}
	err = runCommand(ocCmd, "delete", "rc", "--all", "--ignore-not-found=true", "-n", ns)
	if err != nil {
		return err
	}
	err = runCommand(ocCmd, "delete", "pod", "--all", "--ignore-not-found=true", "-n", ns)
	if err != nil {
		return err
	}
	if !openshift {
		err = runCommand(ocCmd, "delete", "ingress", "--all", "--ignore-not-found=true", "-n", ns)
		if err != nil {
			return err
		}
	}
	if isUserNS {
		return err
	}
	err = runCommand(ocCmd, "delete", "cm", "--all", "--ignore-not-found=true", "-n", ns)
	if err != nil {
		return err
	}

	err = runCommand(ocCmd, "delete", "service", "--all", "--ignore-not-found=true", "-n", ns)
	if err != nil {
		return err
	}
	/*
		TODO lets not delete secrets or SAs for now to avoid removing tenant stuff like SAs: builder, default & deployer

		err = runCommand(ocCmd, "delete", "secret", "--all", "--ignore-not-found=true", "-n", ns)
		if err != nil {
			return err
		}
		err = runCommand(ocCmd, "delete", "sa", "--all", "--ignore-not-found=true", "-n", ns)
		if err != nil {
			return err
		}
	*/
	return err
}
