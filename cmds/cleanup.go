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
	"os"
	"os/exec"

	"github.com/fabric8io/gofabric8/client"
	"github.com/fabric8io/gofabric8/util"
	oclient "github.com/openshift/origin/pkg/client"
	"github.com/spf13/cobra"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
	k8sclient "k8s.io/kubernetes/pkg/client/unversioned"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/labels"
)

const ()

// NewCmdCleanUp delete all fabric8 apps, environments and configurations
func NewCmdCleanUp(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Hard delete all fabric8 apps, environments and configurations",
		Long:  `Hard delete all fabric8 apps, environments and configurations`,

		Run: func(cmd *cobra.Command, args []string) {

			fmt.Fprintf(os.Stdout, `WARNING this will remove all fabric8 apps, environments and configuration.  Continue? [y/N] `)

			var confirm string
			fmt.Scanln(&confirm)

			if confirm == "y" {
				util.Info("Removing...\n")
				cleanUp(f)
				return
			}
			util.Info("Cancelled")
		},
	}

	return cmd
}

func cleanUp(f *cmdutil.Factory) error {
	c, cfg := client.NewClient(f)
	ns, _, _ := f.DefaultNamespace()
	typeOfMaster := util.TypeOfMaster(c)
	selector, err := unversioned.LabelSelectorAsSelector(&unversioned.LabelSelector{MatchLabels: map[string]string{"provider": "fabric8"}})
	if err != nil {
		return err
	}
	if typeOfMaster == util.OpenShift {
		oc, _ := client.NewOpenShiftClient(cfg)
		cleanUpOpenshiftResources(c, oc, ns, selector)
	}

	cleanUpKubernetesResources(c, ns, selector)

	util.Success("Successfully cleaned up\n")
	return nil
}

func cleanUpOpenshiftResources(c *k8sclient.Client, oc *oclient.Client, ns string, selector labels.Selector) {

	err := deleteDeploymentConfigs(oc, ns, selector)
	if err != nil {
		util.Warnf("%s", err)
	}

	err = deleteBuilds(oc, ns, selector)
	if err != nil {
		util.Warnf("%s", err)
	}

	err = deleteBuildConfigs(oc, ns, selector)
	if err != nil {
		util.Warnf("%s", err)
	}

	err = deleteRoutes(oc, ns, selector)
	if err != nil {
		util.Warnf("%s", err)
	}
}

func cleanUpKubernetesResources(c *k8sclient.Client, ns string, selector labels.Selector) {

	err := deleteDeployments(c, ns, selector)
	if err != nil {
		util.Warnf("%s", err)
	}

	err = deleteReplicationControllers(c, ns, selector)
	if err != nil {
		util.Warnf("%s", err)
	}

	err = deleteReplicaSets(c, ns, selector)
	if err != nil {
		util.Warnf("%s", err)
	}

	err = deleteServices(c, ns, selector)
	if err != nil {
		util.Warnf("%s", err)
	}

	err = deleteSecrets(c, ns, selector)
	if err != nil {
		util.Warnf("%s", err)
	}

	err = deleteIngress(c, ns, selector)
	if err != nil {
		util.Warnf("%s", err)
	}

	err = deleteConfigMaps(c, ns, selector)
	if err != nil {
		util.Warnf("%s", err)
	}

	catalogSelector, err := unversioned.LabelSelectorAsSelector(&unversioned.LabelSelector{MatchLabels: map[string]string{"kind": "catalog"}})
	if err != nil {
		util.Errorf("%s", err)
	} else {
		err = deleteConfigMaps(c, ns, catalogSelector)
		if err != nil {
			util.Warnf("%s", err)
		}
	}

	err = deleteServiceAccounts(c, ns, selector)
	if err != nil {
		util.Warnf("%s", err)
	}

	err = deleteEnvironments(c, selector)
	if err != nil {
		util.Warnf("%s", err)
	}
	err = deletePods(c, ns, selector)
	if err != nil {
		util.Warnf("%s", err)
	}
}

func deleteDeploymentConfigs(oc *oclient.Client, ns string, selector labels.Selector) error {
	// use openshift binary as there's some client side logic to delete openshift DC resources
	e := exec.Command("oc", "delete", "dc", "-l", "provider=fabric8")
	err := e.Run()
	if err != nil {
		return err
	}
	return nil
}

func deleteBuildConfigs(oc *oclient.Client, ns string, selector labels.Selector) error {
	// use openshift binary as there's some client side logic to delete openshift DC resources
	e := exec.Command("oc", "delete", "bc", "-l", "provider=fabric8")
	err := e.Run()
	if err != nil {
		return err
	}
	return nil
}

