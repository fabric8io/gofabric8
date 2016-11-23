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
	k8sclient "k8s.io/kubernetes/pkg/client/unversioned"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

const (
	fromNamespaceFlag = "from-namespace"
	toNamespaceFlag   = "to-namespace"
)

func NewCmdCopyEndpoints(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "copy-endpoints",
		Short: "Copies endpoints from the current namespace to a target namespace",
		Long:  `Copies endpoints from the current namespace to a target namespace`,
		PreRun: func(cmd *cobra.Command, args []string) {
			showBanner()
		},
		Run: func(cmd *cobra.Command, args []string) {

			if len(args) == 0 {
				util.Info("Please specify one or more endpoint names to copy as arguments!\n")
				return
			}
			c, cfg := client.NewClient(f)
			oc, _ := client.NewOpenShiftClient(cfg)

			initSchema()

			toNamespace := cmd.Flags().Lookup(toNamespaceFlag).Value.String()

			fromNamespace := cmd.Flags().Lookup(fromNamespaceFlag).Value.String()
			if len(fromNamespace) == 0 {
				ns, _, err := f.DefaultNamespace()
				if err != nil {
					util.Fatal("No default namespace")
				}
				fromNamespace = ns
			}
			if len(toNamespace) == 0 {
				util.Fatal("No target namespace specified!")
			}

			util.Infof("Copying endpoints from namespace: %s to namespace: %s\n", fromNamespace, toNamespace)
			err := ensureNamespaceExists(c, oc, toNamespace)
			if err != nil {
				util.Fatalf("Failed to copy endpoints %v", err)
			}

			err = copyEndpoints(c, fromNamespace, toNamespace, args)
			if err != nil {
				util.Fatalf("Failed to copy endpoints %v", err)
			}
		},
	}
	cmd.PersistentFlags().StringP(fromNamespaceFlag, "f", "", "the source namespace or uses the default namespace")
	cmd.PersistentFlags().StringP(toNamespaceFlag, "t", "", "the destination namespace")
	return cmd
}

func copyEndpoints(c *k8sclient.Client, fromNs string, toNs string, names []string) error {
	var answer error
	for _, name := range names {
		item, err := c.Endpoints(fromNs).Get(name)
		if err != nil {
			util.Warnf("No endpoint called %s found in namespace %s", name, fromNs)
			answer = err
		}
		name := item.Name
		objectMeta := item.ObjectMeta

		current, err := c.Endpoints(toNs).Get(name)
		if current == nil || err != nil {
			// lets create the endpoint
			newEndpoints := &api.Endpoints{
				ObjectMeta: api.ObjectMeta{
					Name:        name,
					Labels:      objectMeta.Labels,
					Annotations: objectMeta.Annotations,
				},
				Subsets: item.Subsets,
			}
			_, err = c.Endpoints(toNs).Create(newEndpoints)
			if err != nil {
				return err
			}
			util.Infof("Created endpoint %s in namespace %s\n", name, toNs)
		}
	}
	return answer
}
