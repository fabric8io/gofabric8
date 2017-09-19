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
	"time"

	"github.com/fabric8io/gofabric8/client"
	"github.com/fabric8io/gofabric8/util"
	"github.com/spf13/cobra"
	"k8s.io/kubernetes/pkg/api"
	k8api "k8s.io/kubernetes/pkg/api/unversioned"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"strings"
)

type runTestFlags struct {
	confirm            bool
	retryFabric8Lookup bool
	test               string
	namespace          string
	image              string
	testGitRepo        string
	testGitBranch      string

	targetUrl  string
	platform   string
	disableChe bool

	// tenant config
	booster        string
	boosterGitRef  string
	boosterGitRepo string

	cheVersion     string
	jenkinsVersion string
	teamVersion    string
	mavenRepo      string
}

// NewCmdE2ETest performs an end to end test in the current cluster in a local pod
func NewCmdE2ETest(f cmdutil.Factory) *cobra.Command {
	p := &runTestFlags{}
	cmd := &cobra.Command{
		Use:     "e2e",
		Short:   "Runs the end to end system tests",
		Long:    `Runs the end to end system tests`,
		Aliases: []string{"e2e-tests"},

		Run: func(cmd *cobra.Command, args []string) {
			if cmd.Flags().Lookup(yesFlag).Value.String() == "true" {
				p.confirm = true
			}
			err := p.runTest(f)
			if err != nil {
				util.Fatalf("%s\n", err)
			}
			return
		},
	}
	flags := cmd.Flags()
	flags.StringVarP(&p.test, "test", "", "test", "the name of the test")
	flags.StringVarP(&p.namespace, "namespace", "n", "", "the namespace to look for the fabric8 installation. Defaults to the current namespace")
	flags.StringVarP(&p.image, "image", "", "fabric8/fabric8-ui-builder:0.0.8", "the test image to use")
	flags.StringVarP(&p.targetUrl, "url", "", "", "the target URL of the fabric8-ui to test or it defaults to the fabric8 service in the current namespace")
	flags.StringVarP(&p.platform, "platform", "", "", "the target platform to test against. Defaults to `osio` for a targetUrl of `https://openshift.io/` otherwise `fabric8-openshift`. Use `fabric8-kubernetes` if using kubernetes")
	flags.StringVarP(&p.testGitRepo, "repo", "", "https://github.com/fabric8io/fabric8-test.git", "the test git repository to use")
	flags.StringVarP(&p.testGitBranch, "branch", "", "master", "the test git repository branch, SHA or commit id to use")
	flags.BoolVarP(&p.retryFabric8Lookup, "retry", "", false, "should we wait for the fabric8 service to be ready")
	flags.BoolVarP(&p.disableChe, "disable-che", "", true, "should we disable the Che E2E tests?")

	// tenant config
	flags.StringVarP(&p.booster, "booster", "", "", "the booster name to test")
	flags.StringVarP(&p.boosterGitRef, "booster-git-ref", "", "", "the booster git repository reference (branch, tag, sha)")
	flags.StringVarP(&p.boosterGitRepo, "booster-git-repo", "", "", "the booster git repository URL to use for the tests - when using a fork or custom catalog")

	flags.StringVarP(&p.cheVersion, "che-version", "", "", "the Che YAML version for the tenant")
	flags.StringVarP(&p.jenkinsVersion, "jenkins-version", "", "", "the Jenkins YAML version for the tenant")
	flags.StringVarP(&p.teamVersion, "team-version", "", "", "the Team YAML version for the tenant")
	flags.StringVarP(&p.mavenRepo, "maven-repo", "", "", "the maven repository used for tenant YAML if using a PR or custom build")
	return cmd
}

