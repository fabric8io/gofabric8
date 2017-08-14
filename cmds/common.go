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
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/daviddengcn/go-colortext"
	"github.com/fabric8io/gofabric8/util"
	"github.com/spf13/cobra"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/watch"

	buildapi "github.com/openshift/origin/pkg/build/api"
	oclient "github.com/openshift/origin/pkg/client"
	osapi "github.com/openshift/origin/pkg/project/api"
	k8api "k8s.io/kubernetes/pkg/api/unversioned"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

type Result string

const (
	Success Result = "✔"
	Failure Result = "✘"

	// cmd flags
	yesFlag       = "yes"
	hostPathFlag  = "host-path"
	nameFlag      = "name"
	domainFlag    = "domain"
	apiServerFlag = "api-server"
	consoleFlag   = "console"
	templatesFlag = "templates"
	DefaultDomain = ""
)

func defaultNamespace(cmd *cobra.Command, f cmdutil.Factory) (string, error) {
	ns := cmd.Flags().Lookup(namespaceCommandFlag).Value.String()
	if len(ns) > 0 {
		return ns, nil
	}
	nsFile := cmd.Flags().Lookup(namespaceFileFlag).Value.String()
	if len(nsFile) > 0 {
		util.Infof("Loading namespace file %s\n", nsFile)
		if fileNotExist(nsFile) {
			return ns, fmt.Errorf("Could not find file `%s` to resolve the namespace!", nsFile)
		}
		data, err := ioutil.ReadFile(nsFile)
		if err != nil {
			return ns, fmt.Errorf("Failed to read namespace from file `%s` due to: %v", nsFile, err)
		}
		ns = string(data)
		if len(ns) == 0 {
			return ns, fmt.Errorf("The file `%s` is empty so cannot set the namespace!", nsFile)
		}
		return ns, nil
	}
	ns = os.Getenv("KUBERNETES_NAMESPACE")
	if len(ns) > 0 {
		return ns, nil
	}
	ns, _, err := f.DefaultNamespace()
	return ns, err
}

// ensureDeploymentOrDCHasReplicas ensures that the given Deployment or DeploymentConfig has at least the right number
// of replicas
func ensureDeploymentOrDCHasReplicas(c *clientset.Clientset, oc *oclient.Client, ns string, name string, minRelicas int32) error {
	typeOfMaster := util.TypeOfMaster(c)
	if typeOfMaster == util.OpenShift {
		dc, err := oc.DeploymentConfigs(ns).Get(name)
		if err == nil && dc != nil {
			if dc.Spec.Replicas >= minRelicas {
				return nil
			}
			dc.Spec.Replicas = minRelicas
			util.Infof("Scaling DeploymentConfig %s in namespace %s to %d\n", name, ns, minRelicas)
			_, err = oc.DeploymentConfigs(ns).Update(dc)
			return err
		}
	}
	deployment, err := c.Extensions().Deployments(ns).Get(name)
	if err != nil || deployment == nil {
		return fmt.Errorf("Could not find a Deployment or DeploymentConfig called %s in namespace %s due to %v", name, ns, err)
	}
	if deployment.Spec.Replicas >= minRelicas {
		return nil
	}
	deployment.Spec.Replicas = minRelicas
	util.Infof("Scaling Deployment %s in namespace %s to %d\n", name, ns, minRelicas)
	_, err = c.Extensions().Deployments(ns).Update(deployment)
	return err
}

