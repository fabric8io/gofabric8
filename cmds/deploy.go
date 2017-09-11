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
	"bufio"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	goruntime "runtime"
	"strconv"
	"strings"
	"time"

	"github.com/fabric8io/gofabric8/client"
	"github.com/fabric8io/gofabric8/util"
	"github.com/ghodss/yaml"
	aapi "github.com/openshift/origin/pkg/authorization/api"
	aapiv1 "github.com/openshift/origin/pkg/authorization/api/v1"
	buildapi "github.com/openshift/origin/pkg/build/api"
	buildapiv1 "github.com/openshift/origin/pkg/build/api/v1"
	oclient "github.com/openshift/origin/pkg/client"
	"github.com/openshift/origin/pkg/cmd/admin/policy"
	"github.com/openshift/origin/pkg/cmd/server/bootstrappolicy"
	deployapi "github.com/openshift/origin/pkg/deploy/api"
	deployapiv1 "github.com/openshift/origin/pkg/deploy/api/v1"
	oauthapi "github.com/openshift/origin/pkg/oauth/api"
	oauthapiv1 "github.com/openshift/origin/pkg/oauth/api/v1"
	projectapi "github.com/openshift/origin/pkg/project/api"
	projectapiv1 "github.com/openshift/origin/pkg/project/api/v1"
	routeapi "github.com/openshift/origin/pkg/route/api"
	routeapiv1 "github.com/openshift/origin/pkg/route/api/v1"
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
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/apis/extensions/v1beta1"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	restclient "k8s.io/kubernetes/pkg/client/restclient"

	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/runtime"
)

const (
	systemMetadataUrl   = "io/fabric8/platform/packages/fabric8-system/maven-metadata.xml"
	platformMetadataUrl = "io/fabric8/platform/packages/fabric8-platform/maven-metadata.xml"
	ipaasMetadataUrl    = "io/fabric8/ipaas/platform/packages/ipaas-platform/maven-metadata.xml"

	systemPackageUrlPrefix    = "io/fabric8/platform/packages/fabric8-system/%[1]s/fabric8-system-%[1]s-"
	platformPackageUrlPrefix  = "io/fabric8/platform/packages/fabric8-platform/%[1]s/fabric8-platform-%[1]s-"
	consolePackageUrlPrefix   = "io/fabric8/platform/packages/console/%[1]s/console-%[1]s-"
	consolePackageMetadataUrl = "io/fabric8/platform/packages/console/maven-metadata.xml"
	ingressPackageUrlPrefix   = "io/fabric8/platform/packages/ingress/%[1]s/ingress-%[1]s-"
	kubelegoAppUrlPrefix      = "io/fabric8/platform/apps/kube-lego/%[1]s/kube-lego-%[1]s-"
	ipaasPackageUrlPrefix     = "io/fabric8/ipaas/platform/packages/ipaas-platform/%[1]s/ipaas-platform-%[1]s-"

	Fabric8SCC    = "fabric8"
	Fabric8SASSCC = "fabric8-sa-group"
	RestrictedSCC = "restricted"

	runFlag                = "app"
	httpFlag               = "http"
	useIngressFlag         = "ingress"
	useTLSAcmeFlag         = "tls-acme"
	useLoadbalancerFlag    = "loadbalancer"
	versionPlatformFlag    = "version"
	versioniPaaSFlag       = "version-ipaas"
	mavenRepoFlag          = "maven-repo"
	dockerRegistryFlag     = "docker-registry"
	archFlag               = "arch"
	pvFlag                 = "pv"
	updateFlag             = "update"
	packageFlag            = "package"
	legacyFlag             = "legacy"
	exposerFlag            = "exposer"
	githubClientSecretFlag = "github-client-secret"
	githubClientIDFlag     = "github-client-id"
	tlsAcmeEmailFlag       = "tls-acme-email"
	storageclassWaitFlag   = "wait-storageclass"

	systemPackage   = "system"
	platformPackage = "platform"
	consolePackage  = "console"
	iPaaSPackage    = "ipaas"
	ingressPackage  = "ingress"
	kubeLegoApp     = "kube-lego"

	fabric8Environments = "fabric8-environments"
	exposecontrollerCM  = "exposecontroller"

	ingress      = "Ingress"
	loadBalancer = "LoadBalancer"
	nodePort     = "NodePort"
	route        = "Route"

	exposeRule = "exposer"

	externalIPLabel = "fabric8.io/externalIP"

	gogsDefaultUsername = "gogsadmin"
	gogsDefaultPassword = "RedHat$1"

	minishiftDefaultUsername = "admin"
	minishiftDefaultPassword = "admin"

	latest           = "latest"
	mavenRepoDefault = "https://repo1.maven.org/maven2/"
	cdPipeline       = "cd-pipeline"

	fabric8SystemNamespace = "fabric8-system"
)

type Metadata struct {
	Release  string   `xml:"versioning>release"`
	Versions []string `xml:"versioning>versions>version"`
}

// Fabric8Deployment structure to work with the fabric8 deploy command
type DefaultFabric8Deployment struct {
	domain             string
	apiserver          string
	dockerRegistry     string
	arch               string
	mavenRepo          string
	appToRun           string
	packageName        string
	namespace          string
	exposer            string
	githubClientSecret string
	githubClientID     string
	http               bool
	tlsAcmeEmail       string
	templates          bool
	pv                 bool
	deployConsole      bool
	useIngress         bool
	useTLSAcme         bool
	useLoadbalancer    bool
	versionPlatform    string
	versioniPaaS       string
	yes                bool
	openConsole        bool
	legacyFlag         bool
	storageclassWait   bool
}

type createFunc func(c *clientset.Clientset, f cmdutil.Factory, name string) (Result, error)

func isFlag(cmd *cobra.Command, name string) bool {
	flag := cmd.Flags().Lookup(name)
	if flag != nil {
		return flag.Value.String() == "true"
	}
	return false
}