func deleteBuilds(oc *oclient.Client, ns string, selector labels.Selector) error {
	// use openshift binary as there's some client side logic to delete openshift DC resources
	e := exec.Command("oc", "delete", "builds", "-l", "provider=fabric8")
	err := e.Run()
	if err != nil {
		return err
	}
	return nil
}

func deleteRoutes(oc *oclient.Client, ns string, selector labels.Selector) error {
	// use openshift binary as there's some client side logic to delete openshift DC resources
	e := exec.Command("oc", "delete", "routes", "-l", "provider=fabric8")
	err := e.Run()
	if err != nil {
		return err
	}
	return nil
}

func deleteDeployments(c *k8sclient.Client, ns string, selector labels.Selector) error {
	deployments, err := c.Deployments(ns).List(api.ListOptions{LabelSelector: selector})
	if err != nil {
		return err
	}
	zero := int64(0)
	opt := &api.DeleteOptions{GracePeriodSeconds: &zero}
	for _, d := range deployments.Items {
		c.Deployments(ns).Delete(d.Name, opt)
		if err != nil {
			return err
		}
	}
	return nil
}

func deleteReplicationControllers(c *k8sclient.Client, ns string, selector labels.Selector) error {
	rcs, err := c.ReplicationControllers(ns).List(api.ListOptions{LabelSelector: selector})
	if err != nil {
		return err
	}
	for _, rc := range rcs.Items {
		c.ReplicationControllers(ns).Delete(rc.Name)
		if err != nil {
			return err
		}
	}
	return nil
}

func deleteReplicaSets(c *k8sclient.Client, ns string, selector labels.Selector) error {
	rsets, err := c.ReplicaSets(ns).List(api.ListOptions{LabelSelector: selector})
	if err != nil {
		return err
	}
	for _, rs := range rsets.Items {
		c.ReplicaSets(ns).Delete(rs.Name, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func deleteServices(c *k8sclient.Client, ns string, selector labels.Selector) error {
	services, err := c.Services(ns).List(api.ListOptions{LabelSelector: selector})
	if err != nil {
		return err
	}
	for _, s := range services.Items {
		c.Services(ns).Delete(s.Name)
		if err != nil {
			return err
		}
	}
	return nil
}

func deleteSecrets(c *k8sclient.Client, ns string, selector labels.Selector) error {
	secrets, err := c.Secrets(ns).List(api.ListOptions{LabelSelector: selector})
	if err != nil {
		return err
	}
	for _, s := range secrets.Items {
		c.Secrets(ns).Delete(s.Name)
		if err != nil {
			return err
		}
	}
	return nil
}

func deleteIngress(c *k8sclient.Client, ns string, selector labels.Selector) error {
	ing, err := c.Ingress(ns).List(api.ListOptions{LabelSelector: selector})
	if err != nil {
		return err
	}
	for _, i := range ing.Items {
		c.Ingress(ns).Delete(i.Name, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func deleteConfigMaps(c *k8sclient.Client, ns string, selector labels.Selector) error {
	cmps, err := c.ConfigMaps(ns).List(api.ListOptions{LabelSelector: selector})
	if err != nil {
		return err
	}
	for _, cm := range cmps.Items {
		c.ConfigMaps(ns).Delete(cm.Name)
		if err != nil {
			return err
		}
	}
	return nil
}

func deleteServiceAccounts(c *k8sclient.Client, ns string, selector labels.Selector) error {
	sas, err := c.ServiceAccounts(ns).List(api.ListOptions{LabelSelector: selector})
	if err != nil {
		return err
	}
	for _, s := range sas.Items {
		c.ServiceAccounts(ns).Delete(s.Name)
		if err != nil {
			return err
		}
	}
	return nil
}

func deletePods(c *k8sclient.Client, ns string, selector labels.Selector) error {
	pods, err := c.Pods(ns).List(api.ListOptions{LabelSelector: selector})
	if err != nil {
		return err
	}
	zero := int64(0)
	opt := &api.DeleteOptions{GracePeriodSeconds: &zero}
	for _, d := range pods.Items {
		c.Pods(ns).Delete(d.Name, opt)
		if err != nil {
			return err
		}
	}
	return nil
}

func deleteEnvironments(c *k8sclient.Client, selector labels.Selector) error {
	ns, err := c.Namespaces().List(api.ListOptions{LabelSelector: selector})
	if err != nil {
		return err
	}
	for _, n := range ns.Items {
		c.Namespaces().Delete(n.Name)
		if err != nil {
			return err
		}
	}
	return nil
}
