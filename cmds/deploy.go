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
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/fabric8io/gofabric8/client"
	"github.com/fabric8io/gofabric8/util"
	aapi "github.com/openshift/origin/pkg/authorization/api"
	aapiv1 "github.com/openshift/origin/pkg/authorization/api/v1"
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
	k8sclient "k8s.io/kubernetes/pkg/client/unversioned"
	kcmd "k8s.io/kubernetes/pkg/kubectl/cmd"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/runtime"
)

const (
	consoleMetadataUrl           = "io/fabric8/apps/console/maven-metadata.xml"
	baseConsoleUrl               = "io/fabric8/apps/console/%[1]s/console-%[1]s-kubernetes.json"
	consoleKubernetesMetadataUrl = "io/fabric8/apps/console-kubernetes/maven-metadata.xml"
	baseConsoleKubernetesUrl     = "io/fabric8/apps/console-kubernetes/%[1]s/console-kubernetes-%[1]s-kubernetes.json"

	devopsTemplatesDistroUrl = "io/fabric8/forge/distro/distro/%[1]s/distro-%[1]s-templates.zip"
	devOpsMetadataUrl        = "io/fabric8/forge/distro/distro/maven-metadata.xml"

	kubeflixTemplatesDistroUrl = "io/fabric8/kubeflix/distro/distro/%[1]s/distro-%[1]s-templates.zip"
	kubeflixMetadataUrl        = "io/fabric8/kubeflix/distro/distro/maven-metadata.xml"

	zipkinTemplatesDistroUrl = "io/fabric8/zipkin/packages/distro/%[1]s/distro-%[1]s-templates.zip"
	zipkinMetadataUrl        = "io/fabric8/zipkin/packages/distro/maven-metadata.xml"

	iPaaSTemplatesDistroUrl = "io/fabric8/ipaas/distro/distro/%[1]s/distro-%[1]s-templates.zip"
	iPaaSMetadataUrl        = "io/fabric8/ipaas/distro/distro/maven-metadata.xml"

	Fabric8SCC    = "fabric8"
	Fabric8SASSCC = "fabric8-sa-group"
	PrivilegedSCC = "privileged"
	RestrictedSCC = "restricted"

	runFlag             = "app"
	versioniPaaSFlag    = "version-ipaas"
	versionDevOpsFlag   = "version-devops"
	versionKubeflixFlag = "version-kubeflix"
	versionZipkinFlag   = "version-zipkin"
	mavenRepoFlag       = "maven-repo"
	dockerRegistryFlag  = "docker-registry"

	typeLabel          = "type"
	teamTypeLabelValue = "team"
)

type createFunc func(c *k8sclient.Client, f *cmdutil.Factory, name string) (Result, error)

