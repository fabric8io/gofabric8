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
	"k8s.io/kubernetes/pkg/api"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

type e2eSecretFlags struct {
	confirm        bool
	namespace      string
	username       string
	password       string
	secretName     string
	osUsername     string
	osToken        string
	githubUsername string
	githubPassword string
}

// NewCmdE2ESecret creates/updates a Secret for running E2E tests
func NewCmdE2ESecret(f cmdutil.Factory) *cobra.Command {
	p := &e2eSecretFlags{}
	cmd := &cobra.Command{
		Use:     "e2e-secret",
		Short:   "Creates or updates a Secret for the user for E2E tests",
		Long:    `Creates or updates a Secret for the user for E2E tests`,
		Aliases: []string{"e2e-secrets"},

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
	flags.StringVarP(&p.username, "user", "u", "", "the username to test with")
	flags.StringVarP(&p.password, "password", "p", "", "the password to test with")
	flags.StringVarP(&p.secretName, "secret", "", "default-test-user", "the name of the Secret to create/update")
	flags.StringVarP(&p.osUsername, "os-user", "", "", "the name of the OpenShift/Kubernetes Username to use")
	flags.StringVarP(&p.osToken, "os-token", "", "", "the Kubernetes/OpenShift OAuth token to access to the cluster")
	flags.StringVarP(&p.githubUsername, "github-user", "", "", "the GitHub username")
	flags.StringVarP(&p.githubPassword, "github-password", "", "", "the GitHub Personal Access Token or Password")
	return cmd
}

func (p *e2eSecretFlags) runTest(f cmdutil.Factory) error {
	c, cfg := client.NewClient(f)
	oc, _ := client.NewOpenShiftClient(cfg)

	initSchema()

	ns := p.namespace
	if len(ns) == 0 {
		// lets try find the namespace with jenkins inside
		names, err := getNamespacesOrProjects(c, oc)
		if err != nil {
			return err
		}
		for _, name := range names {
			_, err := c.ConfigMaps(name).Get("jenkins")
			if err == nil {
				ns = name
				break
			}
		}
	}
	if len(ns) == 0 {
		ns, _, _ = f.DefaultNamespace()
		util.Warnf("No namespace specified and could not find the jenkins namespace (which has a ConfigMap called jenkins) so defaulting to namespace: %s", ns)
	}
	typeOfMaster := util.TypeOfMaster(c)

	user := p.username
	pwd := p.password
	if len(user) == 0 || len(pwd) == 0 || len(p.osUsername) == 0 {
		if typeOfMaster == util.OpenShift {
			mini, err := util.IsMini()
			if err != nil {
				util.Failuref("error checking if minikube or minishift %v", err)
			}
			if mini {
				if len(user) == 0 {
					user = "developer"
				}
				if len(pwd) == 0 {
					pwd = "developer"
				}
				if len(p.osUsername) == 0 {
					p.osUsername = "developer"
				}
			}
		} else {
			if len(user) == 0 {
				user = p.githubUsername
			}
			if len(pwd) == 0 {
				pwd = p.githubPassword
			}
		}
	}

	if len(user) == 0 {
		return fmt.Errorf("No --user parameter specified!")
	}
	if len(pwd) == 0 {
		return fmt.Errorf("No --password parameter specified!")
	}
	name := p.secretName
	if len(name) == 0 {
		name = "default-test-user"
	}

	if typeOfMaster == util.Kubernetes {
		if len(p.githubUsername) == 0 {
			p.githubUsername = p.username
		}
		if len(p.githubPassword) == 0 {
			p.githubPassword = p.password
		}
	} else {
		if len(p.githubUsername) == 0 {
			return fmt.Errorf("No --github-user parameter specified!")
		}
		if len(p.githubPassword) == 0 {
			return fmt.Errorf("No --github-password parameter specified!")
		}

		if len(p.osToken) == 0 {
			// TODO load from ~/.kube/config?
		}
	}
	if len(p.osUsername) == 0 {
		p.osUsername = p.githubUsername

		if len(p.osUsername) == 0 {
			return fmt.Errorf("No --os-user parameter specified!")
		}
	}

	secrets, err := c.Secrets(ns).List(api.ListOptions{})
	if err != nil {
		return fmt.Errorf("Failed to load secrets in namespace %s due to %s", ns, err)
	}

	create := true
	updatedSecret := api.Secret{
		ObjectMeta: api.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"provider": "fabric8",
				"test":     "e2e",
			},
		},
	}
	for _, secret := range secrets.Items {
		if secret.Name == name {
			updatedSecret.ObjectMeta = secret.ObjectMeta
			create = false
			break
		}
	}

	// now lets create the script and replace it
	if updatedSecret.Data == nil {
		updatedSecret.Data = map[string][]byte{}
	}
	updatedSecret.Data["user"] = []byte(user)
	updatedSecret.Data["password"] = []byte(pwd)
	updatedSecret.Data["os-user"] = []byte(p.osUsername)
	updatedSecret.Data["github-user"] = []byte(p.githubUsername)
	updatedSecret.Data["github-password"] = []byte(p.githubPassword)
	updatedSecret.Data["os-token"] = []byte(p.osToken)

	secretResource := c.Secrets(ns)
	if create {
		_, err = secretResource.Create(&updatedSecret)
		if err != nil {
			return fmt.Errorf("Failed to create secret %s in namespace %s due to %s", name, ns, err)
		}
		util.Infof("Created secret %s/%s\n", ns, name)
	} else {
		_, err = secretResource.Update(&updatedSecret)
		if err != nil {
			return fmt.Errorf("Failed to update secret %s in namespace %s due to %s", name, ns, err)
		}
		util.Infof("Updated secret %s/%s\n", ns, name)
	}
	return nil
}

func (p *e2eSecretFlags) createSecretScript() []byte {
	script := fmt.Sprintf(`export USERNAME=%s
export PASSWORD=%s
`, p.username, p.password)
	return []byte(script)
}
