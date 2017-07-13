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
	"strings"

	"github.com/fabric8io/gofabric8/client"
	"github.com/fabric8io/gofabric8/util"
	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
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
			pv := cmd.Flags().Lookup(pvFlag).Value.String() == "true"
			create := cmd.Flags().Lookup(updateFlag).Value.String() != "true"

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
			exposer := cmd.Flags().Lookup(exposerFlag).Value.String()
			githubClientID := cmd.Flags().Lookup(githubClientIDFlag).Value.String()
			githubClientSecret := cmd.Flags().Lookup(githubClientSecretFlag).Value.String()

			params := defaultParameters(c, exposer, githubClientID, githubClientSecret, ns)

			yes := cmd.Flags().Lookup(yesFlag).Value.String() == "false"
			if strings.Contains(domain, "=") {
				util.Warnf("\nInvalid domain: %s\n\n", domain)

			} else if confirmAction(yes) {
				oc, _ := client.NewOpenShiftClient(cfg)
				initSchema()

				for _, app := range args {
					runTemplate(c, oc, app, ns, domain, apiserver, pv, create, params)
				}
			}
		},
	}
	cmd.PersistentFlags().StringP(domainFlag, "d", defaultDomain(), "The domain name to append to the service name to access web applications")
	cmd.PersistentFlags().String(apiServerFlag, "", "overrides the api server url")
	cmd.PersistentFlags().String(exposerFlag, "", "The exposecontroller strategy such as Ingress, Router, NodePort, LoadBalancer")
	cmd.PersistentFlags().String(githubClientIDFlag, "", "The github OAuth Application Client ID. Defaults to $GITHUB_OAUTH_CLIENT_ID if not specified")
	cmd.PersistentFlags().String(githubClientSecretFlag, "", "The github OAuth Application Client Secret. Defaults to $GITHUB_OAUTH_CLIENT_SECRET if not specified")
	cmd.PersistentFlags().Bool(pvFlag, true, "Enable the use of persistence (enabling the PersistentVolumeClaims)?")
	cmd.PersistentFlags().Bool(updateFlag, false, "Enable update mode which updates any existing resources?")
	return cmd
}