// waitForReadyPodForDeploymentOrDC waits for a ready pod in a Deployment or DeploymentConfig
// in the given namespace with the given name
func waitForReadyPodForDeploymentOrDC(c *clientset.Clientset, oc *oclient.Client, ns string, name string) (string, error) {
	typeOfMaster := util.TypeOfMaster(c)
	if typeOfMaster == util.OpenShift {
		dc, err := oc.DeploymentConfigs(ns).Get(name)
		if err == nil && dc != nil {
			selector := dc.Spec.Selector
			if selector == nil {
				return "", fmt.Errorf("No selector defined on Deployment %s in namespace %s", name, ns)
			}
			return waitForReadyPodForSelector(c, oc, ns, selector)
		}
	}
	deployment, err := c.Extensions().Deployments(ns).Get(name)
	if err != nil || deployment == nil {
		return "", fmt.Errorf("Could not find a Deployment or DeploymentConfig called %s in namespace %s due to %v", name, ns, err)
	}
	selector := deployment.Spec.Selector
	if selector == nil {
		return "", fmt.Errorf("No selector defined on Deployment %s in namespace %s", name, ns)
	}
	labels := selector.MatchLabels
	if labels == nil {
		return "", fmt.Errorf("No MatchLabels defined on the Selector of Deployment %s in namespace %s", name, ns)
	}
	return waitForReadyPodForSelector(c, oc, ns, labels)
}

// waitForBCDeleted waits for the given BC to be deleted
func waitForBCDeleted(c *oclient.Client, ns string, name string) {
	first := true
	for {
		_, err := c.BuildConfigs(ns).Get(name)
		time.Sleep(time.Second * 2)
		if err != nil {
			//util.Infof("can't get %s/%s due to %s\n", ns, name, err)
			return
		}
		if first {
			first = false
			util.Infof("Waiting for BuildConfig %s to be deleted\n", name)
		}
	}
}

func waitForReadyPodForSelector(c *clientset.Clientset, oc *oclient.Client, ns string, labels map[string]string) (string, error) {
	selector, err := unversioned.LabelSelectorAsSelector(&unversioned.LabelSelector{MatchLabels: labels})
	if err != nil {
		return "", err
	}
	util.Infof("Waiting for a running pod in namespace %s with labels %v\n", ns, labels)
	for {
		pods, err := c.Pods(ns).List(api.ListOptions{
			LabelSelector: selector,
		})
		if err != nil {
			return "", err
		}
		name := ""
		lastTime := time.Time{}
		for _, pod := range pods.Items {
			phase := pod.Status.Phase
			if phase == api.PodRunning {
				created := pod.CreationTimestamp
				if name == "" || created.After(lastTime) {
					lastTime = created.Time
					name = pod.Name
				}
			}
		}
		if name != "" {
			util.Info("Found newest pod: ")
			util.Successf("%s\n", name)
			return name, nil
		}

		// TODO replace with a watch flavour
		time.Sleep(time.Second)
	}
}

// watchAndWaitForBuild waits for the given build to complete
func watchAndWaitForBuild(c *oclient.Client, ns string, name string, timeout time.Duration) error {
	_, err := c.Builds(ns).Get(name)
	if err != nil {
		return fmt.Errorf("Failed to find Build %s/%s due to %s", ns, name, err)
	}
	// TODO we may wanna add a field selector on the name
	lastPhase := buildapi.BuildPhaseNew
	w, err := c.Builds(ns).Watch(api.ListOptions{})
	if err != nil {
		return err
	}
	_, err = watch.Until(timeout, w, func(e watch.Event) (bool, error) {
		if e.Type == watch.Error {
			return false, fmt.Errorf("encountered error while watching Builds: %v", e.Object)
		}
		obj, ok := e.Object.(*buildapi.Build)
		if !ok {
			return false, fmt.Errorf("received unknown object while watching for Builds: %v", obj)
		}
		build := obj
		if build.Name == name {
			phase := build.Status.Phase
			if phase != lastPhase {
				util.Infof("Build %s is %s\n", name, phase)
				lastPhase = phase
			}
			if phase == buildapi.BuildPhaseComplete {
				return true, nil
			}
			if phase == buildapi.BuildPhaseFailed || phase == buildapi.BuildPhaseError || phase == buildapi.BuildPhaseCancelled {
				return false, fmt.Errorf("Build %s has %s", name, phase)
			}
		}
		return false, nil
	})
	if err != nil {
		return err
	}
	return nil
}

