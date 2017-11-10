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
	api "k8s.io/kubernetes/pkg/api"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

type e2eConsoleFlags struct {
	confirm   bool
	namespace string
	url       string
}

// NewCmdE2eConsole generates the environment variables for an E2E test on a cluster
func NewCmdE2eConsole(f cmdutil.Factory) *cobra.Command {
	p := &e2eConsoleFlags{}
	cmd := &cobra.Command{
		Use:     "e2e-console",
		Short:   "Points the jenkins namespace at the console to use for E2E tests",
		Long:    `Points the jenkins namespace at the console to use for E2E tests`,
		Aliases: []string{"e2e-console-url"},

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
	flags.StringVarP(&p.url, "url", "", "", "the console URL to use. If not specified it will be found from the fabric8 service in a namespace")
	return cmd
}

func (p *e2eConsoleFlags) runTest(f cmdutil.Factory) error {
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

	names, err := getNamespacesOrProjects(c, oc)
	if err != nil {
		return err
	}

	url := p.url
	linkNs := ns
	spaceLink, err := c.ConfigMaps(ns).Get("fabric8-space-link")
	if spaceLink == nil || err != nil {
		for _, name := range names {
			spaceLink, err = c.ConfigMaps(name).Get("fabric8-space-link")
			if spaceLink != nil && err == nil {
				linkNs = name
				break
			}
		}
	}
	if err == nil && spaceLink != nil && spaceLink.Data != nil && len(url) == 0 {
		url = spaceLink.Data["fabric8-console-url"]
	}
	consoleLink, err := c.ConfigMaps(ns).Get("fabric8-console-link")
	if consoleLink == nil || err != nil {
		consoleLink, err = c.ConfigMaps(linkNs).Get("fabric8-console-link")
	}
	if err != nil {
		consoleLink = nil
	}
	if consoleLink != nil && consoleLink.Data != nil && len(url) == 0 {
		url = consoleLink.Data["fabric8-console-url"]
	}
	if len(url) == 0 {
		url = GetServiceURL(ns, "fabric8", c)
		if len(url) == 0 {
			for _, name := range names {
				url = GetServiceURL(name, "fabric8", c)
				if len(url) > 0 {
					break
				}
			}
			if len(url) == 0 {
				return fmt.Errorf("Could not find a service called fabric8 in any of these namespaces %v", names)
			}
		}
	}
	if spaceLink == nil {
		return fmt.Errorf("Could not find a ConfigMap called `fabric8-space-link` in any of these namespaces %v", names)
	}
	consoleLinkCreate := false
	consoleLinkOperation := "update"
	if consoleLink == nil {
		consoleLinkCreate = true
		consoleLinkOperation = "create"
		consoleLink = &api.ConfigMap{
			ObjectMeta: api.ObjectMeta{
				Name:      "fabric8-console-link",
				Namespace: linkNs,
			},
		}
	}
	if consoleLink.Data == nil {
		consoleLink.Data = map[string]string{}
	}
	consoleLink.Data["fabric8-console-url"] = url
	if consoleLinkCreate {
		_, err = c.ConfigMaps(linkNs).Create(consoleLink)
	} else {
		_, err = c.ConfigMaps(linkNs).Update(consoleLink)
	}
	if err != nil {
		return fmt.Errorf("Could not %s a ConfigMap called `fabric8-console-link` in namespace %s due to %v", consoleLinkOperation, linkNs, err)
	}
	fmt.Printf("Updated ConfigMap fabric8-space-link in namespace %s to %s\n", linkNs, url)
	return nil
}