func NewCmdDeploy(f cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy fabric8 to your Kubernetes or OpenShift environment",
		Long:  `deploy fabric8 to your Kubernetes or OpenShift environment`,
		PreRun: func(cmd *cobra.Command, args []string) {
			showBanner()
		},
		Run: func(cmd *cobra.Command, args []string) {
			d := DefaultFabric8Deployment{
				domain:             cmd.Flags().Lookup(domainFlag).Value.String(),
				apiserver:          cmd.Flags().Lookup(apiServerFlag).Value.String(),
				arch:               cmd.Flags().Lookup(archFlag).Value.String(),
				mavenRepo:          cmd.Flags().Lookup(mavenRepoFlag).Value.String(),
				appToRun:           cmd.Flags().Lookup(runFlag).Value.String(),
				namespace:          cmd.Flags().Lookup(namespaceFlag).Value.String(),
				packageName:        cmd.Flags().Lookup(packageFlag).Value.String(),
				exposer:            cmd.Flags().Lookup(exposerFlag).Value.String(),
				githubClientSecret: cmd.Flags().Lookup(githubClientSecretFlag).Value.String(),
				githubClientID:     cmd.Flags().Lookup(githubClientIDFlag).Value.String(),
				tlsAcmeEmail:       cmd.Flags().Lookup(tlsAcmeEmailFlag).Value.String(),

				deployConsole:    cmd.Flags().Lookup(consoleFlag).Value.String() == "true",
				dockerRegistry:   cmd.Flags().Lookup(dockerRegistryFlag).Value.String(),
				useIngress:       cmd.Flags().Lookup(useIngressFlag).Value.String() == "true",
				http:             cmd.Flags().Lookup(httpFlag).Value.String() == "true",
				useTLSAcme:       cmd.Flags().Lookup(useTLSAcmeFlag).Value.String() == "true",
				templates:        cmd.Flags().Lookup(templatesFlag).Value.String() == "true",
				versionPlatform:  cmd.Flags().Lookup(versionPlatformFlag).Value.String(),
				versioniPaaS:     cmd.Flags().Lookup(versioniPaaSFlag).Value.String(),
				useLoadbalancer:  cmd.Flags().Lookup(useLoadbalancerFlag).Value.String() == "true",
				pv:               cmd.Flags().Lookup(pvFlag).Value.String() == "true",
				yes:              cmd.Flags().Lookup(yesFlag).Value.String() == "false",
				openConsole:      cmd.Flags().Lookup(openConsoleFlag).Value.String() == "true",
				legacyFlag:       cmd.Flags().Lookup(legacyFlag).Value.String() == "true",
				storageclassWait: isFlag(cmd, storageclassWaitFlag),
			}
			deploy(f, d)
		},
	}
	cmd.PersistentFlags().StringP(domainFlag, "d", defaultDomain(), "The domain name to append to the service name to access web applications")
	cmd.PersistentFlags().StringP(namespaceFlag, "n", "", "The namespace to deploy to (which is lazly created). Defaults to the current naemspace")
	cmd.PersistentFlags().String(apiServerFlag, "", "overrides the api server url")
	cmd.PersistentFlags().String(archFlag, goruntime.GOARCH, "CPU architecture for referencing Docker images with this as a name suffix")
	cmd.PersistentFlags().String(versionPlatformFlag, "latest", "The version to use for the Fabric8 Platform packages")
	cmd.PersistentFlags().String(versioniPaaSFlag, "latest", "The version to use for the Fabric8 iPaaS templates")
	cmd.PersistentFlags().String(mavenRepoFlag, mavenRepoDefault, "The maven repo used to find releases of fabric8")
	cmd.PersistentFlags().String(dockerRegistryFlag, "", "The docker registry used to download fabric8 images. Typically used to point to a staging registry")
	cmd.PersistentFlags().String(runFlag, "", "(Deprecated) The name of the fabric8 app to startup. e.g. use `--app=cd-pipeline` to run the main CI/CD pipeline app")
	cmd.PersistentFlags().String(exposerFlag, "", "The exposecontroller strategy such as Ingress, Router, NodePort, LoadBalancer")
	cmd.PersistentFlags().String(githubClientIDFlag, "", "The github OAuth Application Client ID. Defaults to $GITHUB_OAUTH_CLIENT_ID if not specified")
	cmd.PersistentFlags().String(githubClientSecretFlag, "", "The github OAuth Application Client Secret. Defaults to $GITHUB_OAUTH_CLIENT_SECRET if not specified")
	cmd.PersistentFlags().String(tlsAcmeEmailFlag, "", "Email address used to register and work with CertBot for automated signed cert ingress rules. Defaults to $TLS_ACME_EMAIL if not specified")
	// TODO re-enable when we're ready for 4.x
	//cmd.PersistentFlags().String(packageFlag, systemPackage, "The name of the package to startup such as 'system' for the 4.x version of fabric8 or 'platform' for 3.x. Otherwise specify a URL or local file of the YAML to install")
	cmd.PersistentFlags().String(packageFlag, platformPackage, "The name of the package to startup such as 'system' for the 4.x version of fabric8 or 'platform' for 3.x. Otherwise specify a URL or local file of the YAML to install")

	cmd.PersistentFlags().Bool(pvFlag, true, "if false will convert deployments to use Kubernetes emptyDir and disable persistence for core apps")
	cmd.PersistentFlags().Bool(templatesFlag, true, "Should the standard Fabric8 templates be installed?")
	cmd.PersistentFlags().Bool(consoleFlag, true, "Should the Fabric8 console be deployed?")
	cmd.PersistentFlags().Bool(httpFlag, false, "Should we generate HTTP rather than HTTPS routes?  Default `true` on minikube or minishift and `false for all else`")
	cmd.PersistentFlags().Bool(useIngressFlag, true, "Should Ingress NGINX controller be enabled by default when deploying to Kubernetes?")
	cmd.PersistentFlags().Bool(useTLSAcmeFlag, true, "Deploy TLS Acme impl kube-lego to auto generate signed certs for public ingress rules.  Requires tls-acme-email flag also. ")
	cmd.PersistentFlags().Bool(useLoadbalancerFlag, false, "Should Cloud Provider LoadBalancer be used to expose services when running to Kubernetes? (overrides ingress)")
	cmd.PersistentFlags().Bool(openConsoleFlag, true, "Should we wait an open the console?")
	cmd.PersistentFlags().Bool(legacyFlag, true, "Should we use the legacy installation mode for versions before 4.x of fabric8?")
	cmd.PersistentFlags().Bool(storageclassWaitFlag, true, "Should we wait for the storageclass resource to be ready on minikube before deploying?")

	return cmd
}

// GetDefaultFabric8Deployment create new instance of Fabric8Deployment
func GetDefaultFabric8Deployment() DefaultFabric8Deployment {
	d := DefaultFabric8Deployment{}
	d.domain = defaultDomain()
	d.arch = goruntime.GOARCH
	d.versioniPaaS = latest
	d.versionPlatform = latest
	d.mavenRepo = mavenRepoDefault
	d.pv = false
	d.useTLSAcme = false
	d.templates = true
	d.deployConsole = true
	d.useLoadbalancer = false
	d.openConsole = false
	d.packageName = platformPackage
	d.storageclassWait = true
	d.legacyFlag = true
	return d
}

