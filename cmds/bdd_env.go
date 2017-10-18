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
	"strings"

	"k8s.io/kubernetes/pkg/api"
	"github.com/fabric8io/gofabric8/client"
	"github.com/fabric8io/gofabric8/util"
	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

type bddEnvFlags struct {
	confirm   bool
	tenantNamespace string
	jenkinsNamespace string
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
	flags.StringVarP(&p.tenantNamespace, "tenant-namespace", "", "", "the tenant namespace to use")
	flags.StringVarP(&p.jenkinsNamespace, "jenkins-namespace", "", "", "the jenkins namespace to use")
	return cmd
}

func (p *bddEnvFlags) runTest(f cmdutil.Factory) error {
	c, cfg, err := client.NewDefaultClient(f)
	if err != nil {
		c, cfg = client.NewClient(f)
	}
	oc, _ := client.NewOpenShiftClient(cfg)

	initSchema()

	jenkinsNs := p.jenkinsNamespace
	names, err := getNamespacesOrProjects(c, oc)
	if err != nil {
		return err
	}
	url := ""
	if len(jenkinsNs) == 0 {
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
		url = GetServiceURL(jenkinsNs, "jenkins", c)
		if len(url) == 0 {
			return fmt.Errorf("Could not find a service called jenkins in namespace %s", jenkinsNs)
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


	githubUser := ""
	githubPassword := ""
	tenantNs := p.tenantNamespace
	if len(tenantNs) == 0 {
		for _, name := range names {
			secret, err := c.Secrets(name).Get("cd-github")
			if err == nil {
				githubUser = secretDataField(secret, "username")
				githubPassword = secretDataField(secret, "password")
			}
		}
		if len(url) == 0 {
			return fmt.Errorf("Could not find a service called jenkins in any of these namespaces %v", names)
		}
	} else {
		secret, err := c.Secrets(tenantNs).Get("cd-github")
		if err == nil {
			githubUser = secretDataField(secret, "username")
			githubPassword = secretDataField(secret, "password")
		}
	}

	fmt.Printf("export BDD_JENKINS_URL=\"%s\"\n", url)
	fmt.Printf("export BDD_JENKINS_USERNAME=\"%s\"\n", username)
	fmt.Printf("export BDD_JENKINS_BEARER_TOKEN=\"%s\"\n", token)
	if len(githubUser) > 0 {
		fmt.Printf("export GITHUB_USER=\"%s\"\n", githubUser)
	}
	if len(githubPassword) > 0 {
		fmt.Printf("export GITHUB_PASSWORD=\"%s\"\n", githubPassword)
	}
	return nil
}

func secretDataField(secret *api.Secret, name string) string {
	if secret.Data != nil {
		data := secret.Data[name]
		if data != nil {
			return string(data)
		}
	}
	return ""
}