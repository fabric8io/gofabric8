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
	yaml "gopkg.in/yaml.v2"
	"k8s.io/kubernetes/pkg/api"
	k8api "k8s.io/kubernetes/pkg/api/unversioned"
	restclient "k8s.io/kubernetes/pkg/client/restclient"
	k8client "k8s.io/kubernetes/pkg/client/unversioned"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

type EnvironmentData struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
	Order     int    `yaml:"order"`
}

func NewCmdGetEnviron(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "environ",
		Short: "Get environment from fabric8-environments configmap",
		Run: func(cmd *cobra.Command, args []string) {
			detectedNS, c, _ := getOpenShiftClient(f)

			selector, err := k8api.LabelSelectorAsSelector(
				&k8api.LabelSelector{MatchLabels: map[string]string{"kind": "environments"}})
			cmdutil.CheckErr(err)

			cfgmap, err := c.ConfigMaps(detectedNS).List(api.ListOptions{LabelSelector: selector})
			cmdutil.CheckErr(err)

			fmt.Printf("%-10s DATA\n", "ENV")
			for _, item := range cfgmap.Items {
				for key, data := range item.Data {
					var ed EnvironmentData
					err := yaml.Unmarshal([]byte(data), &ed)
					cmdutil.CheckErr(err)
					fmt.Printf("%-10s name=%s namespace=%s order=%d\n",
						key, ed.Name, ed.Namespace, ed.Order)
				}
			}
		},
	}
	// NB(chmou): we may try to do the whole shenanigans like kubectl/oc does for
	// outputting stuff but currently this is like swatting flies with a
	// sledgehammer.
	// cmdutil.AddPrinterFlags(cmd)
	return cmd
}

// NewCmdDeleteEnviron is a command to delete an environ using: gofabric8 delete environ abcd
func NewCmdDeleteEnviron(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "environ",
		Short: "Delete environment from fabric8-environments configmap",
		Run: func(cmd *cobra.Command, args []string) {
			detectedNS, c, _ := getOpenShiftClient(f)

			selector, err := k8api.LabelSelectorAsSelector(
				&k8api.LabelSelector{MatchLabels: map[string]string{"kind": "environments"}})
			cmdutil.CheckErr(err)

			if len(args) == 0 {
				util.Errorf("Delete command requires the name of the environment as a parameter.\n.")
				return
			}

			if len(args) != 1 {
				util.Errorf("Delete command can have only one environment name parameter.\n.")
				return
			}

			toDeleteEnv := args[0]

			cfgmap, err := c.ConfigMaps(detectedNS).List(api.ListOptions{LabelSelector: selector})
			cmdutil.CheckErr(err)

			// remove the entry from the config map
			var updatedCfgMap *api.ConfigMap
			for _, item := range cfgmap.Items {
				for k, data := range item.Data {
					var ed EnvironmentData
					err := yaml.Unmarshal([]byte(data), &ed)
					cmdutil.CheckErr(err)

					if ed.Name == toDeleteEnv {
						delete(item.Data, k)
						updatedCfgMap = &item
						goto DeletedConfig
					}

				}
			}

		DeletedConfig:
			if updatedCfgMap == nil {
				util.Warnf("Could not find environment named %s.\n", toDeleteEnv)
				return
			}

			_, err = c.ConfigMaps(detectedNS).Update(updatedCfgMap)
			if err != nil {
				util.Errorf("Failed to update config map after deleting: %v.\n", err)
				return
			}

		},
	}
	return cmd
}

// getOpenShiftClient Get an openshift client and detect the project we want to
// be in
func getOpenShiftClient(f *cmdutil.Factory) (detectedNS string, c *k8client.Client, cfg *restclient.Config) {
	c, cfg = client.NewClient(f)

	initSchema()

	typeOfMaster := util.TypeOfMaster(c)
	isOpenshift := typeOfMaster == util.OpenShift

	if isOpenshift {
		oc, _ := client.NewOpenShiftClient(cfg)
		projects, err := oc.Projects().List(api.ListOptions{})
		if err != nil {
			util.Warnf("Could not list projects: %v", err)
		} else {
			currentNS, _, _ := f.DefaultNamespace()
			detectedNS = detectCurrentUserProject(currentNS, projects.Items)
		}
	}

	if detectedNS == "" {
		detectedNS, _, _ = f.DefaultNamespace()
	}

	return
}
