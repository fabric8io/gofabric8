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
	"strings"

	"github.com/fabric8io/gofabric8/client"
	"github.com/fabric8io/gofabric8/util"
	oclient "github.com/openshift/origin/pkg/client"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
	k8sclient "k8s.io/kubernetes/pkg/client/unversioned"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/labels"
)

// NewCmdCleanUp delete all fabric8 apps, environments and configurations
func NewCmdCleanUpSystem(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "system",
		Short: "Hard delete all fabric8 apps, environments and configurations",
		Long:  `Hard delete all fabric8 apps, environments and configurations`,

		Run: func(cmd *cobra.Command, args []string) {

			var confirm string
			if cmd.Flags().Lookup(yesFlag).Value.String() == "true" {
				confirm = "y"
			} else {
				currentContext, err := util.GetCurrentContext()
				if err != nil {
					util.Fatalf("%s", err)
				}
				fmt.Fprintf(os.Stdout, `WARNING this is destructive and will remove ALL fabric8 apps, environments and configuration from cluster %s.  Continue? [y/N] `, currentContext)

				fmt.Scanln(&confirm)
			}

			if confirm == "y" {
				util.Info("Removing...\n")
				deleteSystem(f)
				return
			}
			util.Info("Cancelled")
		},
	}
	return cmd
}

func NewCmdCleanUpApp(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "app",
		Short: "Hard delete a specific fabric8 app and its environments and configurations",

		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				util.Fatal("You need to specify an app to delete.\n")
			}
			app := args[0]

			var confirm string
			if cmd.Flags().Lookup(yesFlag).Value.String() == "true" {
				confirm = "y"
			} else {
				fmt.Fprintf(os.Stdout, `WARNING this is destructive and will remove the fabric8 app %s, Continue? [y/N] `, app)
				fmt.Scanln(&confirm)
			}

			if confirm == "y" {
				util.Info("Removing...\n")
				selectormap := map[string]string{"project": app}

				deleteApp(f, selectormap)

				return
			}
			util.Info("Cancelled")
		},
	}
	return cmd
}

func NewCmdCleanUpTenant(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tenant",
		Short: "Hard delete your tenant removing all pipelines, apps, jobs and releases",

		Run: func(cmd *cobra.Command, args []string) {
			var confirm string
			if cmd.Flags().Lookup(yesFlag).Value.String() == "true" {
				confirm = "y"
			} else {
				fmt.Fprintf(os.Stdout, `WARNING this is destructive and will remove all your pipelines, apps, jobs and releases. Continue? [y/N] `)
				fmt.Scanln(&confirm)
			}

			if confirm == "y" {
				util.Info("Removing all tenant pipelines...\n")
				err := cleanUpTenant(f)
				if err != nil {
					util.Fatalf("Failed to remove tenant %v", err)
				}
				return
			}
			util.Info("Cancelled")
		},
	}
	return cmd
}

func cleanUpTenant(f *cmdutil.Factory) error {
	return nil
}

func deleteSystem(f *cmdutil.Factory) error {
	var oc *oclient.Client

	c, cfg := client.NewClient(f)
	ns, _, _ := f.DefaultNamespace()
	typeOfMaster := util.TypeOfMaster(c)
	selector, err := unversioned.LabelSelectorAsSelector(&unversioned.LabelSelector{MatchLabels: map[string]string{"provider": "fabric8"}})
	if err != nil {
		return err
	}

	if typeOfMaster == util.OpenShift {
		oc, _ = client.NewOpenShiftClient(cfg)
		initSchema()
		projects, err := oc.Projects().List(api.ListOptions{})
		cmdutil.CheckErr(err)

		ns = detectCurrentUserProject(ns, projects.Items, c)
	}

	deletePersistentVolumeClaims(c, ns, selector)

	if typeOfMaster == util.OpenShift {
		err = deleteProjects(oc, selector)
	} else {
		err = deleteNamespaces(c, selector)
	}

	if err != nil {
		util.Fatalf("%s\n", err)
	}

	util.Success("Successfully cleaned up\n")
	return nil
}

func deleteApp(f *cmdutil.Factory, selectormap map[string]string) error {
	isOpenShift := false
	c, cfg := client.NewClient(f)
	ns, _, _ := f.DefaultNamespace()

	typeOfMaster := util.TypeOfMaster(c)
	selector, err := unversioned.LabelSelectorAsSelector(&unversioned.LabelSelector{MatchLabels: selectormap})
	if err != nil {
		return err
	}
	if typeOfMaster == util.OpenShift {
		oc, _ := client.NewOpenShiftClient(cfg)
		initSchema()
		projects, err := oc.Projects().List(api.ListOptions{})
		cmdutil.CheckErr(err)

		ns = detectCurrentUserProject(ns, projects.Items, c)
		for _, todoNS := range []string{ns, ns + "-che", ns + "-jenkins"} {
			cleanUpOpenshiftResources(c, oc, todoNS, selector)
		}
		isOpenShift = true
	}

	for _, todoNS := range []string{ns, ns + "-che", ns + "-jenkins"} {
		cleanUpKubernetesResources(c, todoNS, selector, isOpenShift)
	}

	util.Success("Successfully cleaned up\n")
	return nil
}

