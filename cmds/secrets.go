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
	"io/ioutil"
	"net/http"
	"encoding/json"
	"math/rand"
	"time"
	"strings"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	k8sclient "github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	cmdutil "github.com/GoogleCloudPlatform/kubernetes/pkg/kubectl/cmd/util"
	tapi "github.com/openshift/origin/pkg/template/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/runtime"
	tapiv1 "github.com/openshift/origin/pkg/template/api/v1"
	"github.com/fabric8io/gofabric8/client"
	"github.com/fabric8io/gofabric8/util"
	"github.com/spf13/cobra"
	"github.com/openshift/origin/pkg/template"
	"github.com/openshift/origin/pkg/template/generator"
)

const (
	testUrl               = "https://gist.githubusercontent.com/jstrachan/ffad63dd5dcd369c9498/raw/cecceb677c9af12e0b3b762d4256565bac90fcfd/jenkins.k8s.json"
)

type createSec func(c *k8sclient.Client, f *cmdutil.Factory, name string, secretType string) (Result, error)

func NewCmdSecrets(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secrets",
		Short: "Set up Secrets on your Kubernetes or OpenShift environment",
		Long:  `set up Secrets on your Kubernetes or OpenShift environment`,
		Run: func(cmd *cobra.Command, args []string) {
			c, cfg := client.NewClient(f)
			ns, _, _ := f.DefaultNamespace()
			util.Info("Setting up secrets on your ")
			util.Success(string(util.TypeOfMaster(c)))
			util.Info(" installation at ")
			util.Success(cfg.Host)
			util.Info(" in namespace ")
			util.Successf("%s\n\n", ns)

			if cmd.Flags().Lookup("yes").Value.String() == "false" {
				util.Info("Continue? [Y/n] ")
				cont := util.AskForConfirmation(true)
				if !cont {
					util.Fatal("Cancelled...\n")
				}
			}

			resp, err := http.Get(testUrl)
			if err != nil {
				util.Fatalf("Cannot get fabric8 template to deploy: %v", err)
			}
			defer resp.Body.Close()
			jsonData, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				util.Fatalf("Cannot get fabric8 template to deploy: %v", err)
			}
			var v1tmpl tapiv1.Template
			err = json.Unmarshal(jsonData, &v1tmpl)
			if err != nil {
				util.Fatalf("Cannot get fabric8 template to deploy: %v", err)
			}
			var tmpl tapi.Template

			err = api.Scheme.Convert(&v1tmpl, &tmpl)
			if err != nil {
				util.Fatalf("Cannot get fabric8 template to deploy: %v", err)
			}

			generators := map[string]generator.Generator{
				"expression": generator.NewExpressionValueGenerator(rand.New(rand.NewSource(time.Now().UnixNano()))),
			}
			p := template.NewProcessor(generators)
			p.Process(&tmpl)

			for _, o := range tmpl.Objects {
				switch o := o.(type) {
				case *runtime.Unstructured:
					if o.Kind == "ReplicationController" {
						var (
							b []byte
							rc api.ReplicationController
						)
						b, err = json.Marshal(o.Object)
						err := json.Unmarshal(b, &rc)
						if err != nil {
							break
						}
						for secretType, secretDataIdentifiers := range rc.Spec.Template.Annotations {
							printSecretResult(secretDataIdentifiers, secretType, createSecret, c, f)
						}
					}
				}
			}
		},
	}
	return cmd
}

func createSecret(c *k8sclient.Client, f *cmdutil.Factory, secretDataIdentifiers string, secretType string) (Result, error) {
	var secret = secret(secretDataIdentifiers, secretType)
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

func printSecretResult(secretDataIdentifiers string, secretType string, v createSec, c *k8sclient.Client, f *cmdutil.Factory) {
	var items = strings.Split(secretDataIdentifiers, ",")
	for i := range items {
		var name = items[i]
		r, err := v(c, f, name, secretType)
		printResult(name + " secret", r, err)
	}
}

func secret(name string, secretType string) api.Secret {
	return api.Secret{
		ObjectMeta: api.ObjectMeta{
			Name:      name,
		},
		Type:      api.SecretType(secretType),
		Data:	getSecretData(secretType, name),
	}
}

func check(e error) {
	if e != nil {
		util.Warnf("Error file %s\n", e)
	}
}

func logSecretImport(file string){
	util.Infof("Importing secret: %s\n", file)
}

func getSecretData(secretType string, name string) map[string][]byte {
	var dataType = strings.Split(secretType, "/")
	var data = make(map[string][]byte)

	switch dataType[1] {
	case "secret-ssh-key":
		logSecretImport(name +"/id_rsa")
		idrsa, err := ioutil.ReadFile(name +"/id_rsa")
		check(err)
		logSecretImport(name +"/id_rsa.pub")
		idrsaPub, err := ioutil.ReadFile(name +"/id_rsa.pub")
		check(err)

		data["id-rsa"] = idrsa
		data["id-rsa.pub"] = idrsaPub

	case "secret-ssh-public-key":
		logSecretImport(name +"/id_rsa.pub")
		idrsaPub, err := ioutil.ReadFile(name +"/id_rsa.pub")
		check(err)

		data["id-rsa.pub"] = idrsaPub

	case "secret-gpg-key":
		logSecretImport(name +"/secring.gpg")
		gpg, err := ioutil.ReadFile(name +"/secring.gpg")
		check(err)

		data["gpg"] = gpg

	default :
		util.Fatalf("No matching data type %s\n", dataType)
	}
	return data
}
