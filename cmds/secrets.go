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
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"strings"

	"github.com/fabric8io/gofabric8/client"
	"github.com/fabric8io/gofabric8/util"
	oclient "github.com/openshift/origin/pkg/client"
	tapi "github.com/openshift/origin/pkg/template/api"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"k8s.io/kubernetes/pkg/api"
	k8sclient "k8s.io/kubernetes/pkg/client"
	"k8s.io/kubernetes/pkg/fields"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/runtime"
)

type Keypair struct {
	pub  []byte
	priv []byte
}

func NewCmdSecrets(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secrets",
		Short: "Set up Secrets on your Kubernetes or OpenShift environment",
		Long:  `set up Secrets on your Kubernetes or OpenShift environment`,
		PreRun: func(cmd *cobra.Command, args []string) {
			showBanner()
		},
		Run: func(cmd *cobra.Command, args []string) {
			c, cfg := client.NewClient(f)
			ns, _, _ := f.DefaultNamespace()
			util.Info("Setting up secrets on your ")
			util.Success(string(util.TypeOfMaster(c)))
			util.Info(" installation at ")
			util.Success(cfg.Host)
			util.Info(" in namespace ")
			util.Successf("%s\n\n", ns)

			if confirmAction(cmd.Flags()) {
				typeOfMaster := util.TypeOfMaster(c)

				if typeOfMaster == util.Kubernetes {
					util.Fatal("Support for Kubernetes not yet available...\n")
				} else {
					oc, _ := client.NewOpenShiftClient(cfg)
					t := getTemplates(oc, ns)

					count := 0
					// get all the Templates and find the annotations on any Pods
					for _, i := range t.Items {
						// convert TemplateList.Objects to Kubernetes resources
						_ = runtime.DecodeList(i.Objects, api.Scheme, runtime.UnstructuredJSONScheme)
						for _, rc := range i.Objects {
							switch rc := rc.(type) {
							case *api.ReplicationController:
								for secretType, secretDataIdentifiers := range rc.Spec.Template.Annotations {
									count += createAndPrintSecrets(secretDataIdentifiers, secretType, c, f, cmd.Flags())
								}
							}
						}
					}

					if count == 0 {
						util.Info("No secrets created as no fabric8 secrets annotations found in the templates\n")
						util.Info("For more details see: https://github.com/fabric8io/fabric8/blob/master/docs/secretAnnotations.md\n")
					}
				}
			}
		},
	}
	cmd.PersistentFlags().BoolP("print-import-folder-structure", "", true, "Prints the folder structures that are being used by the template annotations to import secrets")
	cmd.PersistentFlags().BoolP("print-generated-keys", "", false, "Print any generated secrets to the console")
	cmd.PersistentFlags().BoolP("generate-secrets-data", "g", true, "Generate secrets data if secrets cannot be found on the local filesystem")
	return cmd
}

func createSecret(c *k8sclient.Client, f *cmdutil.Factory, flags *flag.FlagSet, secretDataIdentifiers string, secretType string, keysNames []string) (Result, error) {
	var secret = secret(secretDataIdentifiers, secretType, keysNames, flags)
	ns, _, err := f.DefaultNamespace()
	if err != nil {
		return Failure, err
	}
	rs, err := c.Secrets(ns).Create(&secret)
	if rs != nil {
		return Success, err
	}
	return Failure, err
}

func createAndPrintSecrets(secretDataIdentifiers string, secretType string, c *k8sclient.Client, fa *cmdutil.Factory, flags *flag.FlagSet) int {
	count := 0
	// check to see if multiple public and private keys are needed
	var dataType = strings.Split(secretType, "/")
	switch dataType[1] {
	case "secret-ssh-key":
		items := strings.Split(secretDataIdentifiers, ",")
		for i := range items {
			var name = items[i]
			r, err := createSecret(c, fa, flags, name, secretType, nil)
			printResult(name+" secret", r, err)
			if err == nil {
				count++
			}
		}
	case "secret-ssh-public-key":
		// if this is just a public key then the secret name is at the start of the string
		f := func(c rune) bool {
			return c == ',' || c == '[' || c == ']'
		}
		secrets := strings.FieldsFunc(secretDataIdentifiers, f)
		numOfSecrets := len(secrets)

		var keysNames []string
		if numOfSecrets > 0 {
			// if multiple secrets
			for i := 1; i < numOfSecrets; i++ {
				keysNames = append(keysNames, secrets[i])
			}
		} else {
			// only single secret required
			keysNames[0] = "ssh-key.pub"
		}

		r, err := createSecret(c, fa, flags, secrets[0], secretType, keysNames)

		printResult(secrets[0]+" secret", r, err)
		if err == nil {
			count++
		}

	default:
		gpgKeyName := []string{"gpg.conf", "secring.gpg", "pubring.gpg", "trustdb.gpg"}
		r, err := createSecret(c, fa, flags, secretDataIdentifiers, secretType, gpgKeyName)
		printResult(secretDataIdentifiers+" secret", r, err)
		if err == nil {
			count++
		}
	}
	return count
}