func NewCmdDeploy(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy fabric8 to your Kubernetes or OpenShift environment",
		Long:  `deploy fabric8 to your Kubernetes or OpenShift environment`,
		PreRun: func(cmd *cobra.Command, args []string) {
			showBanner()
		},
		Run: func(cmd *cobra.Command, args []string) {
			c, cfg := client.NewClient(f)
			ns, _, _ := f.DefaultNamespace()

			domain := cmd.Flags().Lookup(domainFlag).Value.String()
			apiserver := cmd.Flags().Lookup(apiServerFlag).Value.String()
			typeOfMaster := util.TypeOfMaster(c)

			util.Info("Deploying fabric8 to your ")
			util.Success(string(typeOfMaster))
			util.Info(" installation at ")
			util.Success(cfg.Host)
			util.Info(" for domain ")
			util.Success(domain)
			util.Info(" in namespace ")
			util.Successf("%s\n\n", ns)

			mavenRepo := cmd.Flags().Lookup(mavenRepoFlag).Value.String()
			if !strings.HasSuffix(mavenRepo, "/") {
				mavenRepo = mavenRepo + "/"
			}
			util.Info("Loading fabric8 releases from maven repository:")
			util.Successf("%s\n", mavenRepo)

			dockerRegistry := cmd.Flags().Lookup(dockerRegistryFlag).Value.String()
			if len(dockerRegistry) > 0 {
				util.Infof("Loading fabric8 docker images from docker registry: %s\n", dockerRegistry)
			}

			if len(apiserver) == 0 {
				apiserver = domain
			}

			if strings.Contains(domain, "=") {
				util.Warnf("\nInvalid domain: %s\n\n", domain)
			} else if confirmAction(cmd.Flags()) {
				v := cmd.Flags().Lookup("fabric8-version").Value.String()

				consoleVersion := f8ConsoleVersion(mavenRepo, v, typeOfMaster)

				versioniPaaS := cmd.Flags().Lookup(versioniPaaSFlag).Value.String()
				versioniPaaS = versionForUrl(versioniPaaS, urlJoin(mavenRepo, iPaaSMetadataUrl))

				versionDevOps := cmd.Flags().Lookup(versionDevOpsFlag).Value.String()
				versionDevOps = versionForUrl(versionDevOps, urlJoin(mavenRepo, devOpsMetadataUrl))

				versionKubeflix := cmd.Flags().Lookup(versionKubeflixFlag).Value.String()
				versionKubeflix = versionForUrl(versionKubeflix, urlJoin(mavenRepo, kubeflixMetadataUrl))

				versionZipkin := cmd.Flags().Lookup(versionZipkinFlag).Value.String()
				versionZipkin = versionForUrl(versionZipkin, urlJoin(mavenRepo, zipkinMetadataUrl))

				util.Warnf("\nStarting fabric8 console deployment using %s...\n\n", consoleVersion)

				oc, _ := client.NewOpenShiftClient(cfg)

				aapi.AddToScheme(api.Scheme)
				aapiv1.AddToScheme(api.Scheme)
				tapi.AddToScheme(api.Scheme)
				tapiv1.AddToScheme(api.Scheme)

				if typeOfMaster == util.Kubernetes {
					uri := fmt.Sprintf(urlJoin(mavenRepo, baseConsoleKubernetesUrl), consoleVersion)
					if len(dockerRegistry) > 0 {
						jsonData, err := loadJsonDataAndAppendDockerRegistryPrefix(uri, dockerRegistry)
						if err == nil {
							tmpFileName := "/tmp/fabric8-console.json"
							t, err := os.OpenFile(tmpFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777)
							if err != nil {
								util.Fatalf("Cannot open the converted fabric8 console template file: %v", err)
							}
							defer t.Close()

							_, err = io.Copy(t, bytes.NewReader(jsonData))
							if err != nil {
								util.Fatalf("Cannot write the converted fabric8 console template file: %v", err)
							}
							uri = tmpFileName
						}
					}
					filenames := []string{uri}

					createCmd := &cobra.Command{}
					cmdutil.AddValidateFlags(createCmd)
					cmdutil.AddOutputFlagsForMutation(createCmd)
					cmdutil.AddApplyAnnotationFlags(createCmd)
					cmdutil.AddRecordFlag(createCmd)
					err := kcmd.RunCreate(f, createCmd, ioutil.Discard, &kcmd.CreateOptions{Filenames: filenames})
					if err != nil {
						printResult("fabric8 console", Failure, err)
					} else {
						printResult("fabric8 console", Success, nil)
					}
					printAddServiceAccount(c, f, "fluentd")
					printAddServiceAccount(c, f, "registry")
				} else {
					r, err := verifyRestrictedSecurityContextConstraints(c, f)
					printResult("SecurityContextConstraints restricted", r, err)
					r, err = deployFabric8SecurityContextConstraints(c, f, ns)
					printResult("SecurityContextConstraints fabric8", r, err)
					r, err = deployFabric8SASSecurityContextConstraints(c, f, ns)
					printResult("SecurityContextConstraints "+Fabric8SASSCC, r, err)

					printAddClusterRoleToUser(oc, f, "cluster-admin", "system:serviceaccount:"+ns+":fabric8")
					printAddClusterRoleToUser(oc, f, "cluster-admin", "system:serviceaccount:"+ns+":jenkins")
					printAddClusterRoleToUser(oc, f, "cluster-reader", "system:serviceaccount:"+ns+":metrics")
					printAddClusterRoleToUser(oc, f, "cluster-reader", "system:serviceaccount:"+ns+":fluentd")

					printAddServiceAccount(c, f, "fluentd")
					printAddServiceAccount(c, f, "registry")
					printAddServiceAccount(c, f, "router")

					if cmd.Flags().Lookup(templatesFlag).Value.String() == "true" {
						uri := fmt.Sprintf(urlJoin(mavenRepo, baseConsoleUrl), consoleVersion)
						jsonData, err := loadJsonDataAndAppendDockerRegistryPrefix(uri, dockerRegistry)
						if err != nil {
							printError("failed to apply docker registry prefix", err)
						}
						createTemplate(jsonData, "fabric8 console", ns, domain, apiserver, c)
					} else {
						printError("Ignoring the deploy of the fabric8 console", nil)
					}
				}
				println("Created fabric8 console")

				if cmd.Flags().Lookup(templatesFlag).Value.String() == "true" {
					println("Installing templates!")
					printError("Install DevOps templates", installTemplates(c, oc, f, versionDevOps, urlJoin(mavenRepo, devopsTemplatesDistroUrl), dockerRegistry, domain))
					printError("Install iPaaS templates", installTemplates(c, oc, f, versioniPaaS, urlJoin(mavenRepo, iPaaSTemplatesDistroUrl), dockerRegistry, domain))
					printError("Install Kubeflix templates", installTemplates(c, oc, f, versionKubeflix, urlJoin(mavenRepo, kubeflixTemplatesDistroUrl), dockerRegistry, domain))
					printError("Install Zipkin templates", installTemplates(c, oc, f, versionZipkin, urlJoin(mavenRepo, zipkinTemplatesDistroUrl), dockerRegistry, domain))
				} else {
					printError("Ignoring the deploy of templates", nil)
				}

				appToRun := cmd.Flags().Lookup(runFlag).Value.String()
				if len(appToRun) > 0 {
					util.Info("\n\nInstalling: ")
					util.Successf("%s\n\n", appToRun)
					jsonData, err := loadTemplateData(ns, appToRun, c, oc)
					if err != nil {
						printError("Failed to load app "+appToRun, err)
					}
					createTemplate(jsonData, appToRun, ns, domain, apiserver, c)
				}

				if typeOfMaster == util.Kubernetes {
					// TODO currently ingress breaks jenkins accessing gogs so lets disable for now
					//printError("Create ingress", createIngressForDomain(ns, domain, c, f))
				} else {
					printError("Create routes", createRoutesForDomain(ns, domain, c, oc, f))
				}

				// lets label the namespace/project as a developer team
				nss := c.Namespaces()
				namespace, err := nss.Get(ns)
				if err != nil {
					printError("Failed to load namespace", err)
				} else {
					if addLabelIfNotxisEt(&namespace.ObjectMeta, typeLabel, teamTypeLabelValue) {
						_, err = nss.Update(namespace)
						if err != nil {
							printError("Failed to label namespace", err)
						}
					}
				}
			}
		},
	}
	cmd.PersistentFlags().StringP("domain", "d", defaultDomain(), "The domain name to append to the service name to access web applications")
	cmd.PersistentFlags().String("api-server", "", "overrides the api server url")
	cmd.PersistentFlags().String(versioniPaaSFlag, "latest", "The version to use for the Fabric8 iPaaS templates")
	cmd.PersistentFlags().String(versionDevOpsFlag, "latest", "The version to use for the Fabric8 DevOps templates")
	cmd.PersistentFlags().String(versionKubeflixFlag, "latest", "The version to use for the Kubeflix templates")
	cmd.PersistentFlags().String(versionZipkinFlag, "latest", "The version to use for the Zipkin templates")
	cmd.PersistentFlags().String(mavenRepoFlag, "https://repo1.maven.org/maven2/", "The maven repo used to find releases of fabric8")
	cmd.PersistentFlags().String(dockerRegistryFlag, "", "The docker registry used to download fabric8 images. Typically used to point to a staging registry")
	cmd.PersistentFlags().String(runFlag, "", "The name of the fabric8 app to startup. e.g. use `--app=cd-pipeline` to run the main CI/CD pipeline app")
	cmd.PersistentFlags().Bool(templatesFlag, true, "Should the standard Fabric8 templates be installed?")
	cmd.PersistentFlags().Bool(consoleFlag, true, "Should the Fabric8 console be deployed?")
	return cmd
}

