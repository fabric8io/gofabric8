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
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	kcmd "github.com/GoogleCloudPlatform/kubernetes/pkg/kubectl/cmd"
	cmdutil "github.com/GoogleCloudPlatform/kubernetes/pkg/kubectl/cmd/util"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/runtime"
	"github.com/fabric8io/gofabric8/client"
	"github.com/fabric8io/gofabric8/util"
	"github.com/openshift/origin/pkg/template"
	tapi "github.com/openshift/origin/pkg/template/api"
	tapiv1 "github.com/openshift/origin/pkg/template/api/v1"
	"github.com/openshift/origin/pkg/template/generator"
	"github.com/spf13/cobra"
)

const (
	consoleMetadataUrl           = "https://repo1.maven.org/maven2/io/fabric8/apps/base/maven-metadata.xml"
	baseConsoleUrl               = "https://repo1.maven.org/maven2/io/fabric8/apps/base/%[1]s/base-%[1]s-kubernetes.json"
	consoleKubernetesMetadataUrl = "https://repo1.maven.org/maven2/io/fabric8/apps/console-kubernetes/maven-metadata.xml"
	baseConsoleKubernetesUrl     = "https://repo1.maven.org/maven2/io/fabric8/apps/console-kubernetes/%[1]s/console-kubernetes-%[1]s-kubernetes.json"
)

func NewCmdDeploy(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy fabric8 to your Kubernetes or OpenShift environment",
		Long:  `deploy fabric8 to your Kubernetes or OpenShift environment`,
		Run: func(cmd *cobra.Command, args []string) {
			c, cfg := client.NewClient(f)
			ns, _, _ := f.DefaultNamespace()
			util.Info("Deploying fabric8 to your ")
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

			v := cmd.Flags().Lookup("version").Value.String()

			typeOfMaster := util.TypeOfMaster(c)
			v = f8Version(v, typeOfMaster)

			util.Warnf("\nStarting deployment of %s...\n\n", v)

			if typeOfMaster == util.Kubernetes {
				uri := fmt.Sprintf(baseConsoleKubernetesUrl, v)
				filenames := []string{uri}

				err := kcmd.RunCreate(f, ioutil.Discard, filenames)
				if err != nil {
					printResult("fabric8 console", Failure, err)
				} else {
					printResult("fabric8 console", Success, nil)
				}
			} else {
				uri := fmt.Sprintf(baseConsoleUrl, v)
				resp, err := http.Get(uri)
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

				tmpl.Parameters = append(tmpl.Parameters, tapi.Parameter{
					Name:  "DOMAIN",
					Value: cmd.Flags().Lookup("domain").Value.String(),
				})

				p.Process(&tmpl)

				for _, o := range tmpl.Objects {
					switch o := o.(type) {
					case *runtime.Unstructured:
						var b []byte
						b, err = json.Marshal(o.Object)
						if err != nil {
							break
						}
						req := c.Post().Body(b)
						if o.Kind != "OAuthClient" {
							req.Namespace(ns).Resource(strings.ToLower(o.TypeMeta.Kind + "s"))
						} else {
							req.AbsPath("oapi", "v1", strings.ToLower(o.TypeMeta.Kind+"s"))
						}
						res := req.Do()
						if res.Error() != nil {
							err = res.Error()
							break
						}
						var statusCode int
						res.StatusCode(&statusCode)
						if statusCode != http.StatusCreated {
							err = fmt.Errorf("Failed to create %s: %d", o.TypeMeta.Kind, statusCode)
							break
						}
					}
				}

				if err != nil {
					printResult("fabric8 console", Failure, err)
				} else {
					printResult("fabric8 console", Success, nil)
				}
			}
		},
	}

	cmd.Flags().String("domain", "vagrant.f8", "The domain name to append to the service name to access web applications")

	return cmd
}

func f8Version(v string, typeOfMaster util.MasterType) string {
	metadataUrl := consoleMetadataUrl
	if typeOfMaster == util.Kubernetes {
		metadataUrl = consoleKubernetesMetadataUrl
	}

	resp, err := http.Get(metadataUrl)
	if err != nil {
		util.Fatalf("Cannot get fabric8 version to deploy: %v", err)
	}
	defer resp.Body.Close()
	// read xml http response
	xmlData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		util.Fatalf("Cannot get fabric8 version to deploy: %v", err)
	}

	type Metadata struct {
		Release  string   `xml:"versioning>release"`
		Versions []string `xml:"versioning>versions>version"`
	}

	var m Metadata
	err = xml.Unmarshal(xmlData, &m)
	if err != nil {
		util.Fatalf("Cannot get fabric8 version to deploy: %v", err)
	}

	if v == "latest" {
		return m.Release
	}

	for _, version := range m.Versions {
		if v == version {
			return version
		}
	}

	util.Errorf("\nUnknown version: %s\n", v)
	util.Fatalf("Valid versions: %v\n", append(m.Versions, "latest"))
	return ""
}
