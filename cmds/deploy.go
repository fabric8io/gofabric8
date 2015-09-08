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
	"archive/zip"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fabric8io/gofabric8/client"
	"github.com/fabric8io/gofabric8/util"
	oclient "github.com/openshift/origin/pkg/client"
	"github.com/openshift/origin/pkg/cmd/admin/policy"
	"github.com/openshift/origin/pkg/cmd/server/bootstrappolicy"
	"github.com/openshift/origin/pkg/template"
	tapi "github.com/openshift/origin/pkg/template/api"
	tapiv1 "github.com/openshift/origin/pkg/template/api/v1"
	"github.com/openshift/origin/pkg/template/generator"
	"github.com/spf13/cobra"
	"k8s.io/kubernetes/pkg/api"
	kapi "k8s.io/kubernetes/pkg/api"
	k8sclient "k8s.io/kubernetes/pkg/client"
	kcmd "k8s.io/kubernetes/pkg/kubectl/cmd"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/runtime"
)

const (
	consoleMetadataUrl           = "https://repo1.maven.org/maven2/io/fabric8/apps/base/maven-metadata.xml"
	baseConsoleUrl               = "https://repo1.maven.org/maven2/io/fabric8/apps/base/%[1]s/base-%[1]s-kubernetes.json"
	consoleKubernetesMetadataUrl = "https://repo1.maven.org/maven2/io/fabric8/apps/console-kubernetes/maven-metadata.xml"
	baseConsoleKubernetesUrl     = "https://repo1.maven.org/maven2/io/fabric8/apps/console-kubernetes/%[1]s/console-kubernetes-%[1]s-kubernetes.json"
	templatesDistroUrl           = "https://repo1.maven.org/maven2/io/fabric8/apps/distro/%[1]s/distro-%[1]s-templates.zip"

	Fabric8SCC    = "fabric8"
	PrivilegedSCC = "privileged"
	RestrictedSCC = "restricted"
)

type createFunc func(c *k8sclient.Client, f *cmdutil.Factory, name string) (Result, error)

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

			if confirmAction(cmd.Flags()) {
				v := cmd.Flags().Lookup("version").Value.String()

				typeOfMaster := util.TypeOfMaster(c)
				v = f8Version(v, typeOfMaster)

				util.Warnf("\nStarting deployment of %s...\n\n", v)

				if typeOfMaster == util.Kubernetes {
					uri := fmt.Sprintf(baseConsoleKubernetesUrl, v)
					filenames := []string{uri}

					createCmd := cobra.Command{}
					createCmd.Flags().StringSlice("filename", filenames, "")
					err := kcmd.RunCreate(f, &createCmd, ioutil.Discard)
					if err != nil {
						printResult("fabric8 console", Failure, err)
					} else {
						printResult("fabric8 console", Success, nil)
					}
				} else {
					oc, _ := client.NewOpenShiftClient(cfg)

					r, err := verifyRestrictedSecurityContextConstraints(c, f)
					printResult("SecurityContextConstraints restricted", r, err)
					r, err = deployFabric8SecurityContextConstraints(c, f)
					printResult("SecurityContextConstraints fabric8", r, err)

					printAddClusterRoleToUser(oc, f, "cluster-admin", "system:serviceaccount:default:fabric8")
					printAddClusterRoleToUser(oc, f, "cluster-reader", "system:serviceaccount:default:metrics")

					printAddServiceAccount(c, f, "metrics")
					printAddServiceAccount(c, f, "router")


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

					printError("Install templates", installTemplates(oc, f, v))

					domain := cmd.Flags().Lookup(domainFlag).Value.String()

					printError("Create routes", createRoutesForDomain(ns, domain, c, oc, f))
				}
			}
		},
	}
	cmd.PersistentFlags().StringP("domain", "d", defaultDomain(), "The domain name to append to the service name to access web applications")
	return cmd
}

func installTemplates(c *oclient.Client, fac *cmdutil.Factory, v string) error {
	ns, _, err := fac.DefaultNamespace()
	if err != nil {
		util.Fatal("No default namespace")
		return err
	}
	templates := c.Templates(ns)

	util.Infof("Downloading templates for fabric8 version %v\n", v)
	uri := fmt.Sprintf(templatesDistroUrl, v)
	resp, err := http.Get(uri)
	if err != nil {
		util.Fatalf("Cannot get fabric8 template to deploy: %v", err)
	}
	defer resp.Body.Close()

	tmpFileName := "/tmp/fabric8-template-distros.tar.gz"
	t, err := os.OpenFile(tmpFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		return err
	}
	defer t.Close()

	_, err = io.Copy(t, resp.Body)
	if err != nil {
		return err
	}

	r, err := zip.OpenReader(tmpFileName)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		mode := f.FileHeader.Mode()
		if mode.IsDir() {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		util.Infof("Loading template %s\n", f.Name)
		jsonData, err := ioutil.ReadAll(rc)
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
			return err
		}

		name := tmpl.ObjectMeta.Name
		_, err = templates.Get(name)
		if err == nil {
			err = templates.Delete(name)
			if err != nil {
				util.Errorf("Could not delete template %s due to: %v\n", name, err)
			}
		}
		_, err = templates.Create(&tmpl)
		if err != nil {
			util.Fatalf("Failed to create template %v", err)
			return err
		}
	}
	return nil
}