func loadTemplateData(ns string, templateName string, c *k8sclient.Client, oc *oclient.Client) ([]byte, error) {
	typeOfMaster := util.TypeOfMaster(c)
	if typeOfMaster == util.Kubernetes {
		catalogName := "catalog-" + templateName
		configMap, err := c.ConfigMaps(ns).Get(catalogName)
		if err != nil {
			return nil, err
		}
		for k, v := range configMap.Data {
			if strings.LastIndex(k, ".json") >= 0 {
				return []byte(v), nil
			}
		}
		return nil, fmt.Errorf("Could not find a key for the catalog %s which ends with `.json`", catalogName)

	} else {
		template, err := oc.Templates(ns).Get(templateName)
		if err != nil {
			return nil, err
		}
		return json.Marshal(template)
	}
	return nil, nil
}

func createTemplate(jsonData []byte, templateName string, ns string, domain string, apiserver string, c *k8sclient.Client) {
	var v1tmpl tapiv1.Template
	err := json.Unmarshal(jsonData, &v1tmpl)
	if err != nil {
		util.Fatalf("Cannot get %s template to deploy. error: %v\ntemplate: %s", templateName, err, string(jsonData))
	}
	var tmpl tapi.Template

	err = api.Scheme.Convert(&v1tmpl, &tmpl)
	if err != nil {
		util.Fatalf("Cannot convert %s template to deploy: %v", templateName, err)
	}

	generators := map[string]generator.Generator{
		"expression": generator.NewExpressionValueGenerator(rand.New(rand.NewSource(time.Now().UnixNano()))),
	}
	p := template.NewProcessor(generators)

	tmpl.Parameters = append(tmpl.Parameters, tapi.Parameter{
		Name:  "DOMAIN",
		Value: domain,
	}, tapi.Parameter{
		Name:  "APISERVER",
		Value: apiserver,
	})

	p.Process(&tmpl)

	println("Creating " + templateName + " resources")

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
		printResult(templateName, Failure, err)
	} else {
		printResult(templateName, Success, nil)
	}
}

