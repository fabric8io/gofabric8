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
	"github.com/spf13/cobra"
	"k8s.io/kubernetes/pkg/api"
	k8sclient "k8s.io/kubernetes/pkg/client/unversioned"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"time"
)

const (
	allFlag         = "all"
	timeoutFlag     = "timeout"
	sleepPeriodFlag = "sleep"
)

func NewCmdWaitFor(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wait-for",
		Short: "Waits for the listed deployments to be ready - useful for automation and testing",
		Long:  `Waits for the listed deployments to be ready - useful for automation and testing`,
		PreRun: func(cmd *cobra.Command, args []string) {
			showBanner()
		},
		Run: func(cmd *cobra.Command, args []string) {

			waitAll := cmd.Flags().Lookup(allFlag).Value.String() == "true"

			durationText := cmd.Flags().Lookup(timeoutFlag).Value.String()
			maxDuration, err := time.ParseDuration(durationText)
			if err != nil {
				util.Fatalf("Could not parse duration `%s` from flag --%s. Error %v\n", durationText, timeoutFlag, err)
			}
			durationText = cmd.Flags().Lookup(sleepPeriodFlag).Value.String()
			sleepMillis, err := time.ParseDuration(durationText)
			if err != nil {
				util.Fatalf("Could not parse duration `%s` from flag --%s. Error %v\n", durationText, sleepPeriodFlag, err)
			}

			if !waitAll && len(args) == 0 {
				util.Infof("Please specify one or more names of Deployment or DeploymentConfig resources or use the --%s flag to match all Deployments and DeploymentConfigs\n", allFlag)
				return
			}
			c, cfg := client.NewClient(f)
			oc, _ := client.NewOpenShiftClient(cfg)

			initSchema()

			fromNamespace := cmd.Flags().Lookup(fromNamespaceFlag).Value.String()
			if len(fromNamespace) == 0 {
				ns, _, err := f.DefaultNamespace()
				if err != nil {
					util.Fatal("No default namespace")
				}
				fromNamespace = ns
			}

			timer := time.NewTimer(maxDuration)
			go func() {
				<-timer.C
				util.Fatalf("Timed out waiting for Deployments. Waited: %v\n", maxDuration)
			}()

			util.Infof("Waiting for Deployments to be ready in namespace %s\n", fromNamespace)

			typeOfMaster := util.TypeOfMaster(c)

			for i := 0; i < 2; i++ {
				handleError(waitForDeployments(c, fromNamespace, waitAll, args, sleepMillis))
				if typeOfMaster == util.OpenShift {
					handleError(waitForDeploymentConfigs(oc, fromNamespace, waitAll, args, sleepMillis))
				}
			}
			timer.Stop()
			util.Infof("Deployments are ready now!\n")

		},
	}
	cmd.PersistentFlags().Bool(allFlag, false, "waits for all the Deployments or DeploymentConfigs to be ready")
	cmd.PersistentFlags().StringP(fromNamespaceFlag, "f", "", "the source namespace or uses the default namespace")
	cmd.PersistentFlags().String(timeoutFlag, "60m", "the maximum amount of time to wait for the Deployemnts to be ready before failing. e.g. an expression like: 1.5h, 12m, 10s")
	cmd.PersistentFlags().String(sleepPeriodFlag, "1s", "the sleep period while polling for Deployment status (e.g. 1s)")
	return cmd
}

func handleError(err error) {
	if err != nil {
		util.Fatalf("Failed to wait %v\n", err)
	}
}

func waitForDeployments(c *k8sclient.Client, ns string, waitAll bool, names []string, sleepMillis time.Duration) error {
	if waitAll {
		deployments, err := c.Extensions().Deployments(ns).List(api.ListOptions{})
		if err != nil {
			return err
		}
		for _, deployment := range deployments.Items {
			name := deployment.Name
			err = waitForDeployment(c, ns, name, sleepMillis)
			if err != nil {
				return err
			}
		}
	} else {
		for _, name := range names {
			err := waitForDeployment(c, ns, name, sleepMillis)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func waitForDeploymentConfigs(c *oclient.Client, ns string, waitAll bool, names []string, sleepMillis time.Duration) error {
	if waitAll {
		deployments, err := c.DeploymentConfigs(ns).List(api.ListOptions{})
		if err != nil {
			return err
		}
		for _, deployment := range deployments.Items {
			name := deployment.Name
			err = waitForDeploymentConfig(c, ns, name, sleepMillis)
			if err != nil {
				return err
			}
		}
	} else {
		for _, name := range names {
			err := waitForDeploymentConfig(c, ns, name, sleepMillis)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func waitForDeployment(c *k8sclient.Client, ns string, name string, sleepMillis time.Duration) error {
	util.Infof("Deployment %s waiting for it to be ready...\n", name)
	for {
		deployment, err := c.Extensions().Deployments(ns).Get(name)
		if err != nil {
			return err
		}
		available := deployment.Status.AvailableReplicas
		unavailable := deployment.Status.UnavailableReplicas
		if unavailable == 0 && available > 0 {
			util.Infof("DeploymentConfig %s now has %d available replicas\n", name, available)
			return nil
		}
		time.Sleep(sleepMillis)
	}
}

func waitForDeploymentConfig(c *oclient.Client, ns string, name string, sleepMillis time.Duration) error {
	util.Infof("DeploymentConfig %s waiting for it to be ready...\n", name)
	for {
		deployment, err := c.DeploymentConfigs(ns).Get(name)
		if err != nil {
			return err
		}
		available := deployment.Status.AvailableReplicas
		unavailable := deployment.Status.UnavailableReplicas
		if unavailable == 0 && available > 0 {
			util.Infof("DeploymentConfig %s now has %d available replicas\n", name, available)
			return nil
		}
		time.Sleep(sleepMillis)
	}
}
