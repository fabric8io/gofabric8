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
	"github.com/spf13/cobra"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/resource"
	k8sclient "k8s.io/kubernetes/pkg/client/unversioned"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

func NewCmdVolumes(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "volumes",
		Short: "Creates a persisent volume for any pending persistance volume claims",
		Long:  `Creates a persisent volume for any pending persistance volume claims`,
		PreRun: func(cmd *cobra.Command, args []string) {
			showBanner()
		},
		Run: func(cmd *cobra.Command, args []string) {
			c, _ := client.NewClient(f)
			ns, _, err := f.DefaultNamespace()
			if err != nil {
				util.Fatal("No default namespace")
			} else {

				found, pendingClaimNames := findPendingPVS(c, ns)
				if found {
					createPV(c, ns, pendingClaimNames)
				}
			}
		},
	}
	//cmd.PersistentFlags().StringP(hostPathFlag, "", "", "Defines the host folder on which to define a persisent volume for single node setups")
	return cmd
}

func findPendingPVS(c *k8sclient.Client, ns string) (bool, []string) {

	pvcs, err := c.PersistentVolumeClaims(ns).List(api.ListOptions{})
	if err != nil {
		util.Infof("Failed to find any PersistentVolumeClaims, %s in namespace %s\n", err, ns)
	}

	if pvcs != nil {
		items := pvcs.Items
		pendingClaimNames := make([]string, 0, len(items))
		for _, item := range items {
			status := item.Status.Phase
			if status == "Pending" {
				pendingClaimNames = append(pendingClaimNames, item.ObjectMeta.Name)
			}
		}
		if len(pendingClaimNames) > 0 {
			return true, pendingClaimNames
		}
	}
	return false, nil
}

func createPV(c *k8sclient.Client, ns string, pvcNames []string) (Result, error) {

	for _, pvcName := range pvcNames {
		hostPath := "/" + pvcName
		pvs := c.PersistentVolumes()
		rc, err := pvs.List(api.ListOptions{})
		if err != nil {
			util.Errorf("Failed to load PersistentVolumes with error %v", err)
		}
		items := rc.Items
		for _, volume := range items {
			vname := volume.ObjectMeta.Name
			if vname == pvcName {
				util.Infof("Already created PersistentVolumes for %s\n", pvcName)
			}
		}

		// lets create a new PV
		util.Infof("PersistentVolume name %s will be created on host path %s\n", pvcName, hostPath)
		pv := api.PersistentVolume{
			ObjectMeta: api.ObjectMeta{
				Name: pvcName,
			},
			Spec: api.PersistentVolumeSpec{
				Capacity: api.ResourceList{
					api.ResourceName(api.ResourceStorage): resource.MustParse("1G"),
				},
				AccessModes: []api.PersistentVolumeAccessMode{api.ReadWriteMany},
				PersistentVolumeSource: api.PersistentVolumeSource{
					HostPath: &api.HostPathVolumeSource{Path: hostPath},
				},
			},
		}

		_, err = pvs.Create(&pv)
		if err != nil {
			util.Errorf("Failed to create PersistentVolume %s at %s with error %v", pvcName, hostPath, err)
		}
	}

	return Success, nil
}
