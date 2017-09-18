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
	k8api "k8s.io/kubernetes/pkg/api/unversioned"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

type e2eSecretFlags struct {
	confirm      bool
	namespace    string
	username     string
	password     string
	secretName   string
	platformKind string
	disableChe   bool
}

// NewCmdE2ESecret creates/updates a Secret for running E2E tests
func NewCmdE2ESecret(f cmdutil.Factory) *cobra.Command {
	p := &e2eSecretFlags{}
	cmd := &cobra.Command{
		Use:     "e2e-secret",
		Short:   "Creates or updates a Secret for the E2E tests",
		Long:    `Creates or updates a Secret for the E2E tests`,
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
	flags.StringVarP(&p.secretName, "secret", "", "", "the name of the Secret to create/update")
	flags.StringVarP(&p.platformKind, "platform", "", "fabric8-openshift", "the kind of platform to run against. Either `osio`, `fabric8-openshift` or `fabric8-kubernetes`")
	flags.BoolVarP(&p.disableChe, "disable-che", "", true, "should we disable che tests in the generated test Secret")
	return cmd
}

func (p *e2eSecretFlags) runTest(f cmdutil.Factory) error {
	c, _ := client.NewClient(f)
	initSchema()

	ns := p.namespace
	if len(ns) == 0 {
		ns, _, _ = f.DefaultNamespace()
	}
	if len(ns) == 0 {
		return fmt.Errorf("No namespace is defined and no namespace specified!")
	}

	user := p.username
	pwd := p.password
	if len(user) == 0 {
		return fmt.Errorf("No --user parameter specified!")
	}
	if len(pwd) == 0 {
		return fmt.Errorf("No --password parameter specified!")
	}
	name := p.secretName
	if len(name) == 0 {
		name = "e2e-for-" + user
	}

	selector, err := k8api.LabelSelectorAsSelector(
		&k8api.LabelSelector{MatchLabels: map[string]string{"test": "e2e"}})
	if err != nil {
		return err
	}

	secrets, err := c.Secrets(ns).List(api.ListOptions{LabelSelector: selector})
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
			updatedSecret = secret
			create = false
			break
		}
	}

	// now lets create the script and replace it
	if updatedSecret.Data == nil {
		updatedSecret.Data = map[string][]byte{}
	}
	updatedSecret.Data["script"] = p.createSecretScript()

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
	cheFlag := "false"
	if p.disableChe {
		cheFlag = "true"
	}
	script := fmt.Sprintf(`export USERNAME=%s
export PASSWORD=%s
export DISABLE_CHE=%s
export TARGET_PLATFORM=%s
`, p.username, p.password, cheFlag, p.platformKind)
	return []byte(script)
}
