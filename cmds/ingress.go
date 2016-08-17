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
	rapi "github.com/openshift/origin/pkg/route/api"
	rapiv1 "github.com/openshift/origin/pkg/route/api/v1"
	"github.com/spf13/cobra"
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/extensions"
	k8sclient "k8s.io/kubernetes/pkg/client/unversioned"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

func NewCmdIngress(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ingress",
		Short: "Creates any missing Ingress resources for services",
		Long:  `Creates any missing Ingress resources for Services which are of type LoadBalancer`,
		PreRun: func(cmd *cobra.Command, args []string) {
			showBanner()
		},
		Run: func(cmd *cobra.Command, args []string) {
			c, cfg := client.NewClient(f)
			ns, _, err := f.DefaultNamespace()
			if err != nil {
				util.Fatal("No default namespace")
				printResult("Get default namespace", Failure, err)
			} else {
				domain := cmd.Flags().Lookup(domainFlag).Value.String()

				util.Info("Setting up ingress on your ")
				util.Success(string(util.TypeOfMaster(c)))
				util.Info(" installation at ")
				util.Success(cfg.Host)
				util.Info(" in namespace ")
				util.Successf("%s at domain %s\n\n", ns, domain)
				err := createIngressForDomain(ns, domain, c, f)
				printError("Create Ingress", err)
			}
		},
	}
	cmd.PersistentFlags().StringP(domainFlag, "", defaultDomain(), "The domain to put the created routes inside")
	return cmd
}

func createIngressForDomain(ns string, domain string, c *k8sclient.Client, fac *cmdutil.Factory) error {
	rapi.AddToScheme(kapi.Scheme)
	rapiv1.AddToScheme(kapi.Scheme)

	ingressClient := c.Extensions().Ingress(ns)
	ingresses, err := ingressClient.List(kapi.ListOptions{})
	if err != nil {
		util.Errorf("Failed to load ingresses in namespace %s with error %v", ns, err)
		return err
	}
	rc, err := c.Services(ns).List(kapi.ListOptions{})
	if err != nil {
		util.Errorf("Failed to load services in namespace %s with error %v", ns, err)
		return err
	}
	var labels = make(map[string]string)
	labels["provider"] = "fabric8"

	items := rc.Items
	for _, service := range items {
		name := service.ObjectMeta.Name
		serviceSpec := service.Spec
		found := false

		// TODO we should probably add an annotation to disable ingress creation
		if name != "jenkinshift" {
			for _, ingress := range ingresses.Items {
				if ingress.GetName() == name {
					found = true
					break
				}
				// TODO look for other ingresses with different names?
				for _, rule := range ingress.Spec.Rules {
					http := rule.HTTP
					if http != nil {
						for _, path := range http.Paths {
							ruleService := path.Backend.ServiceName
							if ruleService == name {
								found = true
								break
							}
						}
					}
				}
			}
			if !found {
				ports := serviceSpec.Ports
				hostName := name + "." + ns + "." + domain
				if len(ports) > 0 {
					rules := []extensions.IngressRule{}
					for _, port := range ports {
						rule := extensions.IngressRule{
							Host: hostName,
							IngressRuleValue: extensions.IngressRuleValue{
								HTTP: &extensions.HTTPIngressRuleValue{
									Paths: []extensions.HTTPIngressPath{
										{
											Backend: extensions.IngressBackend{
												ServiceName: name,
												// we need to use target port until https://github.com/nginxinc/kubernetes-ingress/issues/41 is fixed
												//ServicePort: intstr.FromInt(port.Port),
												ServicePort: port.TargetPort,
											},
										},
									},
								},
							},
						}
						rules = append(rules, rule)
					}
					ingress := extensions.Ingress{
						ObjectMeta: kapi.ObjectMeta{
							Labels: labels,
							Name:   name,
						},
						Spec: extensions.IngressSpec{
							Rules: rules,
						},
					}
					// lets create the ingress
					_, err = ingressClient.Create(&ingress)
					if err != nil {
						util.Errorf("Failed to create the ingress %s with error %v", name, err)
						return err
					}
				}
			}
		}
	}
	return nil
}