func secret(name string, secretType string, keysNames []string, flags *flag.FlagSet) api.Secret {
	return api.Secret{
		ObjectMeta: api.ObjectMeta{
			Name: name,
		},
		Type: api.SecretType(secretType),
		Data: getSecretData(secretType, name, keysNames, flags),
	}
}

func check(e error) {
	if e != nil {
		util.Warnf("Warning: %s\n", e)
	}
}

func logSecretImport(file string) {
	util.Infof("Importing secret: %s\n", file)
}

func getSecretData(secretType string, name string, keysNames []string, flags *flag.FlagSet) map[string][]byte {
	var dataType = strings.Split(secretType, "/")
	var data = make(map[string][]byte)

	switch dataType[1] {
	case "secret-ssh-key":
		if flags.Lookup("print-import-folder-structure").Value.String() == "true" {
			logSecretImport(name + "/ssh-key")
			logSecretImport(name + "/ssh-key.pub")
		}

		sshKey, err1 := ioutil.ReadFile(name + "/ssh-key")
		sshKeyPub, err2 := ioutil.ReadFile(name + "/ssh-key.pub")

		// if we cant find the public and private key to import, and generation flag is set then lets generate the keys
		if (err1 != nil && err2 != nil) && flags.Lookup("generate-secrets-data").Value.String() == "true" {
			util.Info("No secrets found on local filesystem, generating SSH public and private key pair\n")
			keypair := generateSshKeyPair(flags.Lookup("print-generated-keys").Value.String())
			data["ssh-key"] = keypair.priv
			data["ssh-key.pub"] = keypair.pub

		} else if (err1 != nil || err2 != nil) && flags.Lookup("generate-secrets-data").Value.String() == "true" {
			util.Infof("Found some keys to import but with errors so unable to generate SSH public and private key pair. %s\n", name)
			check(err1)
			check(err2)
		} else {
			// if we're not generating the keys and there's an error importing them then still create the secret but with empty data
			check(err1)
			check(err2)

			data["ssh-key"] = sshKey
			data["ssh-key.pub"] = sshKeyPub
		}
		return data

	case "secret-ssh-public-key":

		for i := 0; i < len(keysNames); i++ {
			if flags.Lookup("print-import-folder-structure").Value.String() == "true" {
				logSecretImport(name + "/" + keysNames[i])
			}

			sshPub, err := ioutil.ReadFile(name + "/" + keysNames[i])
			// if we cant find the public key to import and generation flag is set then lets generate the key
			if (err != nil) && flags.Lookup("generate-secrets-data").Value.String() == "true" {
				util.Info("No secrets found on local filesystem, generating SSH public key\n")
				keypair := generateSshKeyPair(flags.Lookup("print-generated-keys").Value.String())
				data[keysNames[i]] = keypair.pub

			} else {
				// if we're not generating the keys and there's an error importing them then still create the secret but with empty data
				check(err)
				data[keysNames[i]] = sshPub
			}
		}
		return data

	case "secret-gpg-key":
		for i := 0; i < len(keysNames); i++ {
			if flags.Lookup("print-import-folder-structure").Value.String() == "true" {
				logSecretImport(name + "/" + keysNames[i])
			}
			gpg, err := ioutil.ReadFile(name + "/" + keysNames[i])
			check(err)

			data[keysNames[i]] = gpg
		}

	default:
		util.Fatalf("No matching data type %s\n", dataType)
	}
	return data
}

func generateSshKeyPair(logGeneratedKeys string) Keypair {

	priv, err := rsa.GenerateKey(rand.Reader, 2014)
	if err != nil {
		util.Fatalf("Error generating key", err)
	}
	err = priv.Validate()
	if err != nil {
		util.Fatalf("Validation failed.", err)
	}

	// Get der format. priv_der []byte
	priv_der := x509.MarshalPKCS1PrivateKey(priv)

	// pem.Block
	// blk pem.Block
	priv_blk := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   priv_der,
	}

	// Resultant private key in PEM format.
	// priv_pem string
	priv_pem := string(pem.EncodeToMemory(&priv_blk))

	if logGeneratedKeys == "true" {
		util.Infof(priv_pem)
	}

	// Public Key generation
	pub := priv.PublicKey
	pub_der, err := x509.MarshalPKIXPublicKey(&pub)
	if err != nil {
		util.Fatalf("Failed to get der format for PublicKey.", err)
	}

	pub_blk := pem.Block{
		Type:    "PUBLIC KEY",
		Headers: nil,
		Bytes:   pub_der,
	}
	pub_pem := string(pem.EncodeToMemory(&pub_blk))
	if logGeneratedKeys == "true" {
		util.Infof(pub_pem)
	}

	return Keypair{
		pub:  []byte(pub_pem),
		priv: []byte(priv_pem),
	}
}

func getTemplates(c *oclient.Client, ns string) *tapi.TemplateList {

	rc, err := c.Templates(ns).List(labels.Everything(), fields.Everything())
	if err != nil {
		util.Fatalf("No Templates found in namespace %s\n", ns)
	}
	return rc
}
