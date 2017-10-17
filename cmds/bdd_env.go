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
	"strings"

	"github.com/fabric8io/gofabric8/client"
	"github.com/fabric8io/gofabric8/util"
	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

type bddEnvFlags struct {
	confirm   bool
	namespace string
}

// NewCmdBddEnv generates the environment variables for a BDD test on a cluster
func NewCmdBddEnv(f cmdutil.Factory) *cobra.Command {
	p := &bddEnvFlags{}
	cmd := &cobra.Command{
		Use:     "bdd-env",
		Short:   "Generates the BDD environment variables for use by the BDD test pipeline",
		Long:    `Generates the BDD environment variables for use by the BDD test pipeline`,
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

func (p *bddEnvFlags) runTest(f cmdutil.Factory) error {
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
	url := ""
	if len(ns) == 0 {
		names, err := getNamespacesOrProjects(c, oc)
		if err != nil {
			return err
		}
		for _, name := range names {
			url = GetServiceOrRouteURL(name, "jenkins", c, oc, "https://")
			if len(url) > 0 {
				break
			}
		}
		if len(url) == 0 {
			return fmt.Errorf("Could not find a service called jenkins in any of these namespaces %v", names)
		}
	} else {
		url = GetServiceURL(ns, "jenkins", c)
		if len(url) == 0 {
			return fmt.Errorf("Could not find a service called jenkins in namespace %s", ns)
		}
	}

	authInfo, err := util.GetContextAuthInfo()
	if err != nil {
		return err
	}
	if authInfo == nil {
		return fmt.Errorf("Could not find the auth info in $KUBECONFIG or ~/.kube/config")
	}
	token := authInfo.Token
	//username := authInfo.Username
	username, err := runCommandWithOutput("oc", "whoami")
	if err != nil {
		return err
	}
	username = strings.TrimSpace(username)

	fmt.Printf("export BDD_JENKINS_URL=\"%s\"\n", url)
	fmt.Printf("export BDD_JENKINS_USERNAME=\"%s\"\n", username)
	fmt.Printf("export BDD_JENKINS_TOKEN=\"%s\"\n", token)
	return nil
}
