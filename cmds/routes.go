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
	"os"
	"github.com/fabric8io/gofabric8/client"
	"github.com/fabric8io/gofabric8/util"
	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

func NewCmdRoutes(f *cmdutil.Factory) *cobra.Command {
	defaultDomain := os.Getenv("KUBERNETES_DOMAIN")
	if defaultDomain == "" {
		defaultDomain = DefaultDomain
	}
	cmd := &cobra.Command{
		Use:   "routes",
		Short: "Creates any missing Routes for services",
		Long:  `Creates any missing Route resources for Services which need to be exposed remotely`,
		Run: func(cmd *cobra.Command, args []string) {
			c, cfg := client.NewClient(f)
			oc, _ := client.NewOpenShiftClient(cfg)
			ns, _, err := f.DefaultNamespace()
			if err != nil {
				util.Fatal("No default namespace")
				printResult("Get default namespace", Failure, err)
			} else {
				util.Info("Creating a persistent volume for your ")
				util.Success(string(util.TypeOfMaster(c)))
				util.Info(" installation at ")
				util.Success(cfg.Host)
				util.Info(" in namespace ")
				util.Successf("%s\n\n", ns)

				domain := cmd.Flags().Lookup(domainFlag).Value.String()

				err := createRoutesForDomain(ns, domain, c, oc, f)
				printError("Create Routes", err)
			}
		},
	}
	cmd.PersistentFlags().StringP(domainFlag, "", defaultDomain, "The domain to put the created routes inside")
	return cmd
}