func initSchema() {
	aapi.AddToScheme(api.Scheme)
	aapiv1.AddToScheme(api.Scheme)
	buildapi.AddToScheme(api.Scheme)
	buildapiv1.AddToScheme(api.Scheme)
	tapi.AddToScheme(api.Scheme)
	tapiv1.AddToScheme(api.Scheme)
	projectapi.AddToScheme(api.Scheme)
	projectapiv1.AddToScheme(api.Scheme)
	routeapi.AddToScheme(api.Scheme)
	routeapiv1.AddToScheme(api.Scheme)
	deployapi.AddToScheme(api.Scheme)
	deployapiv1.AddToScheme(api.Scheme)
	oauthapi.AddToScheme(api.Scheme)
	oauthapiv1.AddToScheme(api.Scheme)
}

func deploy(f cmdutil.Factory, d DefaultFabric8Deployment) {
	c, cfg := client.NewClient(f)
	ns, _, _ := f.DefaultNamespace()
	if len(d.namespace) > 0 {
		ns = d.namespace
	}

	domain := d.domain
	dockerRegistry := d.dockerRegistry

	mini, err := util.IsMini()
	if err != nil {
		util.Failuref("error checking if minikube or minishift %v", err)
	}

	packageName := d.packageName
	if len(packageName) == 0 {
		util.Fatalf("Missing value for --%s", packageFlag)
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

	// default nip domain if local deployment incase users deploy ingress controller or router
	if mini && typeOfMaster == util.OpenShift {
		domain = ip + ".nip.io"
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

	if len(d.appToRun) > 0 {
		util.Warn("Please note that the --app parameter is now deprecated.\n")
		util.Warn("Please use the --package argument to specify a package like `platform`, `console`, `ipaas` or to refer to a URL or file of the YAML package to install\n")
	}

	if strings.Contains(domain, "=") {
		util.Warnf("\nInvalid domain: %s\n\n", domain)
	} else if confirmAction(d.yes) {

		oc, _ := client.NewOpenShiftClient(cfg)

		initSchema()

		ensureNamespaceExists(c, oc, ns)

		// lets only check if the flag was changed from the default of true on the CLI
		legacyPackage := d.legacyFlag
		if d.legacyFlag {
			legacyPackage = isVersion3Package(packageName)
		}

		// lets only check if the flag was changed from the default of true on the CLI
		// and default to http on minikube + minishift
		http := d.http
		if !d.http && mini {
			http = true
		}

		if typeOfMaster == util.OpenShift {
			if legacyPackage {
				r, err := verifyRestrictedSecurityContextConstraints(c, f)
				printResult("SecurityContextConstraints restricted", r, err)
				r, err = deployFabric8SecurityContextConstraints(c, f, ns)
				printResult("SecurityContextConstraints fabric8", r, err)
				r, err = deployFabric8SASSecurityContextConstraints(c, f, ns)
				printResult("SecurityContextConstraints "+Fabric8SASSCC, r, err)

				printAddClusterRoleToUser(oc, f, "cluster-admin", "system:serviceaccount:"+ns+":fabric8")

				// TODO replace all of this with the necessary RoleBindings inside the OpenShift YAML...
				printAddClusterRoleToUser(oc, f, "cluster-admin", "system:serviceaccount:"+ns+":jenkins")

				printAddClusterRoleToUser(oc, f, "cluster-admin", "system:serviceaccount:"+ns+":configmapcontroller")
				printAddClusterRoleToUser(oc, f, "cluster-admin", "system:serviceaccount:"+ns+":exposecontroller")

				printAddClusterRoleToUser(oc, f, "cluster-reader", "system:serviceaccount:"+ns+":metrics")
				printAddClusterRoleToUser(oc, f, "cluster-reader", "system:serviceaccount:"+ns+":fluentd")

				printAddClusterRoleToGroup(oc, f, "cluster-reader", "system:serviceaccounts")

				printAddServiceAccount(c, f, "fluentd")
				printAddServiceAccount(c, f, "registry")
				printAddServiceAccount(c, f, "router")
			}
		} else {
			if mini {
				if packageName == systemPackage {
					err = validateSystemKubernetesVersion(c)
					if err != nil {
						util.Fatalf("Incompatible Kubernetes version: %v\n\n", err)
					}
				}
				addIngressInfraLabel(c, ns)
				// TODO output wildcard DNS information here?

				if d.storageclassWait && !legacyPackage {
					waitForStorageClass()
				}
			}
		}

		if typeOfMaster == util.Kubernetes && !mini && !legacyPackage {
			// deploy ingress controller
			if d.useIngress {
				// check if we already have an ingress controller running running
				CheckService("nginx-ingress", "nginx-ingress", c)
				if err != nil {
					ingressParams := make(map[string]string)
					err = deployPackage(ingressPackage, mavenRepo, domain, apiserver, legacyPackage, d, typeOfMaster, "nginx-ingress", c, oc, ingressParams)
					if err != nil {
						util.Fatalf("unable to deploy %s %v\n", "ingress", err)
					}
				}
				// wait for an external LoadBalancer IP to give exposecontroller
				util.Info("Waiting for External LoadBalancer IP Address for service nginx-ingress in namespace nginx-ingress.  This may take a few minutes.")
				util.Info("If you are running your own ingress controller please run deploy again passing the --ingress=false flag.")
				externalIP, err := WaitForExternalIPAddress("nginx-ingress", "nginx-ingress", c)
				if err != nil {
					util.Failuref("error getting external ip address for ingress service %v", err)
				}
				if d.domain == "" {
					domain = externalIP + ".nip.io"
				} else {
					util.Info("\n")
					util.Warnf("-------------------------\n")
					util.Info("\n")
					util.Warnf("Please configure your wildcard DNS for domain %s to point at external IP %s\n", d.domain, externalIP)
					util.Info("\n")
					util.Warnf("-------------------------\n")
					reader := bufio.NewReader(os.Stdin)
					fmt.Print("Enter to continue: ")
					reader.ReadString('\n')
				}
			}

			if d.useTLSAcme && !http {
				_, err := c.Deployments("kube-lego").Get("kube-lego")
				if err != nil {
					// deploy kube-lego
					kubeLegoParams := getTLSAcmeEmail(c, d.tlsAcmeEmail)
					err = deployPackage(kubeLegoApp, mavenRepo, domain, apiserver, legacyPackage, d, typeOfMaster, "kube-lego", c, oc, kubeLegoParams)
					if err != nil {
						util.Fatalf("unable to deploy %s %v\n", "kube-lego", err)
					}
				}
			}
		}

		// deploy the main package
		params := defaultParameters(c, d.exposer, d.githubClientID, d.githubClientSecret, ns, packageName, http, legacyPackage, d.useTLSAcme)
		err = deployPackage(packageName, mavenRepo, domain, apiserver, legacyPackage, d, typeOfMaster, ns, c, oc, params)
		if err != nil {
			util.Fatalf("unable to deploy %s %v\n", packageName, err)
		}

		if legacyPackage {
			externalNodeName := ""
			if typeOfMaster == util.Kubernetes {
				if !mini && d.useIngress {
					ensureNamespaceExists(c, oc, fabric8SystemNamespace)
					util.Infof("ns is %s\n", ns)
					runTemplate(c, oc, "ingress-nginx", ns, domain, apiserver, d.pv, true, params)
					externalNodeName = addIngressInfraLabel(c, ns)
				}
			}

			updateExposeControllerConfig(c, ns, apiserver, domain, mini, d.useLoadbalancer)

			printSummary(typeOfMaster, externalNodeName, ns, domain, c)
		}
		keycloakUrl := strings.TrimSuffix(FindServiceURL(ns, "keycloak", c, true), "/")
		if len(keycloakUrl) == 0 {
			util.Warn("\nCould not find keycloak service yet!\n")
		} else {
			clientID := params["GITHUB_OAUTH_CLIENT_ID"]
			callbackURL := keycloakUrl + "/auth/realms/fabric8/broker/github/endpoint"

			util.Info("\nPlease make sure your github OAuth Application Client ID: ")
			util.Success(clientID)
			util.Info(" points to the\nAuthorization callback URL: ")
			util.Success(callbackURL)
			util.Info("\n\n")
		}
		if !legacyPackage && typeOfMaster != util.Kubernetes {
			if mini {
				util.Info("\n\nCreating OAuthClient and adding role to fabric8-tenant\n")

				err = runCommand("bash", "-c", fmt.Sprintf(`cat <<EOF | oc create --as system:admin -f -
kind: OAuthClient
apiVersion: v1
metadata:
  name: fabric8-online-platform
secret: fabric8
redirectURIs:
- "%s/auth/realms/fabric8/broker/openshift-v3/endpoint"
grantMethod: prompt
EOF`, keycloakUrl))
				if err != nil {
					util.Fatalf("%s", err)
				}
				err = runCommand("oc", "adm", "policy", "add-cluster-role-to-user", "cluster-admin", fmt.Sprintf("system:serviceaccount:%s:init-tenant", ns), "--as", "system:admin")
				if err != nil {
					util.Fatalf("%s", err)
				}
			} else {
				util.Info("\n\nPlease can you invoke the following commands as a cluster admin\n\n")
				util.Infof(`cat <<EOF | oc create -f -
kind: OAuthClient
apiVersion: v1
metadata:
  name: fabric8-online-platform
secret: fabric8
redirectURIs:
- "%s/auth/realms/fabric8/broker/openshift-v3/endpoint"
grantMethod: prompt
EOF
oc adm policy add-cluster-role-to-user cluster-admin system:serviceaccount:%s:init-tenant

`, keycloakUrl, ns)
			}
		}

		if d.openConsole {
			openService(ns, "fabric8", c, false, true)
		}
	}
}

func deployPackage(packageName, mavenRepo, domain, apiserver string, legacyPackage bool, d DefaultFabric8Deployment, typeOfMaster util.MasterType, ns string, c *clientset.Clientset, oc *oclient.Client, params map[string]string) error {
	uri, err := getTemplateURI(packageName, mavenRepo, legacyPackage, d, typeOfMaster)
	if err != nil {
		util.Fatalf("Error getting URI for %s: %v\n\n", packageName, err)
	}
	yamlData, format, err := getYAMLData(uri)
	if err != nil {
		util.Fatalf("unable to get yaml data for URI %s %v\n", uri, err)
	}

	createTemplate(yamlData, format, packageName, ns, domain, apiserver, c, oc, d.pv, true, params)
	return nil
}

func getTemplateURI(packageName, mavenRepo string, legacyPackage bool, d DefaultFabric8Deployment, typeOfMaster util.MasterType) (string, error) {
	versionPlatform := ""
	baseUri := ""
	switch packageName {
	case "":
	case systemPackage:
		baseUri = systemPackageUrlPrefix
		versionPlatform = versionForUrl(d.versionPlatform, urlJoin(mavenRepo, systemMetadataUrl))
		logPackageVersion(packageName, versionPlatform)
	case platformPackage:
		baseUri = platformPackageUrlPrefix
		versionPlatform = versionForUrl(d.versionPlatform, urlJoin(mavenRepo, platformMetadataUrl))
		logPackageVersion(packageName, versionPlatform)
	case consolePackage:
		baseUri = consolePackageUrlPrefix
		versionPlatform = versionForUrl(d.versionPlatform, urlJoin(mavenRepo, consolePackageMetadataUrl))
		logPackageVersion(packageName, versionPlatform)
	case iPaaSPackage:
		baseUri = ipaasPackageUrlPrefix
		versionPlatform = versionForUrl(d.versioniPaaS, urlJoin(mavenRepo, ipaasMetadataUrl))
		logPackageVersion(packageName, versionPlatform)
	case ingressPackage:
		baseUri = ingressPackageUrlPrefix
		versionPlatform = versionForUrl(d.versionPlatform, urlJoin(mavenRepo, systemMetadataUrl))
		logPackageVersion(packageName, versionPlatform)
	case kubeLegoApp:
		baseUri = kubelegoAppUrlPrefix
		versionPlatform = versionForUrl(d.versionPlatform, urlJoin(mavenRepo, systemMetadataUrl))
		logPackageVersion(packageName, versionPlatform)
	default:
		baseUri = ""
	}
	uri := ""
	if len(baseUri) > 0 {
		uri = fmt.Sprintf(urlJoin(mavenRepo, baseUri), versionPlatform)

	} else {
		// lets assume the package is a file or a uri already
		if strings.Contains(packageName, "://") {
			uri = packageName
		} else {
			d, err := os.Stat(packageName)
			if err != nil {
				util.Fatalf("package %s not recognised and is not a local file %s\n", packageName, err)
			}
			if m := d.Mode(); m.IsDir() {
				util.Fatalf("package %s not recognised and is not a local file %s\n", packageName, err)
			}
			absFile, err := filepath.Abs(packageName)
			if err != nil {
				util.Fatalf("package %s not recognised and is not a local file %s\n", packageName, err)
			}
			uri = "file://" + absFile
		}
	}

	if typeOfMaster == util.Kubernetes {
		if !strings.HasPrefix(uri, "file://") {
			if legacyPackage {
				uri += "kubernetes.yml"
			} else {
				uri += "k8s-template.yml"
			}
		}

	} else {
		if !strings.HasPrefix(uri, "file://") {
			uri += "openshift.yml"
		}
	}
	return uri, nil
}

func getYAMLData(uri string) ([]byte, string, error) {
	yamlData := []byte{}
	format := "yaml"
	if strings.HasPrefix(uri, "file://") {
		fileName := strings.TrimPrefix(uri, "file://")
		if strings.HasSuffix(fileName, ".json") {
			format = "json"
		}
		yamlData, err := ioutil.ReadFile(fileName)
		if err != nil {
			util.Fatalf("Cannot load file %s got: %v", fileName, err)
		}
		return yamlData, format, nil
	}
	resp, err := http.Get(uri)
	if err != nil {
		util.Fatalf("Cannot load YAML package at %s got: %v", uri, err)
	}
	defer resp.Body.Close()
	yamlData, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		util.Fatalf("Cannot load YAML from %s got: %v", uri, err)
	}
	return yamlData, format, nil
}

func waitForStorageClass() {
	retry := 200

	util.Infof("Waiting for the default storageclass\n")

	err := RetryAfter(retry, func() (err error) {
		strToLookFor := "standard (default)"
		cmd := exec.Command("kubectl", "get", "StorageClass")
		found, err := waitCmdForText(cmd, strToLookFor)
		if found {
			err = nil
		} else {
			err = fmt.Errorf("No default storage class found yet")
		}
		return
	}, time.Second*2)

	if err != nil {
		util.Fatalf("No default storageclass found! Please try again later\n")
	}
}

func validateSystemKubernetesVersion(c *clientset.Clientset) error {
	info, err := c.ServerVersion()
	if err != nil {
		return fmt.Errorf("Could not load the ServerVersion due to %v", err)
	}
	minMajor := 1
	minMinor := 7
	majorText := info.Major
	minorText := info.Minor
	major, err := strconv.Atoi(majorText)
	if err != nil {
		util.Warnf("Could not parse major version text %s due to %v", majorText, err)
		util.Warnf("We require kubernetes %d.%d or later - will assume its OK!", minMajor, minMinor)
		return nil
	}
	minor, err := strconv.Atoi(minorText)
	if err != nil {
		util.Warnf("Could not parse major version text %s due to %v", minorText, err)
		util.Warnf("We require kubernetes %d.%d or later - will assume its OK!", minMajor, minMinor)
		return nil
	}
	util.Info("Kubernetes server version is: ")
	util.Successf("%d.%d\n\n", major, minor)

	if major > minMajor || (major == minMajor && minor >= minMinor) {
		return nil
	}
	return fmt.Errorf("Kubernetes Server version %d.%d is too old. We require version %d.%d or later", major, minor, minMajor, minMinor)
}

func logPackageVersion(packageName string, version string) {
	util.Info("Deploying package: ")
	util.Success(packageName)
	util.Info(" version: ")
	util.Success(version)
	util.Info("\n\n")
}

func updateExposeControllerConfig(c *clientset.Clientset, ns string, apiserver string, domain string, mini bool, useLoadBalancer bool) {
	// create a populate the exposecontroller config map
	cfgms := c.ConfigMaps(ns)

	_, err := cfgms.Get(exposecontrollerCM)
	if err == nil {
		util.Infof("\nRecreating configmap %s \n", exposecontrollerCM)
		err = cfgms.Delete(exposecontrollerCM, nil)
		if err != nil {
			printError("\nError deleting ConfigMap: "+exposecontrollerCM, err)
		}
	}
	apiserverAndPort := apiserver
	if !strings.Contains(apiserverAndPort, ":") {
		apiserverAndPort = apiserverAndPort + ":8443"
	}

	domainData := "domain: " + domain + "\n"
	exposeData := exposeRule + ": " + defaultExposeRule(c, mini, useLoadBalancer) + "\n"
	apiserverData := "apiserver: " + apiserverAndPort + "\n"
	namespaceData := "watch-current-namespace: true\n"
	configFile := map[string]string{
		"config.yml": domainData + exposeData + apiserverData + namespaceData,
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
}

func createMissingPVs(c *clientset.Clientset, ns string) {
	found, pvcs, pendingClaimNames := findPendingPVs(c, ns)
	if found {
		sshCommand := ""
		createPV(c, ns, pendingClaimNames, sshCommand)
		items := pvcs.Items
		for _, item := range items {
			status := item.Status.Phase
			if status == api.ClaimPending || status == api.ClaimLost {
				err := c.PersistentVolumeClaims(ns).Delete(item.ObjectMeta.Name, nil)
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

func printSummary(typeOfMaster util.MasterType, externalNodeName string, ns string, domain string, c *clientset.Clientset) {
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

	found, _ := checkIfPVCsPending(c, ns)
	if found {
		util.Errorf("There are pending PersistentVolumeClaims\n")
		util.Infof("If using a local cluster run `gofabric8 volumes` to create missing HostPath volumes\n")
		util.Infof("If using a remote cloud then enable dynamic persistence with a StorageClass.  For details see http://fabric8.io/guide/getStarted/persistence.html\n")
		util.Info("\n")
	}
	util.Infof("Downloading images and waiting to open the fabric8 console...\n")
	util.Info("\n")
	util.Info("-------------------------\n")
}

func getClientTypeName(typeOfMaster util.MasterType) string {
	if typeOfMaster == util.OpenShift {
		return "oc"
	}
	return "kubectl"
}

func addIngressInfraLabel(c *clientset.Clientset, ns string) string {
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
	firstNodeName := ""
	if !hasExistingExposeIPLabel && len(nodes.Items) > 0 {
		for _, node := range nodes.Items {

			changed = addLabelIfNotExist(&node.ObjectMeta, externalIPLabel, "true")
			if changed {
				_, err = nodeClient.Update(&node)
				if err != nil {
					printError("Failed to label node with ", err)
				}
				return node.Name
			}
			if len(firstNodeName) == 0 {
				firstNodeName = node.Name
			}
		}
	}
	if !changed && !hasExistingExposeIPLabel {
		if len(firstNodeName) > 0 {
			// lets try using kubectl
			cmd := exec.Command("kubectl", "label", "node", firstNodeName, "fabric8.io/externalIP=true", "--overwrite")
			if err := cmd.Run(); err == nil {
				return firstNodeName
			}
			util.Warnf("Unable to add label for ingress controller to run on a specific node, please add manually: kubectl label node [your node name] %s=true\n\n", externalIPLabel)
		}
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

func runTemplate(c *clientset.Clientset, oc *oclient.Client, appToRun string, ns string, domain string, apiserver string, pv bool, create bool, params map[string]string) {
	util.Info("\n\nInstalling: ")
	util.Successf("%s\n\n", appToRun)
	typeOfMaster := util.TypeOfMaster(c)
	if typeOfMaster == util.Kubernetes {
		jsonData, format, err := loadTemplateData(ns, appToRun, c, oc)
		if err != nil {
			printError("Failed to load app "+appToRun, err)
		}
		createTemplate(jsonData, format, appToRun, ns, domain, apiserver, c, oc, pv, true, params)
	} else {
		tmpl, err := oc.Templates(ns).Get(appToRun)
		if err != nil {
			printError("Failed to load template "+appToRun, err)
		}
		util.Infof("Loaded template with %d objects", len(tmpl.Objects))
		processTemplate(tmpl, ns, domain, apiserver, params)

		objectCount := len(tmpl.Objects)

		util.Infof("Creating "+appToRun+" template resources from %d objects\n", objectCount)
		for _, o := range tmpl.Objects {
			err = processItem(c, oc, &o, ns, pv, create)
		}
	}
}

func loadTemplateData(ns string, templateName string, c *clientset.Clientset, oc *oclient.Client) ([]byte, string, error) {
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
}

func createTemplate(jsonData []byte, format, templateName, ns, domain, apiserver string, c *clientset.Clientset, oc *oclient.Client, pv bool, create bool, params map[string]string) {
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

	processTemplate(&tmpl, ns, domain, apiserver, params)

	objectCount := len(tmpl.Objects)

	linker := runtime.SelfLinker(meta.NewAccessor())

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
			processData(jsonData, format, templateName, ns, c, oc, pv, create)
		} else {
			for _, i := range v1List.Items {
				data := i.Raw
				if data == nil {
					util.Infof("no data!\n")
					continue
				}
				kind := ""
				name := ""
				o := i.Object
				if o != nil {
					name, err = linker.Name(o)
					if err != nil {
						util.Warnf("Could not find resource name for %s\n", templateName)
					}
					objectKind := o.GetObjectKind()
					if objectKind != nil {
						groupVersionKind := objectKind.GroupVersionKind()
						kind = groupVersionKind.Kind
					}
				}
				if len(kind) == 0 {
					processData(data, format, templateName, ns, c, oc, pv, create)
				} else {
					// TODO how to find the Namespace?
					err = processResource(c, oc, data, ns, name, kind, create)
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
			err = processItem(c, oc, &o, ns, pv, create)
		}
	}
	if err != nil {
		printResult(templateName, Failure, err)
	} else {
		printResult(templateName, Success, nil)
	}
}

func getName(o runtime.Object) (string, error) {
	linker := runtime.SelfLinker(meta.NewAccessor())
	return linker.Name(o)
}

func processTemplate(tmpl *tapi.Template, ns, domain, apiserver string, params map[string]string) {
	generators := map[string]generator.Generator{
		"expression": generator.NewExpressionValueGenerator(rand.New(rand.NewSource(time.Now().UnixNano()))),
	}
	p := template.NewProcessor(generators)

	ip, port, err := net.SplitHostPort(apiserver)
	if err != nil && !strings.Contains(err.Error(), "missing port in address") {
		util.Fatalf("%s", err)
	}

	if len(domain) == 0 {
		domain = ip + ".nip.io"
	}

	params["NAMESPACE"] = ns
	params["APISERVER_HOSTPORT"] = apiserver
	params["APISERVER"] = ip
	params["NODE_IP"] = ip
	params["OAUTH_AUTHORIZE_PORT"] = port
	params["DOMAIN"] = domain

	for k, v := range params {
		found := false
		for i, param := range tmpl.Parameters {
			if param.Name == k {
				tmpl.Parameters[i].Value = v
				found = true
				break
			}

		}
		if !found {
			tmpl.Parameters = append(tmpl.Parameters, tapi.Parameter{
				Name:  k,
				Value: v,
			})
		}
	}
	for _, param := range tmpl.Parameters {
		util.Infof("Template %s = %s\n", param.Name, param.Value)
	}

	errorList := p.Process(tmpl)
	for _, errInfo := range errorList {
		util.Errorf("Processing template field %s got error %s\n", errInfo.Field, errInfo.Detail)
	}
}

func processData(jsonData []byte, format string, templateName string, ns string, c *clientset.Clientset, oc *oclient.Client, pv bool, create bool) {
	// lets check if its an RC / ReplicaSet or something
	o, groupVersionKind, err := api.Codecs.UniversalDeserializer().Decode(jsonData, nil, nil)
	if err != nil {
		printResult(templateName, Failure, err)
		return
	}
	name, err := getName(o)
	if err != nil {
		printResult(templateName, Failure, err)
		return
	}
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
		err = processResource(c, oc, jsonData, ns, name, kind, create)
		if err != nil {
			util.Warnf("Failed to create %s: %v\n", kind, err)
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

func processItem(c *clientset.Clientset, oc *oclient.Client, item *runtime.Object, ns string, pv bool, create bool) error {
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
			// namespace := metadata["namespace"]
			// switch namespace := namespace.(type) {
			// case string:
			// 	//util.Infof("Custom namespace '%s'\n", namespace)
			// 	if len(namespace) <= 0 {
			// 		// TODO why is the namespace empty?
			// 		// lets default the namespace to the default gogs namespace
			// 		namespace = "user-secrets-source-admin"
			// 		if metadata["name"] == "ingress-nginx" || metadata["name"] == "nginx-config" {
			// 			namespace = fabric8SystemNamespace
			// 		}
			// 	}
			// 	ns = namespace

			// 	// lets check that this new namespace exists
			// 	err := ensureNamespaceExists(c, oc, ns)
			// 	if err != nil {
			// 		printErr(err)
			// 	}
			// }
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
		if len(name) == 0 {
			name, err = getName(o)
			if err != nil {
				return err
			}
		}
		return processResource(c, oc, b, ns, name, kind, create)
	default:
		util.Infof("Unknown type %v\n", reflect.TypeOf(item))
	}
	return nil
}

func ensureNamespaceExists(c *clientset.Clientset, oc *oclient.Client, ns string) error {
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

func processResource(c *clientset.Clientset, oc *oclient.Client, b []byte, ns string, name string, kind string, create bool) error {
	util.Infof("Processing resource kind: %s in namespace %s name %s\n", kind, ns, name)
	var paths []string
	kinds := strings.ToLower(kind + "s")
	if kind == "Deployment" {
		paths = []string{"apis", "extensions/v1beta1", "namespaces", ns, kinds}
	} else if kind == "BuildConfig" || kind == "DeploymentConfig" || kind == "Template" || kind == "PolicyBinding" || kind == "Role" || kind == "RoleBinding" || kind == "Route" {
		paths = []string{"oapi", "v1", "namespaces", ns, kinds}
	} else if kind == "OAuthClient" || kind == "Project" || kind == "ProjectRequest" {
		paths = []string{"oapi", "v1", kinds}
	} else if kind == "Namespace" {
		paths = []string{"api", "v1", "namespaces"}
	} else if kind == "CustomResourceDefinition" {
		paths = []string{"apis", "apiextensions.k8s.io", "v1beta1", "customresourcedefinitions"}
	} else {
		paths = []string{"api", "v1", "namespaces", ns, kinds}
	}

	updatePaths := append(paths, name)
	if !create {
		// lets check if the resource already exists
		req2 := c.Core().RESTClient().Get().AbsPath(updatePaths...)
		res2 := req2.Do()
		data, err := res2.Raw()
		if err != nil {
			util.Infof("Error looking up resource %s got %v\n", name, err)
			create = true
		} else {
			var statusCode int
			res2.StatusCode(&statusCode)
			if statusCode < 200 || statusCode >= 300 {
				util.Infof("Could not find resource, got status %d so creating rather than updating the resouce\n", statusCode)
				create = true
			} else if kind == "PersistentVolumeClaim" {
				util.Infof("Ignoring the %s resource %s as one already exists\n", kind, name)
				return nil
			} else if kind == "Deployment" {
				var old extensions.Deployment
				var new extensions.Deployment
				err = yaml.Unmarshal(data, &old)
				if err != nil {
					return fmt.Errorf("Cannot unmarshal current Deployment %s. error: %v", name, err)
				}
				err = yaml.Unmarshal(b, &new)
				if err != nil {
					return fmt.Errorf("Cannot unmarshal new Deployment %s. error: %v", name, err)
				}

				// now lets copy across any missing annotations / labels / data values
				old.Labels = overwriteStringMaps(old.Labels, new.Labels)
				old.Annotations = overwriteStringMaps(old.Annotations, new.Annotations)
				old.Spec = new.Spec
				old.Name = name
				_, err = c.Extensions().Deployments(ns).Update(&old)
				if err != nil {
					return fmt.Errorf("Failed to update Deployment %s. Error %v", name, err)
				}
				return nil
			} else if kind == "DeploymentConfig" {
				var old deployapiv1.DeploymentConfig
				var new deployapiv1.DeploymentConfig
				err = yaml.Unmarshal(data, &old)
				if err != nil {
					return fmt.Errorf("Cannot unmarshal current DeploymentConfig %s. error: %v", name, err)
				}
				err = yaml.Unmarshal(b, &new)
				if err != nil {
					return fmt.Errorf("Cannot unmarshal new DeploymentConfig %s. error: %v", name, err)
				}

				// now lets copy across any missing annotations / labels / data values
				new.Labels = overwriteStringMaps(new.Labels, old.Labels)
				new.Annotations = overwriteStringMaps(new.Annotations, old.Annotations)
				new.ResourceVersion = old.ResourceVersion
				new.Name = name

				// lets convert the v1 to the api version
				var converted deployapi.DeploymentConfig
				err = api.Scheme.Convert(&new, &converted, nil)
				if err != nil {
					return fmt.Errorf("Cannot convert v1 to api DeploymentConfig %s due to: %v", name, err)
				}

				_, err = oc.DeploymentConfigs(ns).Update(&converted)
				if err != nil {
					return fmt.Errorf("Failed to update DeploymentConfig %s. Error %v resource %+v", name, err, converted)
				}
				return nil
			} else if kind == "Service" {
				var old api.Service
				var new api.Service
				err = yaml.Unmarshal(data, &old)
				if err != nil {
					return fmt.Errorf("Cannot unmarshal current Service %s. error: %v", name, err)
				}
				err = yaml.Unmarshal(b, &new)
				if err != nil {
					return fmt.Errorf("Cannot unmarshal new Service %s. error: %v", name, err)
				}

				// now lets copy across any missing annotations / labels / data values
				old.Labels = overwriteStringMaps(old.Labels, new.Labels)
				old.Annotations = overwriteStringMaps(old.Annotations, new.Annotations)
				oldClusterIP := old.Spec.ClusterIP
				old.Spec = new.Spec
				old.Spec.ClusterIP = oldClusterIP
				old.Name = name
				_, err = c.Services(ns).Update(&old)
				if err != nil {
					return fmt.Errorf("Failed to update Service %s. Error %v", name, err)
				}
				return nil
			} else if kind == "ConfigMap" {
				var old api.ConfigMap
				var new api.ConfigMap
				err = yaml.Unmarshal(data, &old)
				if err != nil {
					return fmt.Errorf("Cannot unmarshal current ConfigMap %s. error: %v", name, err)
				}
				err = yaml.Unmarshal(b, &new)
				if err != nil {
					return fmt.Errorf("Cannot unmarshal new ConfigMap %s. error: %v", name, err)
				}

				// now lets copy across any missing annotations / labels / data values
				old.Data = mergeStringMaps(old.Data, new.Data)
				old.Labels = overwriteStringMaps(old.Labels, new.Labels)
				old.Annotations = overwriteStringMaps(old.Annotations, new.Annotations)
				old.Name = name
				_, err = c.ConfigMaps(ns).Update(&old)
				if err != nil {
					return fmt.Errorf("Failed to update ConfigMap %s. Error %v", name, err)
				}
				return nil
			} else if kind == "Secret" {
				var old api.Secret
				var new api.Secret
				err = yaml.Unmarshal(data, &old)
				if err != nil {
					return fmt.Errorf("Cannot unmarshal current Secret %s. error: %v", name, err)
				}
				err = yaml.Unmarshal(b, &new)
				if err != nil {
					return fmt.Errorf("Cannot unmarshal new Secret %s. error: %v", name, err)
				}

				// now lets copy across any missing annotations / labels / data values
				old.Data = mergeByteMaps(old.Data, new.Data)
				old.Labels = overwriteStringMaps(old.Labels, new.Labels)
				old.Annotations = overwriteStringMaps(old.Annotations, new.Annotations)
				old.Name = name
				_, err = c.Secrets(ns).Update(&old)
				if err != nil {
					return fmt.Errorf("Failed to update Secret %s. Error %v", name, err)
				}
				return nil
			}
		}
	}
	if !create {
		paths = updatePaths
	}
	var req *restclient.Request
	if create {
		req = c.Core().RESTClient().Post().Body(b)

	} else {
		req = c.Core().RESTClient().Put().Body(b)
	}
	req.AbsPath(paths...)
	res := req.Do()
	var err error = nil
	if res.Error() != nil {
		err = res.Error()
	}
	var statusCode int
	res.StatusCode(&statusCode)
	if statusCode < 200 || statusCode > 207 {
		return fmt.Errorf("Failed to create %s: %d %v", kind, statusCode, err)
	}
	return nil
}

// overwriteStringMaps overrides all values ignoring whatever values are in the original map
func overwriteStringMaps(result map[string]string, overrides map[string]string) map[string]string {
	if result == nil {
		if overrides == nil {
			return map[string]string{}
		} else {
			return overrides
		}
	}
	for k, v := range overrides {
		result[k] = v
	}
	return result
}

// mergeStringMaps merges the overrides onto the result returning the new map with the results
func mergeStringMaps(result map[string]string, overrides map[string]string) map[string]string {
	if result == nil {
		if overrides == nil {
			return map[string]string{}
		} else {
			return overrides
		}
	}
	for k, v := range overrides {
		if len(result[k]) == 0 {
			result[k] = v
		}
	}
	return result
}

// mergeByteMaps merges the overrides onto the result returning the new map with the results
func mergeByteMaps(result map[string][]byte, overrides map[string][]byte) map[string][]byte {
	if result == nil {
		if overrides == nil {
			return map[string][]byte{}
		} else {
			return overrides
		}
	}
	for k, v := range overrides {
		if len(result[k]) == 0 {
			result[k] = v
		}
	}
	return result
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
func deployFabric8SecurityContextConstraints(c *clientset.Clientset, f cmdutil.Factory, ns string) (Result, error) {
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
		err = c.SecurityContextConstraints().Delete(name, nil)
		if err != nil {
			return Failure, err
		}
	}
	_, err = c.SecurityContextConstraints().Create(&scc)
	if err != nil {
		util.Errorf("Cannot create SecurityContextConstraints: %v\n", err)
		util.Errorf("Failed to create SecurityContextConstraints %v in namespace %s: %v\n", scc, ns, err)
		return Failure, err
	}
	util.Infof("SecurityContextConstraints %s is setup correctly\n", name)
	return Success, err
}

func deployFabric8SASSecurityContextConstraints(c *clientset.Clientset, f cmdutil.Factory, ns string) (Result, error) {
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
		err = c.SecurityContextConstraints().Delete(name, nil)
		if err != nil {
			return Failure, err
		}
	}
	_, err = c.SecurityContextConstraints().Create(&scc)
	if err != nil {
		util.Errorf("Cannot create SecurityContextConstraints: %v\n", err)
		util.Errorf("Failed to create SecurityContextConstraints %v in namespace %s: %v\n", scc, ns, err)
		return Failure, err
	}
	util.Infof("SecurityContextConstraints %s is setup correctly\n", name)
	return Success, err
}

// Ensure that the `restricted` SecurityContextConstraints has the RunAsUser set to RunAsAny
//
// if `restricted does not exist lets create it
// otherwise if needed lets modify the RunAsUser
func verifyRestrictedSecurityContextConstraints(c *clientset.Clientset, f cmdutil.Factory) (Result, error) {
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
			util.Errorf("Failed to update SecurityContextConstraints %v in namespace %s: %v\n", rc, ns, err)
			return Failure, err
		}
		util.Infof("SecurityContextConstraints %s is updated to enable fabric8\n", name)
	} else {
		util.Infof("SecurityContextConstraints %s is configured correctly\n", name)
	}
	return Success, err
}

func printAddServiceAccount(c *clientset.Clientset, f cmdutil.Factory, name string) (Result, error) {
	r, err := addServiceAccount(c, f, name)
	message := fmt.Sprintf("addServiceAccount %s", name)
	printResult(message, r, err)
	return r, err
}

func addServiceAccount(c *clientset.Clientset, f cmdutil.Factory, name string) (Result, error) {
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

func printAddClusterRoleToUser(c *oclient.Client, f cmdutil.Factory, roleName string, userName string) (Result, error) {
	err := addClusterRoleToUser(c, f, roleName, userName)
	message := fmt.Sprintf("addClusterRoleToUser %s %s", roleName, userName)
	r := Success
	if err != nil {
		r = Failure
	}
	printResult(message, r, err)
	return r, err
}

func printAddClusterRoleToGroup(c *oclient.Client, f cmdutil.Factory, roleName string, groupName string) (Result, error) {
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
func addClusterRoleToUser(c *oclient.Client, f cmdutil.Factory, roleName string, userName string) error {
	options := policy.RoleModificationOptions{
		RoleName:            roleName,
		RoleBindingAccessor: policy.NewClusterRoleBindingAccessor(c),
		Users:               []string{userName},
	}

	return options.AddRole()
}

// simulates: oadm policy add-cluster-role-to-group roleName groupName
func addClusterRoleToGroup(c *oclient.Client, f cmdutil.Factory, roleName string, groupName string) error {
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

func loadMetadata(metadataUrl string) (*Metadata, error) {
	resp, err := http.Get(metadataUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// read xml http response
	xmlData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var m Metadata
	err = xml.Unmarshal(xmlData, &m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func versionForUrl(v string, metadataUrl string) string {
	m, err := loadMetadata(metadataUrl)
	if err != nil {
		util.Fatalf("Failed to get metadata: %v on URL %s", err, metadataUrl)
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

func defaultExposeRule(c *clientset.Clientset, mini bool, useLoadBalancer bool) string {
	if util.TypeOfMaster(c) == util.OpenShift {
		return route
	}
	if mini {
		return nodePort
	}
	if useLoadBalancer {
		return loadBalancer
	}
	return ingress
}

func checkIfPVCsPending(c *clientset.Clientset, ns string) (bool, error) {
	timeout := time.After(20 * time.Second)
	tick := time.Tick(2 * time.Second)
	util.Info("Checking if PersistentVolumeClaims bind to a PersistentVolume ")
	// Keep trying until we're timed out or got a result or got an error
	for {
		select {
		// Got a timeout! fail with a timeout error
		case <-timeout:
			return true, errors.New("timed out")
		// Got a tick, check if PVc have bound
		case <-tick:
			found, _, _ := findPendingPVs(c, ns)
			if !found {
				util.Info("\n")
				return false, nil
			}
			util.Info(".")
			// retry
		}
	}
}

func waitCmdForText(cmd *exec.Cmd, str string) (ret bool, err error) {
	// NOTE(chmou): Make sure the process is not stuck
	timer := time.AfterFunc(90*time.Second, func() {
		cmd.Process.Kill()
	})

	raw, err := cmd.Output()
	if err != nil {
		return
	}

	s := bufio.NewScanner(bytes.NewReader(raw))
	for s.Scan() {
		if strings.HasPrefix(s.Text(), str) {
			ret = true
		}
	}

	timer.Stop()
	return
}