func (p *runTestFlags) runTest(f cmdutil.Factory) error {
	c, _ := client.NewClient(f)
	initSchema()

	typeOfMaster := util.TypeOfMaster(c)

	ns := p.namespace
	if len(ns) == 0 {
		ns, _, _ = f.DefaultNamespace()
	}
	if len(ns) == 0 {
		return fmt.Errorf("No namespace is defined and no namespace specified!")
	}
	url := p.targetUrl
	if len(url) == 0 {
		url = FindServiceURL(ns, "fabric8", c, p.retryFabric8Lookup)
	}
	if len(url) == 0 {
		return fmt.Errorf("No --url specified and no fabric8 service found in namespace %s!", ns)
	}

	util.Infof("testing the fabric8 installation at URL: %s\n", url)

	selector, err := k8api.LabelSelectorAsSelector(
		&k8api.LabelSelector{MatchLabels: map[string]string{"test": "e2e"}})
	if err != nil {
		return err
	}

	secrets, err := c.Secrets(ns).List(api.ListOptions{LabelSelector: selector})
	if err != nil {
		return fmt.Errorf("Failed to load secrets in namespace %s due to %s", ns, err)
	}
	script := "git clone --branch " + p.testGitBranch + " " + p.testGitRepo + ` /tmp/fabric8-test
cd /tmp/fabric8-test
source /opt/env/script
./pod_EE_tests.sh
`

	completeStatus := ""

	cheFlag := "false"
	if p.disableChe {
		cheFlag = "true"
	}
	platform := p.platform
	if len(platform) == 0 {
		if strings.HasPrefix(url, "https://openshift.io") {
			platform = "osio"
		} else {
			if typeOfMaster == util.Kubernetes {
				platform = "fabric8-kubernetes"
			} else {
				platform = "fabric8-openshift"
			}
		}
	}

	for _, secret := range secrets.Items {
		name := p.test + "-" + secret.Name

		// lets delete the pod if it exists
		c.Pods(ns).Delete(name, &api.DeleteOptions{})
		waitForPodToBeDeleted(c, ns, name)

		pod := api.Pod{
			ObjectMeta: api.ObjectMeta{
				Name: name,
				Labels: map[string]string{
					"provider": "fabric8",
					"app":      "e2e",
					"secret":   secret.Name,
				},
			},
			Spec: api.PodSpec{
				Containers: []api.Container{
					{
						Name:  "test",
						Image: p.image,
						Env: []api.EnvVar{
							{
								Name:  "TARGET_URL",
								Value: url,
							},
							{
								Name:  "DISABLE_CHE",
								Value: cheFlag,
							},
							{
								Name:  "TARGET_PLATFORM",
								Value: platform,
							},
							{
								Name:  "QUICKSTART",
								Value: p.booster,
							},
							{
								Name:  "BOOSTER_GIT_REF",
								Value: p.boosterGitRef,
							},
							{
								Name:  "BOOSTER_GIT_REPO",
								Value: p.boosterGitRepo,
							},
							{
								Name:  "TENANT_CHE_VERSION",
								Value: p.cheVersion,
							},
							{
								Name:  "TENANT_JENKINS_VERSION",
								Value: p.jenkinsVersion,
							},
							{
								Name:  "TENANT_TEAM_VERSION",
								Value: p.teamVersion,
							},
							{
								Name:  "TENANT_MAVEN_REPO",
								Value: p.mavenRepo,
							},
						},
						Command: []string{
							"/bin/bash",
						},
						Args: []string{
							"-c",
							script,
						},
						VolumeMounts: []api.VolumeMount{
							{
								Name:      "env",
								ReadOnly:  true,
								MountPath: "/opt/env",
							},
						},
					},
				},
				Volumes: []api.Volume{
					{
						Name: "env",
						VolumeSource: api.VolumeSource{
							Secret: &api.SecretVolumeSource{
								SecretName: secret.Name,
								Items: []api.KeyToPath{
									{
										Key:  "script",
										Path: "script",
									},
								},
							},
						},
					},
				},
				RestartPolicy: api.RestartPolicyNever,
			},
		}
		_, err := c.Pods(ns).Create(&pod)
		if err != nil {
			util.Warnf("Failed to create pod %s due to %s", name, err)
		} else {

			_, err := waitForPodToBeRunningOrFinished(c, ns, name)
			if err != nil {
				return err
			}

			// wait for the pod to die and tail its logs...
			err = runCommand("kubectl", "logs", "-n", ns, "-f", name)
			if err != nil {
				util.Warnf("Failed to tail pod %s in namespace %s due to %s\n", name, ns, err)
			}

			status, err := waitForPodToFinish(c, ns, name)
			if err != nil {
				return err
			}

			util.Infof("Pod %s completed with status %s", name, status)
			if len(status) > 0 {
				completeStatus = status
			}
			err = c.Pods(ns).Delete(name, &api.DeleteOptions{})
			if err != nil {
				util.Warnf("Failed to delete pod %s in namespace %s due to %s\n", name, ns, err)
			}
		}
	}
	if len(secrets.Items) == 0 {
		return fmt.Errorf("No Secrets found in namespace %s which have the label: test=e2e\nPlease use the `gofabric8 e2e-secret` command to install a secret for a test user and password\n", ns)
	}
	if len(completeStatus) > 0 {
		return fmt.Errorf("FAIL test due to: %s", completeStatus)
	}
	util.Infof("OK test completed!\n")
	return nil
}

func waitForPodToBeDeleted(c *clientset.Clientset, ns string, podName string) error {
	logged := false
	for {
		pods, err := c.Pods(ns).List(api.ListOptions{})
		if err == nil {
			found := false
			for _, pod := range pods.Items {
				if pod.Name == podName {
					found = true
				}
			}
			if !found {
				return nil
			}
		}

		if !logged {
			logged = true
			util.Infof("Waiting for pod %s to be deleted\n", podName)
		}
		// TODO replace with a watch flavour
		time.Sleep(time.Second)
	}
}

func waitForPodToBeRunningOrFinished(c *clientset.Clientset, ns string, podName string) (api.PodPhase, error) {
	util.Infof("Waiting for pod %s in namespace %s to be running\n", podName, ns)
	currentPhase := api.PodUnknown
	for {
		pods, err := c.Pods(ns).List(api.ListOptions{})
		if err == nil {
			found := false
			for _, pod := range pods.Items {
				if pod.Name == podName {
					found = true
					phase := pod.Status.Phase
					if phase != currentPhase {
						currentPhase = phase
						util.Infof("Pod %s has phase %s\n", podName, phase)
					}
					if phase == api.PodSucceeded || phase == api.PodFailed || phase == api.PodRunning || phase == api.PodUnknown {
						return phase, nil
					}
				}
			}
			if !found {
				return api.PodUnknown, nil
			}
		}

		// TODO replace with a watch flavour
		time.Sleep(time.Second)
	}
}

func waitForPodToFinish(c *clientset.Clientset, ns string, podName string) (string, error) {
	util.Infof("Waiting for pod %s in namespace %s to complete\n", podName, ns)
	for {
		pods, err := c.Pods(ns).List(api.ListOptions{})
		if err == nil {
			found := false
			for _, pod := range pods.Items {
				if pod.Name == podName {
					found = true
					phase := pod.Status.Phase
					if phase == api.PodSucceeded {
						return "", nil
					} else if phase == api.PodFailed || phase == api.PodUnknown {
						return fmt.Sprintf("%s", phase), nil
					}
				}
			}
			if !found {
				return "Pod not found", nil
			}
		}

		// TODO replace with a watch flavour
		time.Sleep(time.Second)
	}
}
