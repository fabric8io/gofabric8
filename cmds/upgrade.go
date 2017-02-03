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
	"io/ioutil"
	"net/http"

	"github.com/fabric8io/gofabric8/client"
	"github.com/fabric8io/gofabric8/util"
	oclient "github.com/openshift/origin/pkg/client"
	"github.com/spf13/cobra"
	kapi "k8s.io/kubernetes/pkg/api"
	k8sclient "k8s.io/kubernetes/pkg/client/unversioned"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"net"
	"net/url"
	"strings"
)

const (
	metadataUrlKey      = "metadataUrl"
	packageUrlPrefixKey = "packageUrlPrefix"
	versionFlag         = "version"
)

func NewCmdUpgrade(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade [name]",
		Short: "Upgrades the packages if there is a newer version available",
		Long:  `Upgrades the packages if there is a newer version available`,
		Run: func(cmd *cobra.Command, args []string) {
			c, cfg := client.NewClient(f)
			ns, _, err := f.DefaultNamespace()
			if err != nil {
				util.Fatal("No default namespace")
				printResult("Get default namespace", Failure, err)
			} else {
				all := cmd.Flags().Lookup(allFlag).Value.String() == "true"
				pv := cmd.Flags().Lookup(pvFlag).Value.String() == "true"
				version := cmd.Flags().Lookup(versionFlag).Value.String()
				domain := cmd.Flags().Lookup(domainFlag).Value.String()
				apiserver := cmd.Flags().Lookup(apiServerFlag).Value.String()

				if !all && len(args) == 0 {
					util.Failure("Either specify the names of packages to upgrade or use the `--all` command flag to upgrade all packages\n\n")
					return
				}

				ocl, _ := client.NewOpenShiftClient(cfg)
				initSchema()

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
				mini, err := util.IsMini()
				if err != nil {
					util.Failuref("error checking if minikube or minishift %v", err)
				}

				// default xip domain if local deployment incase users deploy ingress controller or router
				if mini && typeOfMaster == util.OpenShift {
					domain = ip + ".xip.io"
				}
				if len(apiserver) == 0 {
					apiserver = u.Host
				}

				util.Info("Checking packages for upgrade in your ")
				util.Success(string(util.TypeOfMaster(c)))
				util.Info(" installation at ")
				util.Success(cfg.Host)
				util.Info(" in namespace ")
				util.Successf("%s\n\n", ns)

				err = upgradePackages(ns, c, ocl, args, all, version, domain, apiserver, pv)
				if err != nil {
					util.Failuref("%v", err)
					util.Blank()
				}
			}
		},
	}
	cmd.PersistentFlags().Bool(allFlag, false, "If enabled then upgrade all packages")
	cmd.PersistentFlags().Bool(pvFlag, true, "if false will convert deployments to use Kubernetes emptyDir and disable persistence for core apps")
	cmd.PersistentFlags().Bool(updateFlag, false, "If the version ")
	cmd.PersistentFlags().String(versionFlag, "latest", "The version to upgrade to")
	cmd.PersistentFlags().StringP(domainFlag, "d", defaultDomain(), "The domain name to append to the service name to access web applications")
	cmd.PersistentFlags().String(apiServerFlag, "", "overrides the api server url")
	return cmd
}

func upgradePackages(ns string, c *k8sclient.Client, ocl *oclient.Client, args []string, all bool, version string, domain string, apiserver string, pv bool) error {
	selector, err := createPackageSelector()
	if err != nil {
		return err
	}
	list, err := c.ConfigMaps(ns).List(kapi.ListOptions{
		LabelSelector: *selector,
	})
	if err != nil {
		util.Errorf("Failed to load package in namespace %s with error %v", ns, err)
		return err
	}

	for _, p := range list.Items {
		name := p.Name
		include := all
		if !all {
			for _, arg := range args {
				if name == arg {
					include = true
					break
				}
			}
		}
		if !include {
			continue
		}
		metadataUrl := p.Data[metadataUrlKey]
		packageUrlPrefix := p.Data[packageUrlPrefixKey]
		if len(metadataUrl) == 0 {
			util.Warnf("Invalid package %s it is missing the `%s` data\n", name, metadataUrl)
			continue
		}
		if len(packageUrlPrefix) == 0 {
			util.Warnf("Invalid package %s it is missing the `%s` data\n", name, packageUrlPrefixKey)
			continue
		}

		newVersion := versionForUrl(version, metadataUrl)

		version := ""
		labels := p.Labels
		if labels != nil {
			version = labels["version"]
		}
		if newVersion != version {
			util.Info("Upgrading package ")
			util.Success(name)
			util.Info(" from version: ")
			util.Success(version)
			util.Info(" to version: ")
			util.Success(newVersion)
			util.Info("\n")

			uri := fmt.Sprintf(packageUrlPrefix, newVersion)
			typeOfMaster := util.TypeOfMaster(c)
			if typeOfMaster == util.Kubernetes {
				uri += "kubernetes.yml"
			} else {
				uri += "openshift.yml"
			}

			util.Infof("About to download package from %s\n", uri)
			yamlData := []byte{}
			format := "yaml"

			resp, err := http.Get(uri)
			if err != nil {
				util.Fatalf("Cannot load YAML package at %s got: %v", uri, err)
			}
			defer resp.Body.Close()
			yamlData, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				util.Fatalf("Cannot load YAML from %s got: %v", uri, err)
			}
			createTemplate(yamlData, format, name, ns, domain, apiserver, c, ocl, pv, false)

		} else {
			util.Info("package ")
			util.Success(name)
			util.Info(" is already on version: ")
			util.Success(newVersion)
			util.Info("\n")
		}
	}
	return nil
}
