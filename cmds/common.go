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
	"runtime"
	"strings"

	"github.com/daviddengcn/go-colortext"
	"github.com/fabric8io/gofabric8/util"
	"github.com/spf13/cobra"

	"io/ioutil"

	osapi "github.com/openshift/origin/pkg/project/api"
	"k8s.io/kubernetes/pkg/api"
	k8api "k8s.io/kubernetes/pkg/api/unversioned"
	k8client "k8s.io/kubernetes/pkg/client/unversioned"
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

func defaultNamespace(cmd *cobra.Command, f *cmdutil.Factory) (string, error) {
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

// currentProject ...
func detectCurrentUserProject(current string, items []osapi.Project, c *k8client.Client) (chosenone string) {
	var detected []string
	var prefixes = []string{"che", "jenkins", "run", "stage"}

	for _, p := range items {
		name := p.Name
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
	for _, p := range items {
		if p.Name == chosenone {
			cfgmap, err := c.ConfigMaps(p.Name).List(api.ListOptions{LabelSelector: selector})
			cmdutil.CheckErr(err)
			if len(cfgmap.Items) == 0 {
				//TODO: add command line switch to specify the environment if we can't detect it.
				util.Fatalf("Could not autodetect your environment, there is no configmaps environment in the `%s` project.\n", p.Name)
			}
			return
		}
	}

	util.Errorf("Cannot find parent project for: %s\n", current)
	return ""
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

func showBanner() {
	if runtime.GOOS == "windows" {
		return
	}
	ct.ChangeColor(ct.Blue, false, ct.None, false)
	fmt.Println(fabric8AsciiArt)
	ct.ResetColor()
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