func deployFabric8SecurityContextConstraints(c *k8sclient.Client, f *cmdutil.Factory) (Result, error) {
	name := Fabric8SCC
	scc := kapi.SecurityContextConstraints{
		ObjectMeta: kapi.ObjectMeta{
			Name: name,
		},
		AllowPrivilegedContainer: true,
		AllowHostNetwork: true,
		AllowHostPorts: true,
		AllowHostDirVolumePlugin: true,
		SELinuxContext: kapi.SELinuxContextStrategyOptions{
			Type: kapi.SELinuxStrategyRunAsAny,
		},
		RunAsUser: kapi.RunAsUserStrategyOptions{
			Type: kapi.RunAsUserStrategyRunAsAny,
		},
		Users:  []string{"system:serviceaccount:openshift-infra:build-controller", "system:serviceaccount:default:default", "system:serviceaccount:default:fabric8", "system:serviceaccount:default:jenkins", "system:serviceaccount:default:router"},
		Groups: []string{bootstrappolicy.ClusterAdminGroup, bootstrappolicy.NodesGroup},
	}
	ns, _, err := f.DefaultNamespace()
	if err != nil {
		util.Fatal("No default namespace")
		return Failure, err
	}
	_, err = c.SecurityContextConstraints().Get(name)
	if err == nil {
		err = c.SecurityContextConstraints().Delete(name)
		if err != nil {
			return Failure, err
		}
	}
	_, err = c.SecurityContextConstraints().Create(&scc)
	if err != nil {
		util.Fatalf("Cannot create SecurityContextConstraints: %v\n", err)
		util.Fatalf("Failed to create SecurityContextConstraints %v in namespace %s: %v\n", scc, ns, err)
		return Failure, err
	}
	util.Infof("SecurityContextConstraints %s is setup correctly\n", name)
	return Success, err
}

// Ensure that the `restricted` SecurityContextConstraints has the RunAsUser set to RunAsAny
//
// if `restricted does not exist lets create it
// otherwise if needed lets modify the RunAsUser
func verifyRestrictedSecurityContextConstraints(c *k8sclient.Client, f *cmdutil.Factory) (Result, error) {
	name := RestrictedSCC
	ns, _, e := f.DefaultNamespace()
	if e != nil {
		util.Fatal("No default namespace")
		return Failure, e
	}
	rc, err := c.SecurityContextConstraints().Get(name)
	if err != nil {
		scc := kapi.SecurityContextConstraints{
			ObjectMeta: kapi.ObjectMeta{
				Name: RestrictedSCC,
			},
			SELinuxContext: kapi.SELinuxContextStrategyOptions{
				Type: kapi.SELinuxStrategyMustRunAs,
			},
			RunAsUser: kapi.RunAsUserStrategyOptions{
				Type: kapi.RunAsUserStrategyRunAsAny,
			},
			Groups: []string{bootstrappolicy.AuthenticatedGroup},
		}

		_, err = c.SecurityContextConstraints().Create(&scc)
		if err != nil {
			return Failure, err
		} else {
			util.Infof("SecurityContextConstraints %s created\n", name)
			return Success, err
		}
	}

	// lets check that the restricted is configured correctly
	if kapi.RunAsUserStrategyRunAsAny != rc.RunAsUser.Type {
		rc.RunAsUser.Type = kapi.RunAsUserStrategyRunAsAny
		_, err = c.SecurityContextConstraints().Update(rc)
		if err != nil {
			util.Fatalf("Failed to update SecurityContextConstraints %v in namespace %s: %v\n", rc, ns, err)
			return Failure, err
		}
		util.Infof("SecurityContextConstraints %s is updated to enable fabric8\n", name)
	} else {
		util.Infof("SecurityContextConstraints %s is configured correctly\n", name)
	}
	return Success, err
}

func printAddServiceAccount(c *k8sclient.Client, f *cmdutil.Factory, name string) (Result, error) {
	r, err := addServiceAccount(c, f, name)
	message := fmt.Sprintf("addServiceAccount %s", name)
	printResult(message, r, err)
	return r, err
}

func addServiceAccount(c *k8sclient.Client, f *cmdutil.Factory, name string) (Result, error) {
	ns, _, e := f.DefaultNamespace()
	if e != nil {
		util.Fatal("No default namespace")
		return Failure, e
	}
	sas := c.ServiceAccounts(ns)
	_, err := sas.Get(name)
	if err != nil {
		sa := kapi.ServiceAccount{
			ObjectMeta: kapi.ObjectMeta{
				Name: name,
			},
		}
		_, err = sas.Create(&sa)
	}
	r := Success
	if err != nil {
		r = Failure
	}
	return r, err
}

func printAddClusterRoleToUser(c *oclient.Client, f *cmdutil.Factory, roleName string, userName string) (Result, error) {
	err := addClusterRoleToUser(c, f, roleName, userName)
	message := fmt.Sprintf("addClusterRoleToUser %s %s", roleName, userName)
	r := Success
	if err != nil {
		r = Failure
	}
	printResult(message, r, err)
	return r, err
}

// simulates: oadm policy add-cluster-role-to-user roleName userName
func addClusterRoleToUser(c *oclient.Client, f *cmdutil.Factory, roleName string, userName string) error {
	namespace, _, err := f.DefaultNamespace()
	if err != nil {
		util.Info("No namespace!'\n")
		return err
	}
	options := policy.RoleModificationOptions{
		RoleName:            roleName,
		RoleBindingAccessor: policy.NewLocalRoleBindingAccessor(namespace, c),
		Users:               []string{userName},
	}
	return options.AddRole()
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
