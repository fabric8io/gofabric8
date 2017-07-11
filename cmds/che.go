/*
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
	"runtime"

	"github.com/fabric8io/gofabric8/client"
	"github.com/fabric8io/gofabric8/util"

	"github.com/spf13/cobra"

	"k8s.io/kubernetes/pkg/api"

	k8sclient "k8s.io/kubernetes/pkg/client/unversioned"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

type cmdCheShell struct {
	cmd  *cobra.Command
	args []string

	namespace string
	shell     string
	pod       string
}

func NewCmdCheShell(f *cmdutil.Factory) *cobra.Command {
	p := &cmdCheShell{}
	cmd := &cobra.Command{
		Use:   "che",
		Short: "Opens a shell in a Che workspace pod",
		Run: func(cmd *cobra.Command, args []string) {
			p.cmd = cmd
			p.args = args
			handleError(p.run(f))

		},
	}
	flags := cmd.Flags()
	flags.StringVarP(&p.namespace, "namespace", "n", "", "the Che workspace to look inside")
	flags.StringVarP(&p.shell, "shell", "", "bash", "the shell to use")
	flags.StringVarP(&p.pod, "pod", "p", "", "the pod name to use")
	return cmd
}

func (p *cmdCheShell) run(f *cmdutil.Factory) error {
	c, cfg := client.NewClient(f)

	initSchema()

	typeOfMaster := util.TypeOfMaster(c)
	isOpenshift := typeOfMaster == util.OpenShift

	cheNS := p.namespace
	if cheNS == "" {
		if isOpenshift {
			oc, _ := client.NewOpenShiftClient(cfg)
			projects, err := oc.Projects().List(api.ListOptions{})
			if err != nil {
				util.Warnf("Could not list projects: %v", err)
			} else {
				currentNS, _, _ := f.DefaultNamespace()
				cheNS = detectCurrentUserProject(currentNS, projects.Items, c)
				if cheNS != "" {
					cheNS += "-che"
				}

			}
		}
	}

	if cheNS == "" {
		cheNS, _, _ = f.DefaultNamespace()
	}

	pods, err := c.Pods(cheNS).List(api.ListOptions{})
	if err != nil {
		return fmt.Errorf("Could not list Che workspace pods in namespace %s due to %v", cheNS, err)
	}
	for _, pod := range pods.Items {
		labels := pod.Labels
		if labels != nil {
			if len(labels["cheContainerIdentifier"]) > 0 {
				return p.openShell(c, cheNS, pod.Name, isOpenshift)
			}
		}
	}
	return fmt.Errorf("No Che workspace pods found in namespace %s", cheNS)
}

func (p *cmdCheShell) openShell(c *k8sclient.Client, namespace string, podName string, isOpenshift bool) error {
	shell := p.shell
	util.Infof("starting %s shell in Che workspace pod %s in namespace %s\n", shell, podName, namespace)
	kubeBinary := "kubectl"
	if isOpenshift {
		kubeBinary = "oc"
	}
	if runtime.GOOS == "windows" {
		kubeBinary += ".exe"
	}

	binaryFile := resolveBinaryLocation(kubeBinary)

	e := exec.Command(binaryFile, "-it", "-n", namespace, "exec", podName, shell)
	e.Stdin = os.Stdin
	e.Stdout = os.Stdout
	e.Stderr = os.Stderr
	err := e.Run()
	if err != nil {
		fmt.Printf("Unable to start shell %s in pod %s on namespace %s due to: %v", shell, podName, namespace, err)
	}
	return nil
}
