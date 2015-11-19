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
	"github.com/fabric8io/gofabric8/client"
	"github.com/fabric8io/gofabric8/util"
	oclient "github.com/openshift/origin/pkg/client"
	rapi "github.com/openshift/origin/pkg/route/api"
	"github.com/spf13/cobra"
	kapi "k8s.io/kubernetes/pkg/api"
	k8sclient "k8s.io/kubernetes/pkg/client"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/labels"
)

func NewCmdRoutes(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "routes",
		Short: "Creates any missing Routes for services",
		Long:  `Creates any missing Route resources for Services which need to be exposed remotely`,
		PreRun: func(cmd *cobra.Command, args []string) {
			showBanner()
		},
		Run: func(cmd *cobra.Command, args []string) {
			c, cfg := client.NewClient(f)
			if util.TypeOfMaster(c) == util.Kubernetes {
				util.Fatal("Routes command is for OpenShift only")
			}
			oc, _ := client.NewOpenShiftClient(cfg)
			ns, _, err := f.DefaultNamespace()
			if err != nil {
				util.Fatal("No default namespace")
				printResult("Get default namespace", Failure, err)
			} else {
				util.Info("Setting up routes on your")
				util.Success(string(util.TypeOfMaster(c)))
				util.Info(" installation at ")
				util.Success(cfg.Host)
				util.Info(" in namespace ")
				util.Successf("%s\n\n", ns)

				domain := cmd.Flags().Lookup(domainFlag).Value.String()

				err := createRoutesForDomain(ns, domain, c, oc, f)
				printError("Create Routes", err)
			}
		},
	}
	cmd.PersistentFlags().StringP(domainFlag, "", defaultDomain(), "The domain to put the created routes inside")
	return cmd
}

func createRoutesForDomain(ns string, domain string, c *k8sclient.Client, oc *oclient.Client, fac *cmdutil.Factory) error {
	rc, err := c.Services(ns).List(labels.Everything())
	if err != nil {
		util.Errorf("Failed to load services in namespace %s with error %v", ns, err)
		return err
	}
	var labels = make(map[string]string)
	labels["provider"] = "fabric8"

	items := rc.Items
	for _, service := range items {
		// TODO use the external load balancer as a way to know if we should create a route?
		name := service.ObjectMeta.Name
		if name != "kubernetes" {
			routes := oc.Routes(ns)
			_, err = routes.Get(name)
			if err != nil {
				hostName := name + "." + domain
				route := rapi.Route{
					ObjectMeta: kapi.ObjectMeta{
						Labels: labels,
						Name:   name,
					},
					Host:        hostName,
					ServiceName: name,
				}
				// lets create the route
				_, err = routes.Create(&route)
				if err != nil {
					util.Errorf("Failed to create the route %s with error %v", name, err)
					return err
				}
			}
		}
	}
	return nil
}