func cleanUpOpenshiftResources(c *k8sclient.Client, oc *oclient.Client, ns string, selector labels.Selector) {

	err := deleteDeploymentConfigs(oc, ns, selector)
	if err != nil {
		util.Fatalf("%s\n", err)
	}

	err = deleteBuilds(oc, ns, selector)
	if err != nil {
		util.Fatalf("%s\n", err)
	}

	err = deleteBuildConfigs(oc, ns, selector)
	if err != nil {
		util.Fatalf("%s\n", err)
	}

	err = deleteRoutes(oc, ns, selector)
	if err != nil {
		util.Fatalf("%s\n", err)
	}
}

func cleanUpKubernetesResources(c *k8sclient.Client, ns string, selector labels.Selector, isOpenShift bool) {

	err := deleteDeployments(c, ns, selector)
	if err != nil {
		util.Fatalf("%s\n", err)
	}

	err = deleteReplicationControllers(c, ns, selector)
	if err != nil {
		util.Errorf("%s\n", err)
	}

	err = deleteReplicaSets(c, ns, selector)
	if err != nil {
		util.Errorf("%s\n", err)
	}

	err = deleteServices(c, ns, selector)
	if err != nil {
		util.Errorf("%s\n", err)
	}

	err = deleteSecrets(c, ns, selector)
	if err != nil {
		util.Errorf("%s\n", err)
	}

	if !isOpenShift {
		err = deleteIngress(c, ns, selector)
		if err != nil {
			util.Errorf("%s\n", err)
		}
	}

	err = deleteConfigMaps(c, ns, selector)
	if err != nil {
		util.Errorf("%s\n", err)
	}

	err = deleteServiceAccounts(c, ns, selector)
	if err != nil {
		util.Errorf("%s\n", err)
	}

	err = deletePods(c, ns, selector)
	if err != nil {
		util.Errorf("%s\n", err)
	}
}

func deleteDeploymentConfigs(oc *oclient.Client, ns string, selector labels.Selector) error {
	fmt.Printf("Deleting dc with label %s\n", selector.String())
	// use openshift binary as there's some client side logic to delete openshift DC resources
	e := exec.Command("oc", "delete", "dc", "-l", selector.String())
	err := e.Run()
	if err != nil {
		return errors.Wrap(err, "failed to delete DeploymentConfigs")
	}
	return nil
}

func deleteBuildConfigs(oc *oclient.Client, ns string, selector labels.Selector) error {
	fmt.Printf("Deleting bc with label %s\n", selector.String())
	// use openshift binary as there's some client side logic to delete openshift DC resources
	e := exec.Command("oc", "delete", "bc", "-l", selector.String())
	err := e.Run()
	if err != nil {
		return errors.Wrap(err, "failed to delete BuildConfigs")
	}
	return nil
}

func deleteBuilds(oc *oclient.Client, ns string, selector labels.Selector) error {
	fmt.Printf("Deleting builds with label %s\n", selector.String())
	// use openshift binary as there's some client side logic to delete openshift DC resources
	e := exec.Command("oc", "delete", "builds", "-l", selector.String())
	err := e.Run()
	if err != nil {
		return errors.Wrap(err, "failed to delete Builds")
	}
	return nil
}

func deleteRoutes(oc *oclient.Client, ns string, selector labels.Selector) error {
	fmt.Printf("Deleting routes with label %s\n", selector.String())
	// use openshift binary as there's some client side logic to delete openshift DC resources
	e := exec.Command("oc", "delete", "routes", "-l", selector.String())
	err := e.Run()
	if err != nil {
		return errors.Wrap(err, "failed to delete Routes")
	}
	return nil
}

func deleteDeployments(c *k8sclient.Client, ns string, selector labels.Selector) error {
	fmt.Printf("Deleting deployments with label %s\n", selector.String())
	e := exec.Command("kubectl", "delete", "deployments", "-l", selector.String())
	err := e.Run()
	if err != nil {
		return errors.Wrap(err, "failed to delete Deployments")
	}
	return nil

}

func deletePersistentVolumeClaims(c *k8sclient.Client, ns string, selector labels.Selector) (err error) {
	fmt.Printf("Deleting pvc with label %s\n", selector.String())
	pvcs, err := c.PersistentVolumeClaims(ns).List(api.ListOptions{LabelSelector: selector})
	if pvcs == nil {
		return
	}
	for _, item := range pvcs.Items {
		name := item.ObjectMeta.Name
		errd := c.PersistentVolumeClaims(ns).Delete(name)
		if errd != nil {
			util.Infof("Error deleting PVC %s\n", name)
		}
	}
	return
}

