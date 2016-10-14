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
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"reflect"
	"regexp"
	goruntime "runtime"
	"strings"
	"time"

	"github.com/fabric8io/gofabric8/client"
	"github.com/fabric8io/gofabric8/util"
	"github.com/ghodss/yaml"
	aapi "github.com/openshift/origin/pkg/authorization/api"
	aapiv1 "github.com/openshift/origin/pkg/authorization/api/v1"
	oclient "github.com/openshift/origin/pkg/client"
	"github.com/openshift/origin/pkg/cmd/admin/policy"
	"github.com/openshift/origin/pkg/cmd/server/bootstrappolicy"
	deployapi "github.com/openshift/origin/pkg/deploy/api"
	deployapiv1 "github.com/openshift/origin/pkg/deploy/api/v1"
	oauthapi "github.com/openshift/origin/pkg/oauth/api"
	oauthapiv1 "github.com/openshift/origin/pkg/oauth/api/v1"
	projectapi "github.com/openshift/origin/pkg/project/api"
	projectapiv1 "github.com/openshift/origin/pkg/project/api/v1"
	"github.com/openshift/origin/pkg/template"
	tapi "github.com/openshift/origin/pkg/template/api"
	tapiv1 "github.com/openshift/origin/pkg/template/api/v1"
	"github.com/openshift/origin/pkg/template/generator"
	"github.com/spf13/cobra"
	"k8s.io/kubernetes/pkg/api"
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/meta"
	"k8s.io/kubernetes/pkg/api/resource"
	"k8s.io/kubernetes/pkg/api/v1"
	"k8s.io/kubernetes/pkg/apis/extensions/v1beta1"
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

	platformMetadataUrl = "io/fabric8/platform/packages/fabric8-platform/maven-metadata.xml"
	ipaasMetadataUrl    = "io/fabric8/ipaas/platform/packages/ipaas-platform/maven-metadata.xml"

	platformPackageUrlPrefix = "io/fabric8/platform/packages/fabric8-platform/%[1]s/fabric8-platform-%[1]s-"
	consolePackageUrlPrefix  = "io/fabric8/platform/packages/console/%[1]s/console-%[1]s-"
	ipaasPackageUrlPrefix    = "io/fabric8/ipaas/platform/packages/ipaas-platform/%[1]s/ipaas-platform-%[1]s-"

	Fabric8SCC    = "fabric8"
	Fabric8SASSCC = "fabric8-sa-group"
	PrivilegedSCC = "privileged"
	RestrictedSCC = "restricted"

	runFlag             = "app"
	useIngressFlag      = "ingress"
	useLoadbalancerFlag = "loadbalancer"
	versionPlatformFlag = "version"
	versionConsoleFlag  = "version-console"
	versioniPaaSFlag    = "version-ipaas"
	versionDevOpsFlag   = "version-devops"
	versionKubeflixFlag = "version-kubeflix"
	versionZipkinFlag   = "version-zipkin"
	mavenRepoFlag       = "maven-repo"
	dockerRegistryFlag  = "docker-registry"
	archFlag            = "arch"
	pvFlag              = "pv"
	packageFlag         = "package"

	platformPackage = "platform"
	consolePackage  = "console"
	iPaaSPackage    = "ipaas"

	domainAnnotation   = "fabric8.io/domain"
	typeLabel          = "type"
	teamTypeLabelValue = "team"

	fabric8Environments = "fabric8-environments"
	exposecontrollerCM  = "exposecontroller"

	ingress      = "Ingress"
	loadBalancer = "LoadBalancer"
	nodePort     = "NodePort"
	route        = "Route"

	boot2docker                   = "boot2docker"
	exposeRule                    = "exposer"
	externalAPIServerAddressLabel = "fabric8.io/externalApiServerAddress"

	externalIPLabel = "fabric8.io/externalIP"

	gogsDefaultUsername = "gogsadmin"
	gogsDefaultPassword = "RedHat$1"

	minishiftDefaultUsername = "admin"
	minishiftDefaultPassword = "admin"

	latest           = "latest"
	mavenRepoDefault = "https://repo1.maven.org/maven2/"
	cdPipeline       = "cd-pipeline"
)

