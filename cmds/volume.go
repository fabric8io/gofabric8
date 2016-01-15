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
	k8sclient "k8s.io/kubernetes/pkg/client"
	"k8s.io/kubernetes/pkg/fields"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/labels"
)

func NewCmdVolume(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "volume",
		Short: "Creates a persisent volume for fabric8 apps needing persistent disk",
		Long:  `Creates a persisent volume so that the PersistentVolumeClaims in fabric8 apps can be satisfied when creating fabric8 apps`,
		PreRun: func(cmd *cobra.Command, args []string) {
			showBanner()
		},
		Run: func(cmd *cobra.Command, args []string) {
			c, cfg := client.NewClient(f)
			ns, _, err := f.DefaultNamespace()
			if err != nil {
				util.Fatal("No default namespace")
			} else {
				util.Info("Creating a persistent volume for your ")
				util.Success(string(util.TypeOfMaster(c)))
				util.Info(" installation at ")
				util.Success(cfg.Host)
				util.Info(" in namespace ")
				util.Successf("%s\n\n", ns)

				r, err := createPersistentVolume(cmd, ns, c, f)
				printResult("Create PersistentVolume", r, err)
			}
		},
	}
	cmd.PersistentFlags().StringP(hostPathFlag, "", "", "Defines the host folder on which to define a persisent volume for single node setups")
	cmd.PersistentFlags().StringP(nameFlag, "", "fabric8", "The name of the PersistentVolume to create")
	return cmd
}

func createPersistentVolume(cmd *cobra.Command, ns string, c *k8sclient.Client, fac *cmdutil.Factory) (Result, error) {
	flags := cmd.Flags()
	hostPath := flags.Lookup(hostPathFlag).Value.String()
	name := flags.Lookup(nameFlag).Value.String()
	pvs := c.PersistentVolumes()
	rc, err := pvs.List(labels.Everything(), fields.Everything())
	if err != nil {
		util.Errorf("Failed to load PersistentVolumes with error %v", err)
		return Failure, err
	}
	items := rc.Items
	for _, volume := range items {
		// TODO use the external load balancer as a way to know if we should create a route?
		vname := volume.ObjectMeta.Name
		if vname == name {
			util.Infof("Already created PersistentVolumes for %s\n", name)
			return Success, nil
		}
	}
	if hostPath == "" {
		return missingFlag(cmd, hostPathFlag)
	}
	if confirmAction(flags) == false {
		return Failure, nil
	}

	// lets create a new PV
	util.Infof("PersistentVolume name %s will be created on host path %s\n", name, hostPath)
	pv := api.PersistentVolume{
		ObjectMeta: api.ObjectMeta{
			Name: name,
		},
		Spec: api.PersistentVolumeSpec{
			Capacity: api.ResourceList{
				api.ResourceName(api.ResourceStorage): resource.MustParse("100G"),
			},
			AccessModes: []api.PersistentVolumeAccessMode{api.ReadWriteMany},
			PersistentVolumeSource: api.PersistentVolumeSource{
				HostPath: &api.HostPathVolumeSource{Path: hostPath},
			},
		},
	}

	_, err = pvs.Create(&pv)
	if err != nil {
		util.Errorf("Failed to create PersistentVolume %s at %s with error %v", name, hostPath, err)
		return Failure, err
	}
	return Success, nil
}