func addLabelIfNotxisEt(metadata *api.ObjectMeta, name string, value string) bool {
	if metadata.Labels == nil {
		metadata.Labels = make(map[string]string)
	}
	labels := metadata.Labels
	current := labels[name]
	if len(current) == 0 {
		labels[name] = value
		return true
	}
	return false
}

func installTemplates(kc *k8sclient.Client, c *oclient.Client, fac *cmdutil.Factory, v string, templateUrl string, dockerRegistry string, domain string) error {
	ns, _, err := fac.DefaultNamespace()
	if err != nil {
		util.Fatal("No default namespace")
		return err
	}
	templates := c.Templates(ns)

	util.Infof("Downloading templates for version %v\n", v)
	uri := fmt.Sprintf(templateUrl, v)
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

	typeOfMaster := util.TypeOfMaster(kc)

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
		jsonData, err = appendDockerRegistryPrefix(jsonData, dockerRegistry)
		if err != nil {
			util.Fatalf("Cannot append docker registry: %v", err)
		}
		jsonData = replaceDomain(jsonData, domain, ns, typeOfMaster)

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
		if typeOfMaster == util.Kubernetes {
			appName := name
			name = "catalog-" + appName

			// lets install ConfigMaps for the templates
			// TODO should the name have a prefix?
			configmap := api.ConfigMap{
				ObjectMeta: api.ObjectMeta{
					Name:      name,
					Namespace: ns,
					Labels: map[string]string{
						"name":     appName,
						"provider": "fabric8.io",
						"kind":     "catalog",
					},
				},
				Data: map[string]string{
					name + ".json": string(jsonData),
				},
			}
			configmaps := kc.ConfigMaps(ns)
			_, err = configmaps.Get(name)
			if err == nil {
				err = configmaps.Delete(name)
				if err != nil {
					util.Errorf("Could not delete configmap %s due to: %v\n", name, err)
				}
			}
			_, err = configmaps.Create(&configmap)
			if err != nil {
				util.Fatalf("Failed to create configmap %v", err)
				return err
			}
		} else {
			// lets install the OpenShift templates
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
	}
	return nil
}

func replaceDomain(jsonData []byte, domain string, ns string, typeOfMaster util.MasterType) []byte {
	if len(domain) <= 0 {
		return jsonData
	}
	text := string(jsonData)
	if typeOfMaster == util.Kubernetes {
		text = strings.Replace(text, "gogs.vagrant.f8", "gogs-"+ns+"."+domain, -1)
	}
	text = strings.Replace(text, "vagrant.f8", domain, -1)
	return []byte(text)
}

func loadJsonDataAndAppendDockerRegistryPrefix(uri string, dockerRegistry string) ([]byte, error) {
	resp, err := http.Get(uri)
	if err != nil {
		util.Fatalf("Cannot get fabric8 template to deploy: %v", err)
	}
	defer resp.Body.Close()
	jsonData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		util.Fatalf("Cannot get fabric8 template to deploy: %v", err)
	}
	jsonData, err = appendDockerRegistryPrefix(jsonData, dockerRegistry)
	if err != nil {
		util.Fatalf("Cannot append docker registry: %v", err)
	}
	return jsonData, nil
}