func detectCurrentUserNamespace(ns string, c *clientset.Clientset, oc *oclient.Client) (string, error) {
	typeOfMaster := util.TypeOfMaster(c)
	if typeOfMaster == util.OpenShift {
		projects, err := oc.Projects().List(api.ListOptions{})
		if err != nil {
			return "", err
		}
		return detectCurrentUserProject(ns, projects.Items, c), nil
	} else {
		namespaces, err := c.Namespaces().List(api.ListOptions{})
		if err != nil {
			return "", err
		}
		return detectCurrentUserNamespaceFromNamespaces(ns, namespaces.Items, c), nil
	}
}

// detectCurrentUserProject finds the user namespace name from the given current projects
func detectCurrentUserNamespaceFromNamespaces(current string, items []api.Namespace, c *clientset.Clientset) (chosenone string) {
	names := []string{}
	for _, p := range items {
		names = append(names, p.Name)
	}
	return detectCurrentUserNamespaceFromNames(current, names, c)
}

func detectCurrentUserProject(current string, items []osapi.Project, c *clientset.Clientset) (chosenone string) {
	names := []string{}
	for _, p := range items {
		names = append(names, p.Name)
	}
	return detectCurrentUserNamespaceFromNames(current, names, c)
}

func detectCurrentUserNamespaceFromNames(current string, items []string, c *clientset.Clientset) (chosenone string) {
	var detected []string
	var prefixes = []string{"che", "jenkins", "run", "stage"}

	for _, name := range items {
		// NB(chmou): if we find a che suffix then store it, we are using the
		// project prefixes as create from init-tenant. this probably need to be
		// updated to be future proof.
		for _, k := range prefixes {
			if strings.HasSuffix(name, "-"+k) {
				detected = append(detected, strings.TrimSuffix(name, "-"+k))
			}
		}
	}

	if len(detected) == 1 {
		chosenone = detected[0]
	}

	if len(detected) > 1 {
		for _, p := range detected {

			if current == p {
				chosenone = current
				break
			}

			for _, k := range prefixes {
				if stripped := strings.TrimSuffix(current, "-"+k); stripped == p {
					chosenone = stripped
					break
				}
			}
		}
		if chosenone == "" {
			chosenone = detected[0]
		}
	}

	selector, err := k8api.LabelSelectorAsSelector(
		&k8api.LabelSelector{MatchLabels: map[string]string{"kind": "environments"}})
	cmdutil.CheckErr(err)

	// Make sure after all it exists
	for _, name := range items {
		if name == chosenone {
			cfgmap, err := c.ConfigMaps(name).List(api.ListOptions{LabelSelector: selector})
			cmdutil.CheckErr(err)
			if len(cfgmap.Items) == 0 {
				//TODO: add command line switch to specify the environment if we can't detect it.
				util.Fatalf("Could not autodetect your environment, there is no configmaps environment in the `%s` namespace.\n", name)
			}
			return
		}
	}

	util.Errorf("Cannot find parent namespace for: %s\n", current)
	return ""
}

// runCommand runs the given command on the command line and returns an error if it fails
func runCommand(prog string, args ...string) error {
	cmd := exec.Command(prog, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		text := prog + " " + strings.Join(args, " ")
		return fmt.Errorf("Failed to run command %s due to error %v", text, err)
	}
	return nil
}

// runCommandWithOutput runs the given command on the command line and returns the output as a string or an error if it fails
func runCommandWithOutput(prog string, args ...string) (string, error) {
	cmd := exec.Command(prog, args...)
	var outb, errb bytes.Buffer
	cmd.Stdin = os.Stdin
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		text := prog + " " + strings.Join(args, " ")
		return "", fmt.Errorf("Failed to run command %s due to error %v", text, err)
	}
	answer := outb.String()
	if len(answer) == 0 {
		answer = errb.String()
	}
	return answer, nil
}

