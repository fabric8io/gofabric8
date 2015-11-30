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
	"strings"

	"github.com/fabric8io/gofabric8/client"
	"github.com/fabric8io/gofabric8/util"
	oclient "github.com/openshift/origin/pkg/client"
	"github.com/spf13/cobra"
	k8sclient "k8s.io/kubernetes/pkg/client"
	"k8s.io/kubernetes/pkg/fields"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/labels"
	kutil "k8s.io/kubernetes/pkg/util"
)

type validateFunc func(c *k8sclient.Client, f *cmdutil.Factory) (Result, error)
type oValidateFunc func(c *oclient.Client, f *cmdutil.Factory) (Result, error)

func NewCmdValidate(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate your Kubernetes or OpenShift environment",
		Long:  `validate your Kubernetes or OpenShift environment`,
		PreRun: func(cmd *cobra.Command, args []string) {
			showBanner()
		},
		Run: func(cmd *cobra.Command, args []string) {
			c, cfg := client.NewClient(f)
			ns, _, _ := f.DefaultNamespace()
			util.Info("Validating your ")
			util.Success(string(util.TypeOfMaster(c)))
			util.Info(" installation at ")
			util.Success(cfg.Host)
			util.Info(" in namespace ")
			util.Successf("%s\n\n", ns)
			printValidationResult("Service account", validateServiceAccount, c, f)
			printValidationResult("Console", validateConsoleDeployment, c, f)

			if util.TypeOfMaster(c) == util.Kubernetes {
				printValidationResult("Templates", validateTemplateService, c, f)
			}

			if util.TypeOfMaster(c) == util.OpenShift {
				printValidationResult("Router", validateRouter, c, f)
				oc, _ := client.NewOpenShiftClient(cfg)
				printOValidationResult("Templates", validateTemplates, oc, f)
				printValidationResult("SecurityContextConstraints", validateSecurityContextConstraints, c, f)
			}

			printValidationResult("PersistentVolumeClaims", validatePersistenceVolumeClaims, c, f)
		},
	}

	return cmd
}

func printValidationResult(check string, v validateFunc, c *k8sclient.Client, f *cmdutil.Factory) {
	r, err := v(c, f)
	printResult(check, r, err)
}

func printOValidationResult(check string, v oValidateFunc, c *oclient.Client, f *cmdutil.Factory) {
	r, err := v(c, f)
	printResult(check, r, err)
}

func printError(check string, err error) {
	r := Success
	if err != nil {
		r = Failure
	}
	printResult(check, r, err)
}

func printResult(check string, r Result, err error) {
	if err != nil {
		r = Failure
	}
	padLen := 78 - len(check)
	pad := ""
	if padLen > 0 {
		pad = strings.Repeat(".", padLen)
	}
	util.Infof("%s%s", check, pad)
	if r == Failure {
		util.Failuref("%-2s", r)
	} else {
		util.Successf("%-2s", r)
	}
	if err != nil {
		util.Failuref("%v", err)
	}
	util.Blank()
}

func validateServiceAccount(c *k8sclient.Client, f *cmdutil.Factory) (Result, error) {
	ns, _, err := f.DefaultNamespace()
	if err != nil {
		return Failure, err
	}
	sa, err := c.ServiceAccounts(ns).Get("fabric8")
	if sa != nil {
		return Success, err
	}
	return Failure, err
}

func validateConsoleDeployment(c *k8sclient.Client, f *cmdutil.Factory) (Result, error) {
	ns, _, err := f.DefaultNamespace()
	if err != nil {
		return Failure, err
	}
	rc, err := c.ReplicationControllers(ns).Get("fabric8")
	if rc != nil {
		return Success, err
	}
	return Failure, err
}

func validatePersistenceVolumeClaims(c *k8sclient.Client, f *cmdutil.Factory) (Result, error) {
	ns, _, err := f.DefaultNamespace()
	if err != nil {
		return Failure, err
	}
	rc, err := c.PersistentVolumeClaims(ns).List(labels.Everything(), fields.Everything())
	if err != nil {
		util.Fatalf("Failed to get PersistentVolumeClaims, %s in namespace %s\n", err, ns)
	}
	if rc != nil {
		items := rc.Items
		pendingClaimNames := make([]string, 0, len(items))
		for _, item := range items {
			status := item.Status.Phase
			if status != "Bound" {
				pendingClaimNames = append(pendingClaimNames, item.ObjectMeta.Name)
			}
		}
		if len(pendingClaimNames) > 0 {
			util.Failuref("PersistentVolumeClaim not Bound for: %s. You need to create a PersistentVolume!\n", strings.Join(pendingClaimNames, ", "))
			util.Info(`
to generate a single node PersistentVolume then type something like this:


cat <<EOF | oc create -f -
---
kind: PersistentVolume
apiVersion: v1
metadata:
  name: fabric8
spec:
  accessModes:
    - ReadWrite
  capacity:
    storage: 1000
  hostPath:
    path: /opt/fabric8-data
EOF


`)
			return Failure, err
		}
		return Success, err
	}
	return Failure, err
}

func validateRouter(c *k8sclient.Client, f *cmdutil.Factory) (Result, error) {
	ns, _, err := f.DefaultNamespace()
	if err != nil {
		return Failure, err
	}
	requirement, err := labels.NewRequirement("router", labels.EqualsOperator, kutil.NewStringSet("router"))
	if err != nil {
		return Failure, err
	}
	label := labels.LabelSelector{*requirement}

	rc, err := c.ReplicationControllers(ns).List(label)
	if err != nil {
		util.Fatalf("Failed to get PersistentVolumeClaims, %s in namespace %s\n", err, ns)
	}
	if rc != nil {
		items := rc.Items
		if len(items) > 0 {
			return Success, err
		}
	}
	//util.Fatalf("No router running in namespace %s\n", ns)
	// TODO lets create a router
	return Failure, err
}

func validateSecurityContextConstraints(c *k8sclient.Client, f *cmdutil.Factory) (Result, error) {
	ns, _, err := f.DefaultNamespace()
	if err != nil {
		return Failure, err
	}
	name := Fabric8SCC
	if ns != "default" {
		name += "-" + ns
	}
	rc, err := c.SecurityContextConstraints().Get(name)
	if err != nil {
		util.Fatalf("Failed to get SecurityContextConstraints, %s in namespace %s\n", err, ns)
	}
	if rc != nil {
		return Success, err
	}
	return Failure, err
}

func validateTemplateService(c *k8sclient.Client, f *cmdutil.Factory) (Result, error) {
	ns, _, err := f.DefaultNamespace()
	if err != nil {
		return Failure, err
	}
	svc, err := c.Services(ns).Get("templates")
	if svc != nil {
		return Success, err
	}
	return Failure, err
}

func validateTemplates(c *oclient.Client, f *cmdutil.Factory) (Result, error) {
	ns, _, err := f.DefaultNamespace()
	if err != nil {
		return Failure, err
	}
	rc, err := c.Templates(ns).Get("management")
	if rc != nil {
		return Success, err
	}
	return Failure, err
}