func appendDockerRegistryPrefix(jsonData []byte, dockerRegistry string) ([]byte, error) {
	if len(dockerRegistry) <= 0 {
		return jsonData, nil
	}
	r, err := regexp.Compile("\"image\" : \"fabric8/")
	if err != nil {
		return nil, err
	}
	return r.ReplaceAll(jsonData, []byte("\"image\" : \""+dockerRegistry+"/fabric8/")), nil
}

func createRoutes(c *k8sclient.Client, oc *oclient.Client, fac *cmdutil.Factory) error {
	ns, _, err := fac.DefaultNamespace()
	if err != nil {
		util.Fatal("No default namespace")
		return err
	}
	domain := os.Getenv("KUBERNETES_DOMAIN")
	if domain == "" {
		domain = DefaultDomain
	}
	return createRoutesForDomain(ns, domain, c, oc, fac)
}

func deployFabric8SecurityContextConstraints(c *k8sclient.Client, f *cmdutil.Factory, ns string) (Result, error) {
	name := Fabric8SCC
	if ns != "default" {
		name += "-" + ns
	}
	scc := kapi.SecurityContextConstraints{
		ObjectMeta: kapi.ObjectMeta{
			Name: name,
		},
		Priority:                 &[]int{10}[0],
		AllowPrivilegedContainer: true,
		AllowHostNetwork:         true,
		AllowHostPorts:           true,
		Volumes:                  []kapi.FSType{kapi.FSTypeAll},
		SELinuxContext: kapi.SELinuxContextStrategyOptions{
			Type: kapi.SELinuxStrategyRunAsAny,
		},
		RunAsUser: kapi.RunAsUserStrategyOptions{
			Type: kapi.RunAsUserStrategyRunAsAny,
		},
		Users: []string{
			"system:serviceaccount:openshift-infra:build-controller",
			"system:serviceaccount:" + ns + ":default",
			"system:serviceaccount:" + ns + ":fabric8",
			"system:serviceaccount:" + ns + ":gerrit",
			"system:serviceaccount:" + ns + ":jenkins",
			"system:serviceaccount:" + ns + ":router",
			"system:serviceaccount:" + ns + ":registry",
			"system:serviceaccount:" + ns + ":gogs",
			"system:serviceaccount:" + ns + ":fluentd",
		},
		Groups: []string{bootstrappolicy.ClusterAdminGroup, bootstrappolicy.NodesGroup},
	}
	_, err := c.SecurityContextConstraints().Get(name)
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

func deployFabric8SASSecurityContextConstraints(c *k8sclient.Client, f *cmdutil.Factory, ns string) (Result, error) {
	name := Fabric8SASSCC
	scc := kapi.SecurityContextConstraints{
		ObjectMeta: kapi.ObjectMeta{
			Name: name,
		},
		SELinuxContext: kapi.SELinuxContextStrategyOptions{
			Type: kapi.SELinuxStrategyRunAsAny,
		},
		RunAsUser: kapi.RunAsUserStrategyOptions{
			Type: kapi.RunAsUserStrategyRunAsAny,
		},
		Groups:  []string{"system:serviceaccounts"},
		Volumes: []kapi.FSType{kapi.FSTypeGitRepo, kapi.FSTypeConfigMap, kapi.FSTypeSecret, kapi.FSTypeEmptyDir},
	}
	_, err := c.SecurityContextConstraints().Get(name)
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
	options := policy.RoleModificationOptions{
		RoleName:            roleName,
		RoleBindingAccessor: policy.NewClusterRoleBindingAccessor(c),
		Users:               []string{userName},
	}

	return options.AddRole()
}

func urlJoin(repo string, path string) string {
	return repo + path
}

func f8ConsoleVersion(mavenRepo string, v string, typeOfMaster util.MasterType) string {
	metadataUrl := urlJoin(mavenRepo, consoleMetadataUrl)
	if typeOfMaster == util.Kubernetes {
		metadataUrl = urlJoin(mavenRepo, consoleKubernetesMetadataUrl)
	}
	return versionForUrl(v, metadataUrl)
}

func versionForUrl(v string, metadataUrl string) string {
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