func defaultDomain() string {
	defaultDomain := os.Getenv("KUBERNETES_DOMAIN")
	if defaultDomain == "" {
		defaultDomain = DefaultDomain
	}
	return defaultDomain
}

func missingFlag(cmd *cobra.Command, name string) (Result, error) {
	util.Errorf("No option -%s specified!\n", hostPathFlag)
	text := cmd.Name()
	parent := cmd.Parent()
	if parent != nil {
		text = parent.Name() + " " + text
	}
	util.Infof("Please try something like: %s --%s='some value' ...\n\n", text, hostPathFlag)
	return Failure, nil
}

func confirmAction(yes bool) bool {
	if yes {
		util.Info("Continue? [Y/n] ")
		cont := util.AskForConfirmation(true)
		if !cont {
			util.Fatal("Cancelled...\n")
			return false
		}
	}
	return true
}

func isVersion3Package(appName string) bool {
	return appName == platformPackage || appName == consolePackage || appName == iPaaSPackage
}
func showBanner() {
	if runtime.GOOS == "windows" {
		return
	}
	ct.ChangeColor(ct.Blue, false, ct.None, false)
	fmt.Println(fabric8AsciiArt)
	ct.ResetColor()
}

func defaultParameters(c *clientset.Clientset, exposer string, githubClientID string, githubClientSecret string, ns string, appName string) map[string]string {
	typeOfMaster := util.TypeOfMaster(c)
	if len(exposer) == 0 {
		if typeOfMaster == util.Kubernetes {
			exposer = "Ingress"
		} else {
			exposer = "Route"
		}
	}
	if isVersion3Package(appName) {
		return map[string]string{
			"NAMESPACE": ns,
			"EXPOSER":   exposer,
		}
	}
	if len(githubClientID) == 0 {
		githubClientID = os.Getenv("GITHUB_OAUTH_CLIENT_ID")
	}
	if len(githubClientSecret) == 0 {
		githubClientSecret = os.Getenv("GITHUB_OAUTH_CLIENT_SECRET")
	}

	if len(githubClientID) == 0 {
		util.Fatalf("No --%s flag was specified or $GITHUB_OAUTH_CLIENT_ID environment variable supplied!\n", githubClientIDFlag)
	}
	if len(githubClientSecret) == 0 {
		util.Fatalf("No --%s flag was specified or $GITHUB_OAUTH_CLIENT_SECRET environment variable supplied!\n", githubClientSecretFlag)
	}

	mini, err := util.IsMini()
	if err != nil {
		util.Failuref("error checking if minikube or minishift %v", err)
	}
	http := "false"
	tlsAcme := "false"
	if mini {
		// default to generating http routes when running locally
		http = "true"
	} else if typeOfMaster == util.Kubernetes {
		// this tells exposecontroller to annotate each ingress rule so that kube-lego generates signed certs
		tlsAcme = "true"
	}
	return map[string]string{
		"NAMESPACE":                  ns,
		"EXPOSER":                    exposer,
		"GITHUB_OAUTH_CLIENT_SECRET": githubClientSecret,
		"GITHUB_OAUTH_CLIENT_ID":     githubClientID,
		"HTTP":     http,
		"TLS_ACME": tlsAcme,
	}
}

func getTLSAcmeEmail(c *clientset.Clientset, tlsAcmeEmail string) map[string]string {
	if len(tlsAcmeEmail) == 0 {
		tlsAcmeEmail = os.Getenv("TLS_ACME_EMAIL")
	}

	if len(tlsAcmeEmail) == 0 {
		util.Fatalf("No --%s flag was specified or $TLS_ACME_EMAIL environment variable supplied!\n", tlsAcmeEmailFlag)
	}

	return map[string]string{
		"TLS_ACME_EMAIL": tlsAcmeEmail,
	}
}