func deleteReplicationControllers(c *k8sclient.Client, ns string, selector labels.Selector) error {
	fmt.Printf("Deleting rc with label %s\n", selector.String())
	rcs, err := c.ReplicationControllers(ns).List(api.ListOptions{LabelSelector: selector})
	if err != nil {
		return err
	}
	for _, rc := range rcs.Items {
		err := c.ReplicationControllers(ns).Delete(rc.Name)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to delete ReplicationController %s\n", rc.Name))
		}
	}
	return nil
}

func deleteReplicaSets(c *k8sclient.Client, ns string, selector labels.Selector) error {
	fmt.Printf("Deleting ReplicaSets with label %s\n", selector.String())

	rsets, err := c.ReplicaSets(ns).List(api.ListOptions{LabelSelector: selector})
	if err != nil {
		return err
	}
	for _, rs := range rsets.Items {
		err := c.ReplicaSets(ns).Delete(rs.Name, nil)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to delete ReplicaSet %s\n", rs.Name))
		}
	}
	return nil
}

func deleteServices(c *k8sclient.Client, ns string, selector labels.Selector) error {
	fmt.Printf("Deleting services with label %s\n", selector.String())
	services, err := c.Services(ns).List(api.ListOptions{LabelSelector: selector})
	if err != nil {
		return err
	}
	for _, s := range services.Items {
		err := c.Services(ns).Delete(s.Name)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to delete Service %s\n", s.Name))
		}
	}
	return nil
}

func deleteSecrets(c *k8sclient.Client, ns string, selector labels.Selector) error {
	fmt.Printf("Deleting secrets with label %s\n", selector.String())
	secrets, err := c.Secrets(ns).List(api.ListOptions{LabelSelector: selector})
	if err != nil {
		return err
	}
	for _, s := range secrets.Items {
		err := c.Secrets(ns).Delete(s.Name)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to delete Secret %s\n", s.Name))
		}
	}
	return nil
}

func deleteIngress(c *k8sclient.Client, ns string, selector labels.Selector) error {
	fmt.Printf("Deleting ingress with label %s\n", selector.String())
	ing, err := c.Ingress(ns).List(api.ListOptions{LabelSelector: selector})
	if err != nil {
		return err
	}
	for _, i := range ing.Items {
		err := c.Ingress(ns).Delete(i.Name, nil)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to delete Ingress %s\n", i.Name))
		}
	}
	return nil
}

func deleteConfigMaps(c *k8sclient.Client, ns string, selector labels.Selector) error {
	fmt.Printf("Deleting configmaps with label %s\n", selector.String())
	cmps, err := c.ConfigMaps(ns).List(api.ListOptions{LabelSelector: selector})
	if err != nil {
		return err
	}
	for _, cm := range cmps.Items {
		err := c.ConfigMaps(ns).Delete(cm.Name)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to delete ConfigMap %s\n", cm.Name))
		}
	}
	return nil
}

func deleteServiceAccounts(c *k8sclient.Client, ns string, selector labels.Selector) error {
	fmt.Printf("Deleting serviceAccount with label %s\n", selector.String())
	sas, err := c.ServiceAccounts(ns).List(api.ListOptions{LabelSelector: selector})
	if err != nil {
		return err
	}
	for _, s := range sas.Items {
		err := c.ServiceAccounts(ns).Delete(s.Name)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to delete ServiceAccount %s\n", s.Name))
		}
	}
	return nil
}

func deletePods(c *k8sclient.Client, ns string, selector labels.Selector) error {
	fmt.Printf("Deleting pods with label %s\n", selector.String())
	pods, err := c.Pods(ns).List(api.ListOptions{LabelSelector: selector})
	if err != nil {
		return err
	}
	zero := int64(0)
	opt := &api.DeleteOptions{GracePeriodSeconds: &zero}
	for _, d := range pods.Items {
		err := c.Pods(ns).Delete(d.Name, opt)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to delete Pod %s\n", d.Name))
		}
	}
	return nil
}

func deleteProjects(oc *oclient.Client, selector labels.Selector) error {
	fmt.Printf("Deleting projects with label %s\n", selector.String())
	ns, err := oc.Projects().List(api.ListOptions{LabelSelector: selector})
	if err != nil {
		return err
	}
	for _, n := range ns.Items {
		err := oc.Projects().Delete(n.Name)
		if err != nil {
			// TODO(chmou): Handle with a special case, see https://goo.gl/vAFxaa
			if strings.HasSuffix(n.Name, "-jenkins") {
				err = nil
				continue
			}
			return errors.Wrap(err, fmt.Sprintf("failed to delete Project %s\n", n.Name))
		}
	}
	return nil
}

func deleteNamespaces(c *k8sclient.Client, selector labels.Selector) error {
	fmt.Printf("Deleting namespaces with label %s\n", selector.String())
	ns, err := c.Namespaces().List(api.ListOptions{LabelSelector: selector})
	if err != nil {
		return err
	}
	for _, n := range ns.Items {
		err := c.Namespaces().Delete(n.Name)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to delete Namespace %s\n", n.Name))
		}
	}
	return nil
}
