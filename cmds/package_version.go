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

	"github.com/fabric8io/gofabric8/client"
	"github.com/fabric8io/gofabric8/util"
	"github.com/spf13/cobra"
	kapi "k8s.io/kubernetes/pkg/api"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"

	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

func NewCmdPackageVersions(f cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "package-versions name",
		Short: "Displays the versions available for a package",
		Long:  `Displays the versions available for a package`,
		Run: func(cmd *cobra.Command, args []string) {
			c, cfg := client.NewClient(f)
			ns, _, err := f.DefaultNamespace()
			if err != nil {
				util.Fatal("No default namespace")
				printResult("Get default namespace", Failure, err)
			} else {
				if len(args) == 0 {
					util.Failure("Please specify the name of a package as an argument\n\n")
					return
				}

				name := args[0]

				util.Info("Checking versions of package in your ")
				util.Success(string(util.TypeOfMaster(c)))
				util.Info(" installation at ")
				util.Success(cfg.Host)
				util.Info(" in namespace ")
				util.Successf("%s\n\n", ns)

				err = packageVersions(ns, c, name)
				if err != nil {
					util.Failuref("%v", err)
					util.Blank()
				}
			}
		},
	}
	return cmd
}

func packageVersions(ns string, c *clientset.Clientset, name string) error {
	selector, err := createPackageSelector()
	if err != nil {
		return err
	}
	list, err := c.ConfigMaps(ns).List(kapi.ListOptions{
		LabelSelector: *selector,
	})
	if err != nil {
		util.Errorf("Failed to load package in namespace %s with error %v", ns, err)
		return err
	}

	found := false
	for _, p := range list.Items {
		if name == p.Name {
			found = true
			metadataUrl := p.Data[metadataUrlKey]
			if len(metadataUrl) == 0 {
				util.Warnf("Invalid package %s it is missing the `%s` data\n", name, metadataUrl)
				continue
			}
			m, err := loadMetadata(metadataUrl)
			if err != nil {
				return fmt.Errorf("Failed to load package metadata at %s due to %v", metadataUrl, err)
			}
			util.Info("Versions of package ")
			util.Success(name)
			util.Info(":\n")
			versions := m.Versions
			for i := len(versions) - 1; i >= 0; i-- {
				v := versions[i]
				util.Success(v)
				util.Info("\n")
			}
		}
	}
	if !found {
		return fmt.Errorf("No package could be found for name: %s\n", name)
	}
	return nil
}
