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
	"syscall"

	"github.com/fabric8io/gofabric8/client"
	"github.com/fabric8io/gofabric8/util"
	oclient "github.com/openshift/origin/pkg/client"
	tapi "github.com/openshift/origin/pkg/template/api"
	tapiv1 "github.com/openshift/origin/pkg/template/api/v1"
	"github.com/spf13/cobra"
	"k8s.io/kubernetes/pkg/api"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/runtime"
)

func NewCmdPull(f cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull [templateNames]",
		Short: "Pulls the docker images for the given templates",
		Long:  `Performs a docker pull on all the docker images referenced in the given templates to preload the local docker registry with images`,
		PreRun: func(cmd *cobra.Command, args []string) {
			tapi.AddToScheme(api.Scheme)
			tapiv1.AddToScheme(api.Scheme)
			showBanner()
		},
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 1 {
				util.Error("No template names specified!")
				cmd.Usage()
			} else {
				_, cfg := client.NewClient(f)
				oc, _ := client.NewOpenShiftClient(cfg)
				ns, _, err := f.DefaultNamespace()
				if err != nil {
					util.Fatal("No default namespace")
				}
				for _, template := range args {
					util.Info("Downloading docker images for template ")
					util.Success(template)
					util.Info("\n\n")

					r, err := downloadTemplateDockerImages(ns, oc, f, template)
					printResult("Download Docker images", r, err)
				}
			}
		},
	}
	return cmd
}

func downloadTemplateDockerImages(ns string, c *oclient.Client, fac cmdutil.Factory, name string) (Result, error) {
	template, err := c.Templates(ns).Get(name)
	if err != nil {
		util.Fatalf("No Template %s found in namespace %s\n", name, ns)
	}

	// convert Template.Objects to Kubernetes resources
	_ = runtime.DecodeList(template.Objects, api.Codecs.UniversalDecoder())
	for _, rc := range template.Objects {
		switch rc := rc.(type) {
		case *api.ReplicationController:
			for _, container := range rc.Spec.Template.Spec.Containers {
				err = downloadDockerImage(container.Image)
				if err != nil {
					return Failure, err
				}
			}
		}
	}
	return Success, nil
}

func downloadDockerImage(imageName string) error {
	util.Info("Downloading image ")
	util.Success(imageName)
	util.Info("\n")

	cmd := exec.Command("docker", "pull", imageName)
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
		return nil
	}
}

func printStatus(exitStatus int) {
	if exitStatus != 0 {
		util.Error(fmt.Sprintf("%d", exitStatus))
	}
}

func printErr(err error) {
	if err != nil {
		util.Errorf("%s\n", err.Error())
	}
}
