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
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"strings"
)

func NewCmdRun(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Runs a fabric8 microservice from one of the installed templates",
		Long:  `runs a fabric8 microservice from one of the installed templates`,
		PreRun: func(cmd *cobra.Command, args []string) {
			showBanner()
		},
		Run: func(cmd *cobra.Command, args []string) {
			c, cfg := client.NewClient(f)
			ns, _, _ := f.DefaultNamespace()

			if len(args) == 0 {
				util.Info("Please specify a template name to run\n")
				return
			}
			domain := cmd.Flags().Lookup(domainFlag).Value.String()
			apiserver := cmd.Flags().Lookup(apiServerFlag).Value.String()
			noPV := cmd.Flags().Lookup(noPVFlag).Value.String() == "true"

			typeOfMaster := util.TypeOfMaster(c)

			util.Info("Running an app template to your ")
			util.Success(string(typeOfMaster))
			util.Info(" installation at ")
			util.Success(cfg.Host)
			util.Info(" for domain ")
			util.Success(domain)
			util.Info(" in namespace ")
			util.Successf("%s\n\n", ns)

			if len(apiserver) == 0 {
				apiserver = domain
			}

			if strings.Contains(domain, "=") {
				util.Warnf("\nInvalid domain: %s\n\n", domain)
			} else if confirmAction(cmd.Flags()) {
				oc, _ := client.NewOpenShiftClient(cfg)

				for _, app := range args {
					runTemplate(c, oc, app, ns, domain, apiserver, noPV)
				}
			}
		},
	}
	cmd.PersistentFlags().StringP(domainFlag, "d", defaultDomain(), "The domain name to append to the service name to access web applications")
	cmd.PersistentFlags().String(apiServerFlag, "", "overrides the api server url")
	cmd.PersistentFlags().Bool(noPVFlag, false, "Disable the use of persistence (disabling the PersistentVolumeClaims)?")
	return cmd
}