const fabric8AsciiArt = `             [38;5;25m▄[38;5;25m▄▄[38;5;25m▄[38;5;25m▄[38;5;25m▄[38;5;235m▄[39m         [00m
             [48;5;25;38;5;25m█[48;5;235;38;5;235m█[48;5;235;38;5;235m█[48;5;25;38;5;25m█[48;5;25;38;5;25m█[48;5;25;38;5;25m█[48;5;235;38;5;235m█[49;39m         [00m
     [48;5;233;38;5;235m▄[48;5;235;38;5;25m▄[38;5;25m▄[38;5;25m▄[38;5;24m▄[38;5;25m▄[48;5;233;38;5;235m▄[49;39m [48;5;25;38;5;25m▄[48;5;235;38;5;24m▄[48;5;235;38;5;24m▄[48;5;25;38;5;25m▄[48;5;25;38;5;25m▄[48;5;25;38;5;25m▄[48;5;235;38;5;235m█[49;39m         [00m
     [48;5;235;38;5;235m█[48;5;24;38;5;24m█[48;5;25;38;5;25m█[48;5;24;38;5;24m█[48;5;235;38;5;235m█[48;5;25;38;5;25m█[48;5;235;38;5;235m█[49;39m [38;5;235m▀[38;5;235m▀▀▀▀▀[38;5;233m▀[39m [48;5;235;38;5;24m▄[48;5;235;38;5;25m▄[38;5;25m▄[38;5;25m▄[38;5;24m▄[48;5;235;38;5;25m▄[49;39m  [00m
     [48;5;235;38;5;235m▄[48;5;24;38;5;25m▄[48;5;25;38;5;25m▄[48;5;24;38;5;25m▄[48;5;235;38;5;25m▄[48;5;25;38;5;25m▄[48;5;235;38;5;235m▄[49;39m         [48;5;67;38;5;67m█[48;5;25;38;5;25m█[48;5;25;38;5;25m█[48;5;25;38;5;25m█[48;5;235;38;5;235m█[48;5;25;38;5;25m█[49;39m  [00m
   [38;5;233m▄[38;5;235m▄[48;5;235;38;5;24m▄[48;5;235;38;5;25m▄[49;38;5;235m▄[39m             [48;5;67;38;5;25m▄[48;5;25;38;5;25m▄[48;5;25;38;5;25m▄[48;5;25;38;5;25m▄[48;5;235;38;5;25m▄[48;5;25;38;5;25m▄[49;39m  [00m
   [38;5;235m▀[48;5;25;38;5;24m▄[48;5;24;38;5;25m▄[48;5;25;38;5;68m▄[48;5;24;38;5;25m▄[49;38;5;25m▄[39m      [38;5;235m▄[38;5;235m▄[38;5;17m▄[39m       [38;5;25m▄[38;5;25m▄[38;5;235m▄[39m [00m
    [38;5;23m▀[48;5;110;38;5;60m▄[48;5;110;38;5;254m▄[48;5;25;38;5;25m▄[48;5;25;38;5;25m▄[48;5;233;38;5;25m▄[49;38;5;235m▄[38;5;24m▄[38;5;25m▄[48;5;60;38;5;25m▄[48;5;67;38;5;25m▄[48;5;25;38;5;25m▄[48;5;25;38;5;110m▄[48;5;25;38;5;110m▄[48;5;25;38;5;25m▄[48;5;233;38;5;25m▄[49;39m   [38;5;233m▄[48;5;17;38;5;25m▄[48;5;25;38;5;25m▄[48;5;24;38;5;25m▄[48;5;25;38;5;24m▄[49;38;5;233m▀[39m[00m
      [38;5;60m▀[48;5;153;38;5;24m▄[48;5;68;38;5;110m▄[48;5;25;38;5;67m▄[48;5;25;38;5;25m▄[48;5;110;38;5;25m▄[48;5;67;38;5;255m▄[48;5;32;38;5;110m▄[48;5;68;38;5;110m▄[48;5;68;38;5;67m▄[48;5;25;38;5;110m▄[48;5;25;38;5;110m▄[38;5;110m▄[48;5;25;38;5;67m▄[48;5;24;38;5;67m▄[48;5;233;38;5;25m▄[49;38;5;25m▄[48;5;24;38;5;25m▄[48;5;25;38;5;25m█[38;5;25m▄[48;5;25;38;5;24m▄[49;38;5;17m▀[39m [00m
        [38;5;233m▀[38;5;24m▀[48;5;25;38;5;235m▄[48;5;25;38;5;25m█[48;5;153;38;5;110m▄[48;5;67;38;5;110m▄[48;5;252;38;5;255m▄[48;5;254;38;5;231m▄[48;5;254m▄[48;5;253;38;5;224m▄[48;5;252;38;5;255m▄[48;5;110;38;5;231m▄[48;5;110;38;5;231m▄[48;5;61;38;5;110m▄[48;5;25;38;5;25m▄[38;5;24m▄[48;5;25;38;5;233m▄[49;38;5;24m▀[39m   [00m
          [48;5;235;38;5;235m▄[48;5;25;38;5;25m█[48;5;67;38;5;67m▄[48;5;110;38;5;110m▄[48;5;255;38;5;255m▄[48;5;231;38;5;231m█[48;5;255;38;5;216m▄[48;5;223;38;5;209m▄[48;5;223;38;5;223m▄[48;5;231;38;5;231m█[48;5;231;38;5;231m▄[48;5;110;38;5;110m▄[48;5;235;38;5;235m▄[49;39m      [00m
          [48;5;235;38;5;235m▄[48;5;25;38;5;25m█[48;5;32;38;5;25m▄[48;5;67;38;5;25m▄[48;5;255;38;5;254m▄[48;5;231;38;5;255m▄[48;5;209;38;5;180m▄[48;5;209;38;5;223m▄[48;5;224;38;5;173m▄[48;5;231;38;5;255m▄[48;5;231;38;5;255m▄[48;5;110;38;5;67m▄[48;5;235;38;5;235m▄[49;39m      [00m
           [48;5;25;38;5;235m▄[48;5;25;38;5;25m▄[38;5;25m█[48;5;32m▄[48;5;110;38;5;25m▄[48;5;110;38;5;25m▄[48;5;110m▄[48;5;110m▄[48;5;110m▄[48;5;67m▄[48;5;25;38;5;25m▄[49;39m       [00m
            [48;5;25;38;5;25m▄[48;5;25;38;5;25m▄[38;5;25m▄[48;5;25;38;5;25m▄[49;38;5;235m▀[38;5;235m▀[48;5;25;38;5;25m▄[48;5;25;38;5;25m█[48;5;25;38;5;25m▄[49;39m        [00m
         [38;5;188m▄[48;5;242;38;5;188m▄[48;5;242;38;5;188m▄[48;5;25;38;5;250m▄[48;5;25;38;5;67m▄[48;5;67;38;5;67m▄[48;5;25;38;5;68m▄[48;5;250;38;5;25m▄[48;5;188;38;5;188m▄[48;5;25;38;5;110m▄[48;5;68;38;5;32m▄[48;5;25;38;5;67m▄[48;5;250;38;5;68m▄[48;5;188;38;5;251m▄[48;5;247;38;5;237m▄[49;39m     [00m
         [38;5;237m▀[38;5;242m▀[38;5;242m▀[38;5;247m▀[38;5;188m▀[38;5;251m▀[38;5;188m▀[38;5;188m▀[38;5;188m▀[38;5;188m▀[38;5;188m▀[38;5;188m▀[38;5;247m▀[38;5;237m▀[39m      [00m`
