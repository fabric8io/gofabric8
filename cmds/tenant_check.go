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

	"github.com/fabric8io/gofabric8/client"
	"github.com/fabric8io/gofabric8/util"
	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"strings"
)

type tenantCheckFlags struct {
	cmd  *cobra.Command
	args []string

	tenant string
}

func NewCmdTenantCheck(f cmdutil.Factory) *cobra.Command {
	p := &tenantCheckFlags{}
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Checks your tenant is working correctly",
		Run: func(cmd *cobra.Command, args []string) {
			p.cmd = cmd
			p.args = args
			handleError(p.tenantCheck(f))
		},
	}
	flags := cmd.Flags()
	flags.StringVarP(&p.tenant, "tenant", "t", "", "the name of the tenant to check. If blank it will be discovered")
	return cmd
}

func (p *tenantCheckFlags) tenantCheck(f cmdutil.Factory) error {
	c, cfg := client.NewClient(f)
	ns, _, _ := f.DefaultNamespace()
	oc, _ := client.NewOpenShiftClient(cfg)
	initSchema()

	var err error
	userNS := p.tenant
	if len(userNS) == 0 {
		userNS, err = detectCurrentUserNamespace(ns, c, oc)
		if err != nil {
			return err
		}
	}
	util.Info("Checking tenant: ")
	util.Successf("%s\n\n", userNS)

	jenkinsNS := fmt.Sprintf("%s-jenkins", userNS)

	// check jenkins pod is ready and with correct version
	err = ensureDeploymentOrDCHasReplicas(c, oc, jenkinsNS, "jenkins", 1)
	if err != nil {
		return err
	}

	podName, err := waitForReadyPodForDeploymentOrDC(c, oc, jenkinsNS, "jenkins")
	if err != nil {
		return err
	}
	dc, err := oc.DeploymentConfigs(jenkinsNS).Get("jenkins")
	if err != nil {
		return err
	}
	pod, err := c.Pods(jenkinsNS).Get(podName)
	if err != nil {
		return err
	}
	if dc.Labels == nil {
		return fmt.Errorf("jenkins DeploymentConfig has no labels!")
	}
	if pod.Labels == nil {
		return fmt.Errorf("jenkins Pod %s has no labels!", pod)
	}

	dcVersion := dc.Labels["version"]
	podVersion := pod.Labels["version"]

	if dcVersion != podVersion {
		return fmt.Errorf("Invalid Jenkins pod %s in namespace %s has version %s when should be %s", podName, jenkinsNS, podVersion, dcVersion)
	}
	err = ensureDeploymentOrDCHasReplicas(c, oc, jenkinsNS, "content-repository", 1)
	if err != nil {
		return err
	}
	contentRepoPodName, err := waitForReadyPodForDeploymentOrDC(c, oc, jenkinsNS, "content-repository")
	if err != nil {
		return err
	}

	// check user namespace exists!
	_, err = c.ConfigMaps(userNS).Get("fabric8-environments")
	if err != nil {
		return fmt.Errorf("No ConfigMap called fabric8-environments in your user namespace %s. Please Update your Tenant!", userNS)
	}

	roleBindingSelector := "app=fabric8-online-team"
	rolebindingText, err := runCommandWithOutput("oc", "get", "rolebindings", "-l", roleBindingSelector, "-n", userNS)
	if err != nil {
		return err
	}
	lines := strings.Split(rolebindingText, "\n")
	foundJenkinsRoleBinding := false
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "jenkins-") {
			foundJenkinsRoleBinding = true
		}
	}
	if !foundJenkinsRoleBinding {
		return fmt.Errorf("Failed to find a jenkins RoleBinding in namespace %s with selector %s due to: %s", userNS, roleBindingSelector, rolebindingText)
	}

	util.Infof("\nUser project %s has the ConfigMaps and RoleBindings\n", userNS)
	util.Infof("Jenkins pod %s is running correctly in namespace %s with version %s\n", podName, jenkinsNS, podVersion)

	util.Infof("\nDisk usage in jenkins pod %s in namespace %s\n", podName, jenkinsNS)
	err = runCommand("oc", "exec", "-t", podName, "df", "/var/lib/jenkins")
	if err != nil {
		return err
	}
	util.Infof("\nDisk usage in content repository pod %s in namespace %s\n", contentRepoPodName, jenkinsNS)
	err = runCommand("oc", "exec", "-t", contentRepoPodName, "df", "/var/www/html")
	if err != nil {
		return err
	}
	return nil
}
