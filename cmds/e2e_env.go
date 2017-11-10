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
	"os"

	"github.com/fabric8io/gofabric8/client"
	"github.com/fabric8io/gofabric8/util"
	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

type e2eEnvFlags struct {
	confirm   bool
	namespace string
}

// NewCmdE2eEnv generates the environment variables for an E2E test on a cluster
func NewCmdE2eEnv(f cmdutil.Factory) *cobra.Command {
	p := &e2eEnvFlags{}
	cmd := &cobra.Command{
		Use:     "e2e-env",
		Short:   "Generates the E2E environment variables for use by the E2E test pipeline",
		Long:    `Generates the E2E environment variables for use by the E2E test pipeline`,
		Aliases: []string{"e2e-environment"},

		Run: func(cmd *cobra.Command, args []string) {
			if cmd.Flags().Lookup(yesFlag).Value.String() == "true" {
				p.confirm = true
			}
			err := p.runTest(f)
			if err != nil {
				util.Fatalf("%s\n", err)
			}
			return
		},
	}
	flags := cmd.Flags()
	flags.StringVarP(&p.namespace, "namespace", "", "", "the namespace to look for the fabric8 installation. Defaults to the current namespace")
	return cmd
}

func (p *e2eEnvFlags) runTest(f cmdutil.Factory) error {
	c, cfg, err := client.NewDefaultClient(f)
	if err != nil {
		c, cfg = client.NewClient(f)
	}
	oc, _ := client.NewOpenShiftClient(cfg)

	initSchema()

	ns := p.namespace
	if len(ns) == 0 {
		ns = os.Getenv("FABRIC8_SYSTEM_NAMESPACE")
	}
	if len(ns) == 0 {
		ns, _, _ = f.DefaultNamespace()
	}
	if len(ns) == 0 {
		return fmt.Errorf("No namespace is defined and no namespace specified!")
	}

	url := ""
	consoleLink, err := c.ConfigMaps(ns).Get("fabric8-console-link")
	if err == nil && consoleLink != nil && consoleLink.Data != nil {
		url = consoleLink.Data["fabric8-console-url"]
	}
	spaceLink, err := c.ConfigMaps(ns).Get("fabric8-space-link")
	if err == nil && spaceLink != nil && spaceLink.Data != nil {
		url = spaceLink.Data["fabric8-console-url"]
	}
	if len(url) == 0 {
		url = GetServiceURL(ns, "fabric8", c)
		if len(url) == 0 {
			names, err := getNamespacesOrProjects(c, oc)
			if err != nil {
				return err
			}
			for _, name := range names {
				url = GetServiceURL(name, "fabric8", c)
				if len(url) > 0 {
					break
				}
			}
			/*
			if len(url) == 0 {
				return fmt.Errorf("Could not find a service called fabric8 in any of these namespaces %v. Please try run the command `gofabric8 e2e-console` to help populate the fabric8-space-link ConfigMap with a link to the console to test against", names)
			}
			*/
		}
	}

	platform := "osio"
	typeOfMaster := util.TypeOfMaster(c)
	if typeOfMaster == util.Kubernetes {
		platform = "fabric8-kubernetes"
	} else {
		platform = "fabric8-openshift"
	}
	if len(url) > 0 {
		fmt.Printf("export TARGET_URL=\"%s\"\n", url)
	}
	fmt.Printf("export TEST_PLATFORM=\"%s\"\n", platform)
	return nil
}
