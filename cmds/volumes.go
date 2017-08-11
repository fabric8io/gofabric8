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
	"path"
	"syscall"

	"github.com/fabric8io/gofabric8/client"
	"github.com/fabric8io/gofabric8/util"
	"github.com/spf13/cobra"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/resource"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

const (
	sshCommandFlag = "ssh-command"
)

func NewCmdVolumes(f cmdutil.Factory) *cobra.Command {
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

				found, pvcs, pendingClaimNames := findPendingPVs(c, ns)
				if found {

					sshCommand := cmd.Flags().Lookup(sshCommandFlag).Value.String()

					createPV(c, ns, pendingClaimNames, sshCommand)
					items := pvcs.Items
					for _, item := range items {
						name := item.ObjectMeta.Name
						status := item.Status.Phase
						if status == api.ClaimPending || status == api.ClaimLost {
							err = c.PersistentVolumeClaims(ns).Delete(name, nil)
							if err != nil {
								util.Infof("Error deleting PVC %s\n", name)
							} else {
								util.Infof("Recreating PVC %s\n", name)
								c.PersistentVolumeClaims(ns).Create(&api.PersistentVolumeClaim{
									ObjectMeta: api.ObjectMeta{
										Name:      name,
										Namespace: ns,
									},
									Spec: api.PersistentVolumeClaimSpec{
										VolumeName:  ns + "-" + name,
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
		},
	}
	cmd.PersistentFlags().String(sshCommandFlag, "", "the ssh command to run commands inside the VM of the single node cluster")
	return cmd
}

func findPendingPVs(c *clientset.Clientset, ns string) (bool, *api.PersistentVolumeClaimList, []string) {

	pvcs, err := c.PersistentVolumeClaims(ns).List(api.ListOptions{})

	if err != nil {
		util.Infof("Failed to find any PersistentVolumeClaims, %s in namespace %s\n", err, ns)
	}

	if pvcs != nil {
		pendingClaims := pvcs.Items
		var pendingClaimNames []string
		for _, item := range pendingClaims {
			status := item.Status.Phase
			if status == api.ClaimPending || status == api.ClaimLost {
				pendingClaimNames = append(pendingClaimNames, item.ObjectMeta.Name)
			}
		}
		if len(pendingClaimNames) > 0 {
			return true, pvcs, pendingClaimNames
		}
	}
	return false, nil, nil
}

func createPV(c *clientset.Clientset, ns string, pvcNames []string, sshCommand string) (Result, error) {

	for _, pvcName := range pvcNames {
		hostPath := path.Join("/data", ns, pvcName)
		nsPvcName := ns + "-" + pvcName
		pvs := c.PersistentVolumes()
		rc, err := pvs.List(api.ListOptions{})
		if err != nil {
			util.Errorf("Failed to load PersistentVolumes with error %v\n", err)
		}
		items := rc.Items
		for _, volume := range items {
			if nsPvcName == volume.ObjectMeta.Name {
				util.Infof("Already created PersistentVolumes for %s\n", nsPvcName)
			}
		}

		// we no longer need to do chmod on kubernetes as we have init containers now
		typeOfMaster := util.TypeOfMaster(c)
		if typeOfMaster != util.Kubernetes || len(sshCommand) > 0 {
			err = configureHostPathVolume(c, ns, hostPath, sshCommand)
			if err != nil {
				util.Errorf("Failed to configure the host path %s with error %v\n", hostPath, err)
			}
		}

		// lets create a new PV
		util.Infof("PersistentVolume name %s will be created on host path %s\n", nsPvcName, hostPath)
		pv := api.PersistentVolume{
			ObjectMeta: api.ObjectMeta{
				Name: nsPvcName,
			},
			Spec: api.PersistentVolumeSpec{
				Capacity: api.ResourceList{
					api.ResourceName(api.ResourceStorage): resource.MustParse("1Gi"),
				},
				AccessModes: []api.PersistentVolumeAccessMode{api.ReadWriteOnce},
				PersistentVolumeSource: api.PersistentVolumeSource{
					HostPath: &api.HostPathVolumeSource{Path: hostPath},
				},
				PersistentVolumeReclaimPolicy: api.PersistentVolumeReclaimRecycle,
			},
		}

		_, err = pvs.Create(&pv)
		if err != nil {
			util.Errorf("Failed to create PersistentVolume %s at %s with error %v\n", nsPvcName, hostPath, err)
		}

	}

	return Success, nil
}

// if we are on minikube or minishift lets try to create the
// hostPath folders with relaxed persmissions
func configureHostPathVolume(c *clientset.Clientset, ns string, hostPath string, sshCommand string) error {
	cli := sshCommand

	if len(cli) == 0 {
		context, isMini, _ := util.GetMiniType()
		if isMini {
			cli = context
		}
	}
	if len(cli) == 0 {
		// lets default to using vagrant if we have a Vagrantfile
		if _, err := os.Stat("Vagrantfile"); err == nil {
			cli = "vagrant"
		}
	}
	if len(cli) == 0 {
		return nil
	}

	shellCommands := []string{
		fmt.Sprintf("sudo mkdir -p %s", hostPath),
		fmt.Sprintf("sudo chmod 777 %s", hostPath),
		fmt.Sprintf("echo hostPath is setup correctly at: %s", hostPath),
	}
	util.Infof("About to modify host paths on the VM via the command: %s\n", cli)

	for _, shellCmd := range shellCommands {
		args := []string{"ssh", fmt.Sprintf("/bin/sh -c '%s'", shellCmd)}
		cmd := exec.Command(cli, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		var waitStatus syscall.WaitStatus
		if err := cmd.Run(); err != nil {
			printErr(err)
			if exitError, ok := err.(*exec.ExitError); ok {
				waitStatus = exitError.Sys().(syscall.WaitStatus)
				printStatus(waitStatus.ExitStatus())
			}
			return err
		} else {
			waitStatus = cmd.ProcessState.Sys().(syscall.WaitStatus)
			printStatus(waitStatus.ExitStatus())
		}
	}
	return nil
}