// Fabric8Deployment structure to work with the fabric8 deploy command
type DefaultFabric8Deployment struct {
	domain          string
	apiserver       string
	dockerRegistry  string
	arch            string
	mavenRepo       string
	appToRun        string
	packageName     string
	templates       bool
	pv              bool
	deployConsole   bool
	useIngress      bool
	useLoadbalancer bool
	versionPlatform string
	versionConsole  string
	versioniPaaS    string
	versionDevOps   string
	versionKubeflix string
	versionZipkin   string
	yes             bool
}

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
			d := DefaultFabric8Deployment{
				domain:          cmd.Flags().Lookup(domainFlag).Value.String(),
				apiserver:       cmd.Flags().Lookup(apiServerFlag).Value.String(),
				arch:            cmd.Flags().Lookup(archFlag).Value.String(),
				mavenRepo:       cmd.Flags().Lookup(mavenRepoFlag).Value.String(),
				appToRun:        cmd.Flags().Lookup(runFlag).Value.String(),
				packageName:     cmd.Flags().Lookup(packageFlag).Value.String(),
				deployConsole:   cmd.Flags().Lookup(consoleFlag).Value.String() == "true",
				dockerRegistry:  cmd.Flags().Lookup(dockerRegistryFlag).Value.String(),
				useIngress:      cmd.Flags().Lookup(useIngressFlag).Value.String() == "true",
				versionConsole:  cmd.Flags().Lookup(versionConsoleFlag).Value.String(),
				templates:       cmd.Flags().Lookup(templatesFlag).Value.String() == "true",
				versionPlatform: cmd.Flags().Lookup(versionPlatformFlag).Value.String(),
				versioniPaaS:    cmd.Flags().Lookup(versioniPaaSFlag).Value.String(),
				versionDevOps:   cmd.Flags().Lookup(versionDevOpsFlag).Value.String(),
				versionKubeflix: cmd.Flags().Lookup(versionKubeflixFlag).Value.String(),
				versionZipkin:   cmd.Flags().Lookup(versionZipkinFlag).Value.String(),
				useLoadbalancer: cmd.Flags().Lookup(useLoadbalancerFlag).Value.String() == "true",
				pv:              cmd.Flags().Lookup(pvFlag).Value.String() == "true",
				yes:             cmd.Flags().Lookup(yesFlag).Value.String() == "false",
			}
			deploy(f, d)
		},
	}
	cmd.PersistentFlags().StringP(domainFlag, "d", defaultDomain(), "The domain name to append to the service name to access web applications")
	cmd.PersistentFlags().String(apiServerFlag, "", "overrides the api server url")
	cmd.PersistentFlags().String(archFlag, goruntime.GOARCH, "CPU architecture for referencing Docker images with this as a name suffix")
	cmd.PersistentFlags().String(versionPlatformFlag, "latest", "The version to use for the Fabric8 Platform packages")
	cmd.PersistentFlags().String(versioniPaaSFlag, "latest", "The version to use for the Fabric8 iPaaS templates")
	cmd.PersistentFlags().String(versionDevOpsFlag, "latest", "The version to use for the Fabric8 DevOps templates")
	cmd.PersistentFlags().String(versionKubeflixFlag, "latest", "The version to use for the Kubeflix templates")
	cmd.PersistentFlags().String(versionZipkinFlag, "latest", "The version to use for the Zipkin templates")
	cmd.PersistentFlags().String(mavenRepoFlag, mavenRepoDefault, "The maven repo used to find releases of fabric8")
	cmd.PersistentFlags().String(dockerRegistryFlag, "", "The docker registry used to download fabric8 images. Typically used to point to a staging registry")
	cmd.PersistentFlags().String(runFlag, cdPipeline, "The name of the fabric8 app to startup. e.g. use `--app=cd-pipeline` to run the main CI/CD pipeline app")
	cmd.PersistentFlags().String(packageFlag, "", "The name of the package to startup. e.g. use `--package=platform` to run the fabric8 platform. Other values `console` or `ipaas`")
	cmd.PersistentFlags().Bool(pvFlag, false, "Default: false, unless on minikube or minishift where persistence is enabled out of the box")
	cmd.PersistentFlags().Bool(templatesFlag, true, "Should the standard Fabric8 templates be installed?")
	cmd.PersistentFlags().Bool(consoleFlag, true, "Should the Fabric8 console be deployed?")
	cmd.PersistentFlags().Bool(useIngressFlag, true, "Should Ingress NGINX controller be enabled by default when deploying to Kubernetes?")
	cmd.PersistentFlags().Bool(useLoadbalancerFlag, false, "Should Cloud Provider LoadBalancer be used to expose services when running to Kubernetes? (overrides ingress)")

	return cmd
}

// GetDefaultFabric8Deployment create new instance of Fabric8Deployment
func GetDefaultFabric8Deployment() DefaultFabric8Deployment {
	d := DefaultFabric8Deployment{}
	d.domain = defaultDomain()
	d.arch = goruntime.GOARCH
	d.versionConsole = latest
	d.versioniPaaS = latest
	d.versionDevOps = latest
	d.versionKubeflix = latest
	d.versionZipkin = latest
	d.mavenRepo = mavenRepoDefault
	d.appToRun = cdPipeline
	d.pv = false
	d.templates = true
	d.deployConsole = true
	d.useLoadbalancer = false
	return d
}

func initSchema() {
	aapi.AddToScheme(api.Scheme)
	aapiv1.AddToScheme(api.Scheme)
	tapi.AddToScheme(api.Scheme)
	tapiv1.AddToScheme(api.Scheme)
	projectapi.AddToScheme(api.Scheme)
	projectapiv1.AddToScheme(api.Scheme)
	deployapi.AddToScheme(api.Scheme)
	deployapiv1.AddToScheme(api.Scheme)
	oauthapi.AddToScheme(api.Scheme)
	oauthapiv1.AddToScheme(api.Scheme)
}

func deploy(f *cmdutil.Factory, d DefaultFabric8Deployment) {
	c, cfg := client.NewClient(f)
	ns, _, _ := f.DefaultNamespace()

	domain := d.domain
	dockerRegistry := d.dockerRegistry
	arch := d.arch

	mini, err := util.IsMini()

	// during initial deploy we cant rely on just the context - check the node names too
	if !mini {
		_, mini = isNodeNameMini(c, ns)
	}

	if err != nil {
		util.Failuref("Unable to detect platform deploying to %v", err)
	}
	typeOfMaster := util.TypeOfMaster(c)

	// extract the ip address from the URL
	u, err := url.Parse(cfg.Host)
	if err != nil {
		util.Fatalf("%s", err)
	}

	ip, _, err := net.SplitHostPort(u.Host)
	if err != nil && !strings.Contains(err.Error(), "missing port in address") {
		util.Fatalf("%s", err)
	}

	// default xip domain if local deployment incase users deploy ingress controller or router
	if mini && typeOfMaster == util.OpenShift {
		domain = ip + ".xip.io"
	}

	// default to the server from the current context
	apiserver := u.Host
	if d.apiserver != "" {
		apiserver = d.apiserver
	}

	util.Info("Deploying fabric8 to your ")
	util.Success(string(typeOfMaster))
	util.Info(" installation at ")
	util.Success(cfg.Host)
	util.Info(" for domain ")
	util.Success(domain)
	util.Info(" in namespace ")
	util.Successf("%s\n\n", ns)

	pv, err := shouldEnablePV(c, d.pv)
	if err != nil {
		util.Fatalf("No nodes available, something bad has happened: %v", err)
	}

	mavenRepo := d.mavenRepo
	if !strings.HasSuffix(mavenRepo, "/") {
		mavenRepo = mavenRepo + "/"
	}
	util.Info("Loading fabric8 releases from maven repository:")
	util.Successf("%s\n", mavenRepo)

	if len(dockerRegistry) > 0 {
		util.Infof("Loading fabric8 docker images from docker registry: %s\n", dockerRegistry)
	}

	if len(apiserver) == 0 {
		apiserver = domain
	}

	if strings.Contains(domain, "=") {
		util.Warnf("\nInvalid domain: %s\n\n", domain)
	} else if confirmAction(d.yes) {

		oc, _ := client.NewOpenShiftClient(cfg)

		initSchema()

		packageName := d.packageName
		if len(packageName) > 0 {
			versionPlatform := versionForUrl(d.versionPlatform, urlJoin(mavenRepo, platformMetadataUrl))
			baseUri := ""

			switch packageName {
			case "":
			case platformPackage:
				baseUri = platformPackageUrlPrefix
			case consolePackage:
				baseUri = consolePackageUrlPrefix
			case iPaaSPackage:
				baseUri = ipaasPackageUrlPrefix
				versionPlatform = versionForUrl(d.versioniPaaS, urlJoin(mavenRepo, ipaasMetadataUrl))
			default:
				util.Fatalf("Unknown package name %s passed via the flag %s", packageName, packageFlag)
			}
			uri := fmt.Sprintf(urlJoin(mavenRepo, baseUri), versionPlatform)

			if typeOfMaster == util.Kubernetes {
				uri += "kubernetes.yml"
			} else {
				uri += "openshift.yml"

				r, err := verifyRestrictedSecurityContextConstraints(c, f)
				printResult("SecurityContextConstraints restricted", r, err)
				r, err = deployFabric8SecurityContextConstraints(c, f, ns)
				printResult("SecurityContextConstraints fabric8", r, err)
				r, err = deployFabric8SASSecurityContextConstraints(c, f, ns)
				printResult("SecurityContextConstraints "+Fabric8SASSCC, r, err)

				/*
					printAddClusterRoleToUser(oc, f, "cluster-admin", "system:serviceaccount:"+ns+":fabric8")
					printAddClusterRoleToUser(oc, f, "cluster-admin", "system:serviceaccount:"+ns+":jenkins")
				*/
				printAddClusterRoleToUser(oc, f, "cluster-admin", "system:serviceaccount:"+ns+":exposecontroller")
				/*
					printAddClusterRoleToUser(oc, f, "cluster-reader", "system:serviceaccount:"+ns+":metrics")
					printAddClusterRoleToUser(oc, f, "cluster-reader", "system:serviceaccount:"+ns+":fluentd")
				*/

				printAddClusterRoleToGroup(oc, f, "cluster-reader", "system:serviceaccounts")

				printAddServiceAccount(c, f, "fluentd")
				printAddServiceAccount(c, f, "registry")
				printAddServiceAccount(c, f, "router")
			}

			// now lets apply this template
			util.Infof("Now about to install package %s\n", uri)

			resp, err := http.Get(uri)
			if err != nil {
				util.Fatalf("Cannot load YAML package at %s got: %v", uri, err)
			}
			defer resp.Body.Close()
			yamlData, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				util.Fatalf("Cannot load YAML from %s got: %v", uri, err)
			}

			format := "yaml"
			createTemplate(yamlData, format, packageName, ns, domain, apiserver, c, oc, pv)

			createMissingPVs(c, ns)
		} else {
			consoleVersion := f8ConsoleVersion(mavenRepo, d.versionConsole, typeOfMaster)
			versionDevOps := versionForUrl(d.versionDevOps, urlJoin(mavenRepo, devOpsMetadataUrl))
			versionKubeflix := versionForUrl(d.versionKubeflix, urlJoin(mavenRepo, kubeflixMetadataUrl))
			versionZipkin := versionForUrl(d.versionZipkin, urlJoin(mavenRepo, zipkinMetadataUrl))
			versioniPaaS := versionForUrl(d.versioniPaaS, urlJoin(mavenRepo, iPaaSMetadataUrl))

			util.Warnf("\nStarting fabric8 console deployment using %s...\n\n", consoleVersion)

			if typeOfMaster == util.Kubernetes {
				uri := fmt.Sprintf(urlJoin(mavenRepo, baseConsoleKubernetesUrl), consoleVersion)
				if fabric8ImageAdaptionNeeded(dockerRegistry, arch) {
					jsonData, err := loadJsonDataAndAdaptFabric8Images(uri, dockerRegistry, arch)
					if err == nil {
						tmpFileName := path.Join(os.TempDir(), "fabric8-console.json")
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

				if d.deployConsole {
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
				printAddClusterRoleToUser(oc, f, "cluster-admin", "system:serviceaccount:"+ns+":exposecontroller")
				printAddClusterRoleToUser(oc, f, "cluster-reader", "system:serviceaccount:"+ns+":metrics")
				printAddClusterRoleToUser(oc, f, "cluster-reader", "system:serviceaccount:"+ns+":fluentd")

				printAddClusterRoleToGroup(oc, f, "cluster-reader", "system:serviceaccounts")

				printAddServiceAccount(c, f, "fluentd")
				printAddServiceAccount(c, f, "registry")
				printAddServiceAccount(c, f, "router")

				if d.templates {
					if d.deployConsole {
						uri := fmt.Sprintf(urlJoin(mavenRepo, baseConsoleUrl), consoleVersion)
						format := "json"
						jsonData, err := loadJsonDataAndAdaptFabric8Images(uri, dockerRegistry, arch)
						if err != nil {
							printError("failed to apply docker registry prefix", err)
						}

						// lets delete the OAuthClient first as the domain may have changed
						oc.OAuthClients().Delete("fabric8")
						createTemplate(jsonData, format, "fabric8 console", ns, domain, apiserver, c, oc, pv)

						oac, err := oc.OAuthClients().Get("fabric8")
						if err != nil {
							printError("failed to get the OAuthClient called fabric8", err)
						}

						// lets add the nodePort URL to the OAuthClient
						service, err := c.Services(ns).Get("fabric8")
						if err != nil {
							printError("failed to get the Service called fabric8", err)
						}
						port := 0
						for _, p := range service.Spec.Ports {
							port = int(p.NodePort)
						}
						if port == 0 {
							util.Errorf("Failed to find nodePort on the Service called fabric8\n")
						}
						redirectURL := fmt.Sprintf("http://%s:%d", ip, port)
						println("Adding OAuthClient redirectURL: " + redirectURL)
						oac.RedirectURIs = append(oac.RedirectURIs, redirectURL)
						oac.ResourceVersion = ""
						oc.OAuthClients().Delete("fabric8")
						_, err = oc.OAuthClients().Create(oac)
						if err != nil {
							printError("failed to create the OAuthClient called fabric8", err)
						}

					}
				} else {
					printError("Ignoring the deploy of the fabric8 console", nil)
				}
			}
			if d.deployConsole {
				println("Created fabric8 console")
			}

			if d.templates {
				println("Installing templates!")
				printError("Install DevOps templates", installTemplates(c, oc, f, versionDevOps, urlJoin(mavenRepo, devopsTemplatesDistroUrl), dockerRegistry, arch, domain))
				printError("Install iPaaS templates", installTemplates(c, oc, f, versioniPaaS, urlJoin(mavenRepo, iPaaSTemplatesDistroUrl), dockerRegistry, arch, domain))
				printError("Install Kubeflix templates", installTemplates(c, oc, f, versionKubeflix, urlJoin(mavenRepo, kubeflixTemplatesDistroUrl), dockerRegistry, arch, domain))
				printError("Install Zipkin templates", installTemplates(c, oc, f, versionZipkin, urlJoin(mavenRepo, zipkinTemplatesDistroUrl), dockerRegistry, arch, domain))
			} else {
				printError("Ignoring the deploy of templates", nil)
			}

			runTemplate(c, oc, "exposecontroller", ns, domain, apiserver, pv)
			externalNodeName := ""
			if typeOfMaster == util.Kubernetes {
				if !mini && d.useIngress {
					ensureNamespaceExists(c, oc, "fabric8-system")
					runTemplate(c, oc, "ingress-nginx", ns, domain, apiserver, pv)
					externalNodeName = addIngressInfraLabel(c, ns)
				}
			}

			// create a populate the exposecontroller config map
			cfgms := c.ConfigMaps(ns)

			_, err := cfgms.Get(exposecontrollerCM)
			if err == nil {
				util.Infof("\nRecreating configmap %s \n", exposecontrollerCM)
				err = cfgms.Delete(exposecontrollerCM)
				if err != nil {
					printError("\nError deleting ConfigMap: "+exposecontrollerCM, err)
				}
			}

			domainData := "domain: " + domain + "\n"
			exposeData := exposeRule + ": " + defaultExposeRule(c, mini, d.useLoadbalancer) + "\n"
			configFile := map[string]string{
				"config.yml": domainData + exposeData,
			}
			configMap := kapi.ConfigMap{
				ObjectMeta: kapi.ObjectMeta{
					Name: exposecontrollerCM,
					Labels: map[string]string{
						"provider": "fabric8",
					},
				},
				Data: configFile,
			}
			_, err = cfgms.Create(&configMap)
			if err != nil {
				printError("Failed to create ConfigMap: "+exposecontrollerCM, err)
			}

			if len(d.appToRun) > 0 {
				runTemplate(c, oc, d.appToRun, ns, domain, apiserver, pv)
				createMissingPVs(c, ns)
			}

			// lets label the namespace/project as a developer team
			nss := c.Namespaces()
			namespace, err := nss.Get(ns)
			if err != nil {
				printError("Failed to load namespace", err)
			} else {
				changed := addLabelIfNotExist(&namespace.ObjectMeta, typeLabel, teamTypeLabelValue)
				if len(domain) > 0 {
					if addAnnotationIfNotExist(&namespace.ObjectMeta, domainAnnotation, domain) {
						changed = true
					}
				}
				if changed {
					_, err = nss.Update(namespace)
					if err != nil {
						printError("Failed to label and annotate namespace", err)
					}
				}
			}

			// lets ensure that there is a `fabric8-environments` ConfigMap so that the current namespace
			// shows up as a Team page in the console
			_, err = cfgms.Get(fabric8Environments)
			if err != nil {
				configMap := kapi.ConfigMap{
					ObjectMeta: kapi.ObjectMeta{
						Name: fabric8Environments,
						Labels: map[string]string{
							"provider": "fabric8",
							"kind":     "environments",
						},
					},
				}
				_, err = cfgms.Create(&configMap)
				if err != nil {
					printError("Failed to create ConfigMap: "+fabric8Environments, err)
				}
			}

			nodeClient := c.Nodes()
			nodes, err := nodeClient.List(api.ListOptions{})
			changed := false

			for _, node := range nodes.Items {
				// if running on a single node then we can use node ports to access kubernetes services
				if len(nodes.Items) == 1 {
					changed = addAnnotationIfNotExist(&node.ObjectMeta, externalIPLabel, ip)
				}
				changed = addAnnotationIfNotExist(&node.ObjectMeta, externalAPIServerAddressLabel, cfg.Host)
				if changed {
					_, err = nodeClient.Update(&node)
					if err != nil {
						printError("Failed to annotate node with ", err)
					}
				}
			}
			printSummary(typeOfMaster, externalNodeName, ns, domain)
		}
		openService(ns, "fabric8", c, false)
	}
}

func createMissingPVs(c *k8sclient.Client, ns string) {
	found, pvcs, pendingClaimNames := findPendingPVs(c, ns)
	if found {
		sshCommand := ""
		createPV(c, ns, pendingClaimNames, sshCommand)
		items := pvcs.Items
		for _, item := range items {
			status := item.Status.Phase
			if status == api.ClaimPending || status == api.ClaimLost {
				err := c.PersistentVolumeClaims(ns).Delete(item.ObjectMeta.Name)
				if err != nil {
					util.Infof("Error deleting PVC %s\n", item.ObjectMeta.Name)
				} else {
					util.Infof("Recreating PVC %s\n", item.ObjectMeta.Name)

					c.PersistentVolumeClaims(ns).Create(&api.PersistentVolumeClaim{
						ObjectMeta: api.ObjectMeta{
							Name:      item.ObjectMeta.Name,
							Namespace: ns,
						},
						Spec: api.PersistentVolumeClaimSpec{
							VolumeName:  ns + "-" + item.ObjectMeta.Name,
							AccessModes: []api.PersistentVolumeAccessMode{api.ReadWriteOnce},
							Resources: api.ResourceRequirements{
								Requests: api.ResourceList{
									api.ResourceName(api.ResourceStorage): resource.MustParse("1Gi"),
								},
							},
						},
					})
				}
			}
		}
	}
}

func printSummary(typeOfMaster util.MasterType, externalNodeName string, ns string, domain string) {
	util.Info("\n")
	util.Info("-------------------------\n")
	util.Info("\n")
	clientType := getClientTypeName(typeOfMaster)

	if externalNodeName != "" {
		util.Info("Deploying ingress controller on node ")
		util.Successf("%s", externalNodeName)
		util.Info(" use its external ip when configuring your wildcard DNS.\n")
		util.Infof("To change node move the label: `%s label node %s %s- && %s label node $YOUR_NEW_NODE %s=true`\n", clientType, externalNodeName, externalIPLabel, clientType, externalIPLabel)
		util.Info("\n")
	}

	util.Info("Default GOGS admin username/password = ")
	util.Successf("%s/%s\n", gogsDefaultUsername, gogsDefaultPassword)
	util.Info("\n")

	util.Infof("Downloading images and waiting to open the fabric8 console...\n")
	util.Info("\n")
	util.Info("-------------------------\n")
}

func isNodeNameMini(c *k8sclient.Client, ns string) (string, bool) {
	nodes, err := c.Nodes().List(api.ListOptions{})
	if err != nil {
		util.Errorf("\nUnable to find any nodes: %s\n", err)
	}
	if len(nodes.Items) == 1 {
		node := nodes.Items[0]
		return node.Name, (node.Name == "minishift" || node.Name == "minikube")
	}
	return "", false

}

func shouldEnablePV(c *k8sclient.Client, pv bool) (bool, error) {

	// did we choose to enable PV?
	if pv {
		return true, nil
	}

	// lets default use PV for mini*
	mini, _ := util.IsMini()
	if mini {
		return true, nil
	}

	// until PV works with stackpoint cloud and openshift lets not enable
	return false, nil
}

func getClientTypeName(typeOfMaster util.MasterType) string {
	if typeOfMaster == util.OpenShift {
		return "oc"
	}
	return "kubectl"
}

func addIngressInfraLabel(c *k8sclient.Client, ns string) string {
	nodeClient := c.Nodes()
	nodes, err := nodeClient.List(api.ListOptions{})
	if err != nil {
		util.Errorf("\nUnable to find any nodes: %s\n", err)
	}
	changed := false
	hasExistingExposeIPLabel, externalNodeName := hasExistingLabel(nodes, externalIPLabel)
	if externalNodeName != "" {
		return externalNodeName
	}
	if !hasExistingExposeIPLabel && len(nodes.Items) > 0 {
		for _, node := range nodes.Items {
			if !node.Spec.Unschedulable {
				changed = addLabelIfNotExist(&node.ObjectMeta, externalIPLabel, "true")
				if changed {
					_, err = nodeClient.Update(&node)
					if err != nil {
						printError("Failed to label node with ", err)
					}
					return node.Name
				}
			}
		}
	}
	if !changed && !hasExistingExposeIPLabel {
		util.Warnf("Unable to add label for ingress controller to run on a specific node, please add manually: kubectl label node [your node name] %s=true", externalIPLabel)
	}
	return ""
}

func hasExistingLabel(nodes *api.NodeList, label string) (bool, string) {
	if len(nodes.Items) > 0 {
		for _, node := range nodes.Items {
			if _, found := node.Labels[label]; found {
				return true, node.Name
			}
		}
	}
	return false, ""
}

func runTemplate(c *k8sclient.Client, oc *oclient.Client, appToRun string, ns string, domain string, apiserver string, pv bool) {
	util.Info("\n\nInstalling: ")
	util.Successf("%s\n\n", appToRun)
	typeOfMaster := util.TypeOfMaster(c)
	if typeOfMaster == util.Kubernetes {
		jsonData, format, err := loadTemplateData(ns, appToRun, c, oc)
		if err != nil {
			printError("Failed to load app "+appToRun, err)
		}
		createTemplate(jsonData, format, appToRun, ns, domain, apiserver, c, oc, pv)
	} else {
		tmpl, err := oc.Templates(ns).Get(appToRun)
		if err != nil {
			printError("Failed to load template "+appToRun, err)
		}
		util.Infof("Loaded template with %d objects", len(tmpl.Objects))
		processTemplate(tmpl, ns, domain, apiserver)

		objectCount := len(tmpl.Objects)

		util.Infof("Creating "+appToRun+" template resources from %d objects\n", objectCount)
		for _, o := range tmpl.Objects {
			err = processItem(c, oc, &o, ns, pv)
		}
	}
}

func loadTemplateData(ns string, templateName string, c *k8sclient.Client, oc *oclient.Client) ([]byte, string, error) {
	typeOfMaster := util.TypeOfMaster(c)
	if typeOfMaster == util.Kubernetes {
		catalogName := "catalog-" + templateName
		configMap, err := c.ConfigMaps(ns).Get(catalogName)
		if err != nil {
			return nil, "", err
		}
		for k, v := range configMap.Data {
			if strings.LastIndex(k, ".json") >= 0 {
				return []byte(v), "json", nil
			}
			if strings.LastIndex(k, ".yml") >= 0 || strings.LastIndex(k, ".yaml") >= 0 {
				return []byte(v), "yaml", nil
			}
		}
		return nil, "", fmt.Errorf("Could not find a key for the catalog %s which ends with `.json` or `.yml`", catalogName)

	} else {
		template, err := oc.Templates(ns).Get(templateName)
		if err != nil {
			return nil, "", err
		}
		data, err := json.Marshal(template)
		return data, "json", err
	}
	return nil, "", nil
}

func createTemplate(jsonData []byte, format string, templateName string, ns string, domain string, apiserver string, c *k8sclient.Client, oc *oclient.Client, pv bool) {
	var v1tmpl tapiv1.Template
	var err error
	if format == "yaml" {
		err = yaml.Unmarshal(jsonData, &v1tmpl)
	} else {
		err = json.Unmarshal(jsonData, &v1tmpl)
	}
	if err != nil {
		util.Fatalf("Cannot get %s template to deploy. error: %v\ntemplate: %s", templateName, err, string(jsonData))
	}
	var tmpl tapi.Template

	err = api.Scheme.Convert(&v1tmpl, &tmpl, nil)
	if err != nil {
		util.Fatalf("Cannot convert %s template to deploy: %v", templateName, err)
	}

	processTemplate(&tmpl, ns, domain, apiserver)

	objectCount := len(tmpl.Objects)

	if objectCount == 0 {
		// can't be a template so lets try just process it directly
		var v1List v1.List
		if format == "yaml" {
			err = yaml.Unmarshal(jsonData, &v1List)
		} else {
			err = json.Unmarshal(jsonData, &v1List)
		}
		if err != nil {
			util.Fatalf("Cannot unmarshal List %s. error: %v\ntemplate: %s", templateName, err, string(jsonData))
		}
		if len(v1List.Items) == 0 {
			processData(jsonData, format, templateName, ns, c, oc, pv)
		} else {
			for _, i := range v1List.Items {
				data := i.Raw
				if data == nil {
					util.Infof("no data!\n")
					continue
				}
				kind := ""
				o := i.Object
				if o != nil {
					objectKind := o.GetObjectKind()
					if objectKind != nil {
						groupVersionKind := objectKind.GroupVersionKind()
						kind = groupVersionKind.Kind
					}
				}
				if len(kind) == 0 {
					processData(data, format, templateName, ns, c, oc, pv)
				} else {
					// TODO how to find the Namespace?
					err = processResource(c, data, ns, kind)
					if err != nil {
						util.Fatalf("Failed to process kind %s template: %s error: %v\n", kind, err, templateName)
					}
				}
				if err != nil {
					util.Info("No kind found so processing data directly\n")
					printResult(templateName, Failure, err)
				}
			}
		}
	} else {
		util.Infof("Creating "+templateName+" template resources in namespace %s from %d objects\n", ns, objectCount)
		for _, o := range tmpl.Objects {
			err = processItem(c, oc, &o, ns, pv)
		}
	}
	if err != nil {
		printResult(templateName, Failure, err)
	} else {
		printResult(templateName, Success, nil)
	}
}

func processTemplate(tmpl *tapi.Template, ns string, domain string, apiserver string) {
	generators := map[string]generator.Generator{
		"expression": generator.NewExpressionValueGenerator(rand.New(rand.NewSource(time.Now().UnixNano()))),
	}
	p := template.NewProcessor(generators)

	ip, port, err := net.SplitHostPort(apiserver)
	if err != nil && !strings.Contains(err.Error(), "missing port in address") {
		util.Fatalf("%s", err)
	}

	tmpl.Parameters = append(tmpl.Parameters, tapi.Parameter{
		Name:  "DOMAIN",
		Value: ns + "." + domain,
	}, tapi.Parameter{
		Name:  "APISERVER",
		Value: ip,
	}, tapi.Parameter{
		Name:  "OAUTH_AUTHORIZE_PORT",
		Value: port,
	})

	errorList := p.Process(tmpl)
	for _, errInfo := range errorList {
		util.Errorf("Processing template field %s got error %s\n", errInfo.Field, errInfo.Detail)
	}
}

func processData(jsonData []byte, format string, templateName string, ns string, c *k8sclient.Client, oc *oclient.Client, pv bool) {
	// lets check if its an RC / ReplicaSet or something
	o, groupVersionKind, err := api.Codecs.UniversalDeserializer().Decode(jsonData, nil, nil)
	if err != nil {
		printResult(templateName, Failure, err)
	} else {
		kind := groupVersionKind.Kind
		//util.Infof("Processing resource of kind: %s version: %s\n", kind, groupVersionKind.Version)
		if len(kind) <= 0 {
			printResult(templateName, Failure, fmt.Errorf("Could not find kind from json %s", string(jsonData)))
		} else {
			accessor := meta.NewAccessor()
			ons, err := accessor.Namespace(o)
			if err == nil && len(ons) > 0 {
				util.Infof("Found namespace on kind %s of %s", kind, ons)
				ns = ons

				err := ensureNamespaceExists(c, oc, ns)
				if err != nil {
					printErr(err)
				}
			}
			if !pv {
				if kind == "PersistentVolumeClaim" {
					return
				}
				jsonData = removePVCVolumes(jsonData, format, templateName, kind)
			}
			err = processResource(c, jsonData, ns, kind)
			if err != nil {
				printResult(templateName, Failure, err)
			}
		}
	}
}

func removePVCVolumes(jsonData []byte, format string, templateName string, kind string) []byte {
	var err error
	if kind == "Deployment" {
		var deployment v1beta1.Deployment
		if format == "yaml" {
			err = yaml.Unmarshal(jsonData, &deployment)
		} else {
			err = json.Unmarshal(jsonData, &deployment)
		}
		if err != nil {
			util.Fatalf("Cannot unmarshal Deployment %s. error: %v\ntemplate: %s", templateName, err, string(jsonData))
		} else {
			updated := false
			podSpec := &deployment.Spec.Template.Spec
			for i, _ := range podSpec.Volumes {
				v := &podSpec.Volumes[i]
				pvc := v.PersistentVolumeClaim
				if pvc != nil {
					updated = true
					// lets convert the PVC to an EmptyDir
					v.PersistentVolumeClaim = nil
					v.EmptyDir = &v1.EmptyDirVolumeSource{
						Medium: v1.StorageMediumDefault,
					}
				}
			}
			if updated {
				util.Info("Converted Deployment to avoid the use of PersistentVolumeClaim\n")
				format = "json"
				jsonData, err = json.Marshal(&deployment)
				if err != nil {
					util.Fatalf("Failed to marshal modified Deployment %s. error: %v\ntemplate: %s", templateName, err, string(jsonData))
				}
				//util.Infof("Updated: %s\n", string(jsonData))
			}
		}
	}
	if kind == "DeploymentConfig" {
		var deployment deployapiv1.DeploymentConfig
		if format == "yaml" {
			err = yaml.Unmarshal(jsonData, &deployment)
		} else {
			err = json.Unmarshal(jsonData, &deployment)
		}
		if err != nil {
			util.Fatalf("Cannot unmarshal DeploymentConfig %s. error: %v\ntemplate: %s", templateName, err, string(jsonData))
		} else {
			updated := false
			podSpec := &deployment.Spec.Template.Spec
			for i, _ := range podSpec.Volumes {
				v := &podSpec.Volumes[i]
				pvc := v.PersistentVolumeClaim
				if pvc != nil {
					updated = true
					// lets convert the PVC to an EmptyDir
					v.PersistentVolumeClaim = nil
					v.EmptyDir = &v1.EmptyDirVolumeSource{
						Medium: v1.StorageMediumDefault,
					}
				}
			}
			if updated {
				util.Info("Converted DeploymentConfig to avoid the use of PersistentVolumeClaim\n")
				format = "json"
				jsonData, err = json.Marshal(&deployment)
				if err != nil {
					util.Fatalf("Failed to marshal modified DeploymentConfig %s. error: %v\ntemplate: %s", templateName, err, string(jsonData))
				}
				//util.Infof("Updated: %s\n", string(jsonData))
			}
		}
	}
	return jsonData
}

func processItem(c *k8sclient.Client, oc *oclient.Client, item *runtime.Object, ns string, pv bool) error {
	/*
		groupVersionKind, err := api.Scheme.ObjectKind(*item)
		if err != nil {
			return err
		}
		kind := groupVersionKind.Kind
		//kind := *item.GetObjectKind()
		util.Infof("Procesing kind %s\n", kind)
		b, err := json.Marshal(item)
		if err != nil {
			return err
		}
		return processResource(c, b, ns, kind)
	*/
	o := *item
	switch o := o.(type) {
	case *runtime.Unstructured:
		var (
			name, kind string
		)
		data := o.Object
		metadata := data["metadata"]
		switch metadata := metadata.(type) {
		case map[string]interface{}:
			namespace := metadata["namespace"]
			switch namespace := namespace.(type) {
			case string:
				//util.Infof("Custom namespace '%s'\n", namespace)
				if len(namespace) <= 0 {
					// TODO why is the namespace empty?
					// lets default the namespace to the default gogs namespace
					namespace = "user-secrets-source-admin"
				}
				ns = namespace

				// lets check that this new namespace exists
				err := ensureNamespaceExists(c, oc, ns)
				if err != nil {
					printErr(err)
				}
			}
			n := metadata["name"]
			switch n := n.(type) {
			case string:
				name = n
			}
			k := data["kind"]
			switch k := k.(type) {
			case string:
				kind = k
			}
		}
		//util.Infof("processItem %s with value: %#v\n", ns, o.Object)
		b, err := json.Marshal(o.Object)
		if err != nil {
			return err
		}
		if !pv {
			if kind == "PersistentVolumeClaim" {
				return nil
			}
			b = removePVCVolumes(b, "json", name, kind)
		}
		return processResource(c, b, ns, kind)
	default:
		util.Infof("Unknown type %v\n", reflect.TypeOf(item))
	}
	return nil
}

func ensureNamespaceExists(c *k8sclient.Client, oc *oclient.Client, ns string) error {
	typeOfMaster := util.TypeOfMaster(c)
	if typeOfMaster == util.Kubernetes {
		nss := c.Namespaces()
		_, err := nss.Get(ns)
		if err != nil {
			// lets assume it doesn't exist!
			util.Infof("Creating new Namespace: %s\n", ns)
			entity := kapi.Namespace{
				ObjectMeta: kapi.ObjectMeta{
					Name: ns,
					Labels: map[string]string{
						"provider": "fabric8",
					},
				},
			}
			_, err := nss.Create(&entity)
			return err
		}
	} else {
		_, err := oc.Projects().Get(ns)
		if err != nil {
			// lets assume it doesn't exist!
			request := projectapi.ProjectRequest{
				ObjectMeta: kapi.ObjectMeta{
					Name: ns,
					Labels: map[string]string{
						"provider": "fabric8",
					},
				},
			}
			util.Infof("Creating new Project: %s\n", ns)
			_, err := oc.ProjectRequests().Create(&request)
			return err
		}
	}
	return nil
}

func processResource(c *k8sclient.Client, b []byte, ns string, kind string) error {
	util.Infof("Processing resource kind: %s in namespace %s\n", kind, ns)
	req := c.Post().Body(b)
	if kind == "Deployment" {
		req.AbsPath("apis", "extensions/v1beta1", "namespaces", ns, strings.ToLower(kind+"s"))
	} else if kind == "BuildConfig" || kind == "DeploymentConfig" || kind == "Template" {
		req.AbsPath("oapi", "v1", "namespaces", ns, strings.ToLower(kind+"s"))
	} else if kind == "OAuthClient" || kind == "Project" || kind == "ProjectRequest" || kind == "RoleBinding" {
		req.AbsPath("oapi", "v1", strings.ToLower(kind+"s"))
	} else if kind == "Namespace" {
		req.AbsPath("api", "v1", "namespaces")
	} else {
		req.Namespace(ns).Resource(strings.ToLower(kind + "s"))
	}
	res := req.Do()
	if res.Error() != nil {
		err := res.Error()
		if err != nil {
			util.Warnf("Failed to create %s: %v", kind, err)
			return err
		}
	}
	var statusCode int
	res.StatusCode(&statusCode)
	if statusCode != http.StatusCreated {
		return fmt.Errorf("Failed to create %s: %d", kind, statusCode)
	}
	return nil
}

func addLabelIfNotExist(metadata *api.ObjectMeta, name string, value string) bool {
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

func addAnnotationIfNotExist(metadata *api.ObjectMeta, name string, value string) bool {
	if metadata.Annotations == nil {
		metadata.Annotations = make(map[string]string)
	}
	annotations := metadata.Annotations
	current := annotations[name]
	if len(current) == 0 {
		annotations[name] = value
		return true
	}
	return false
}

func installTemplates(kc *k8sclient.Client, c *oclient.Client, fac *cmdutil.Factory, v string, templateUrl string, dockerRegistry string, arch string, domain string) error {
	ns, _, err := fac.DefaultNamespace()
	if err != nil {
		util.Fatal("No default namespace")
		return err
	}
	templates := c.Templates(ns)

	uri := fmt.Sprintf(templateUrl, v)
	util.Infof("Downloading apps from: %v\n", uri)
	resp, err := http.Get(uri)
	if err != nil {
		util.Fatalf("Cannot get fabric8 template to deploy: %v", err)
	}
	defer resp.Body.Close()

	tmpFileName := path.Join(os.TempDir(), "fabric8-template-distros.tar.gz")
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

		jsonData, err := ioutil.ReadAll(rc)
		if err != nil {
			util.Fatalf("Cannot get fabric8 template to deploy: %v", err)
		}
		jsonData, err = adaptFabric8ImagesInResourceDescriptor(jsonData, dockerRegistry, arch)
		if err != nil {
			util.Fatalf("Cannot append docker registry: %v", err)
		}
		jsonData = replaceDomain(jsonData, domain, ns, typeOfMaster)

		var v1tmpl tapiv1.Template
		lowerName := strings.ToLower(f.Name)

		// if the folder starts with kubernetes/ or openshift/ then lets filter based on the cluster:
		if strings.HasPrefix(lowerName, "kubernetes/") && typeOfMaster != util.Kubernetes {
			//util.Info("Ignoring as on openshift!")
			continue
		}
		if strings.HasPrefix(lowerName, "openshift/") && typeOfMaster == util.Kubernetes {
			//util.Info("Ignoring as on kubernetes!")
			continue
		}
		configMapKeySuffix := ".json"
		if strings.HasSuffix(lowerName, ".yml") || strings.HasSuffix(lowerName, ".yaml") {
			configMapKeySuffix = ".yml"
			err = yaml.Unmarshal(jsonData, &v1tmpl)

		} else if strings.HasSuffix(lowerName, ".json") {
			err = json.Unmarshal(jsonData, &v1tmpl)
		} else {
			continue
		}
		if err != nil {
			util.Fatalf("Cannot unmarshall the fabric8 template %s to deploy: %v", f.Name, err)
		}
		util.Infof("Loading template %s\n", f.Name)

		var tmpl tapi.Template

		err = api.Scheme.Convert(&v1tmpl, &tmpl, nil)
		if err != nil {
			util.Fatalf("Cannot get fabric8 template to deploy: %v", err)
			return err
		}

		name := tmpl.ObjectMeta.Name
		template := true
		if len(name) <= 0 {
			template = false
			name = f.Name
			idx := strings.LastIndex(name, "/")
			if idx > 0 {
				name = name[idx+1:]
			}
			idx = strings.Index(name, ".")
			if idx > 0 {
				name = name[0:idx]
			}

		}
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
					name + configMapKeySuffix: string(jsonData),
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
			if !template {
				templateName := name
				var v1List v1.List
				if configMapKeySuffix == ".json" {
					err = json.Unmarshal(jsonData, &v1List)
				} else {
					err = yaml.Unmarshal(jsonData, &v1List)
				}
				if err != nil {
					util.Fatalf("Cannot unmarshal List %s to deploy. error: %v\ntemplate: %s", templateName, err, string(jsonData))
				}
				if len(v1List.Items) == 0 {
					// lets check if its an RC / ReplicaSet or something
					_, groupVersionKind, err := api.Codecs.UniversalDeserializer().Decode(jsonData, nil, nil)
					if err != nil {
						printResult(templateName, Failure, err)
					} else {
						kind := groupVersionKind.Kind
						util.Infof("Processing resource of kind: %s version: %s\n", kind, groupVersionKind.Version)
						if len(kind) <= 0 {
							printResult(templateName, Failure, fmt.Errorf("Could not find kind from json %s", string(jsonData)))
						} else {
							util.Warnf("Cannot yet process kind %s, kind for %s\n", kind, templateName)
							continue
						}
					}
				} else {
					var kubeList api.List
					err = api.Scheme.Convert(&v1List, &kubeList, nil)
					if err != nil {
						util.Fatalf("Cannot convert %s List to deploy: %v", templateName, err)
					}
					tmpl = tapi.Template{
						ObjectMeta: api.ObjectMeta{
							Name:      name,
							Namespace: ns,
							Labels: map[string]string{
								"name":     name,
								"provider": "fabric8.io",
							},
						},
						Objects: kubeList.Items,
					}
				}
			}

			// remove newlines from description to avoid breaking `oc get template`
			description := tmpl.ObjectMeta.Annotations["description"]
			if len(description) > 0 {
				tmpl.ObjectMeta.Annotations["description"] = strings.Replace(description, "\n", " ", -1)
			}

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
				util.Warnf("Failed to create template %v", err)
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

func loadJsonDataAndAdaptFabric8Images(uri string, dockerRegistry string, arch string) ([]byte, error) {
	resp, err := http.Get(uri)
	if err != nil {
		util.Fatalf("Cannot get fabric8 template to deploy: %v", err)
	}
	defer resp.Body.Close()
	jsonData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		util.Fatalf("Cannot get fabric8 template to deploy: %v", err)
	}
	jsonData, err = adaptFabric8ImagesInResourceDescriptor(jsonData, dockerRegistry, arch)
	if err != nil {
		util.Fatalf("Cannot append docker registry: %v", err)
	}
	return jsonData, nil
}

// Check whether mangling of source descriptors is needed
func fabric8ImageAdaptionNeeded(dockerRegistry string, arch string) bool {
	return len(dockerRegistry) > 0 || arch == "arm"
}

// Prepend a docker registry and add a conditional suffix when running under arm
func adaptFabric8ImagesInResourceDescriptor(jsonData []byte, dockerRegistry string, arch string) ([]byte, error) {
	if !fabric8ImageAdaptionNeeded(dockerRegistry, arch) {
		return jsonData, nil
	}

	var suffix string
	if arch == "arm" {
		suffix = "-arm"
	} else {
		suffix = ""
	}

	var registryReplacePart string
	if len(dockerRegistry) <= 0 {
		registryReplacePart = ""
	} else {
		registryReplacePart = dockerRegistry + "/"
	}

	r, err := regexp.Compile("(\"image\"\\s*:\\s*\")(fabric8/[^:\"]+)(:[^:\"]+)?\"")
	if err != nil {
		return nil, err
	}
	return r.ReplaceAll(jsonData, []byte("${1}"+registryReplacePart+"${2}"+suffix+"${3}\"")), nil
}
func deployFabric8SecurityContextConstraints(c *k8sclient.Client, f *cmdutil.Factory, ns string) (Result, error) {
	name := Fabric8SCC
	if ns != "default" {
		name += "-" + ns
	}
	var priority int32 = 10
	scc := kapi.SecurityContextConstraints{
		ObjectMeta: kapi.ObjectMeta{
			Name: name,
		},
		Priority:                 &priority,
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
				Labels: map[string]string{
					"provider": "fabric8.io",
				},
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

func printAddClusterRoleToGroup(c *oclient.Client, f *cmdutil.Factory, roleName string, groupName string) (Result, error) {
	err := addClusterRoleToGroup(c, f, roleName, groupName)
	message := fmt.Sprintf("addClusterRoleToGroup %s %s", roleName, groupName)
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

// simulates: oadm policy add-cluster-role-to-group roleName groupName
func addClusterRoleToGroup(c *oclient.Client, f *cmdutil.Factory, roleName string, groupName string) error {
	options := policy.RoleModificationOptions{
		RoleName:            roleName,
		RoleBindingAccessor: policy.NewClusterRoleBindingAccessor(c),
		Groups:              []string{groupName},
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

	util.Errorf("\nUnknown version %s for %s\n", v, metadataUrl)
	util.Fatalf("Valid versions: %v\n", append(m.Versions, "latest"))
	return ""
}

func defaultExposeRule(c *k8sclient.Client, mini bool, useLoadBalancer bool) string {
	if mini {
		return nodePort
	}

	if util.TypeOfMaster(c) == util.Kubernetes {
		if useLoadBalancer {
			return loadBalancer
		}
		return ingress
	} else if util.TypeOfMaster(c) == util.OpenShift {
		return route
	}
	return ""
}
