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
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

type logFlags struct {
	name string
}

// NewCmdLog tails the log of the newest pod for a Deployment or DeploymentConfig
func NewCmdLog(f *cmdutil.Factory) *cobra.Command {
	p := &logFlags{}
	cmd := &cobra.Command{
		Use:   "log",
		Short: "Tails the log of the newest pod for the given named Deployment or DeploymentConfig",
		Long:  `Tails the log of the newest pod for the given named Deployment or DeploymentConfig`,

		Run: func(cmd *cobra.Command, args []string) {
			err := p.tailLog(f, args)
			if err != nil {
				util.Fatalf("%s", err)
			}
			return
		},
	}
	flags := cmd.Flags()
	flags.StringVarP(&p.name, "name", "n", "", "the name of the Deployment or DeploymentConfig to log")
	return cmd
}

func (p *logFlags) tailLog(f *cmdutil.Factory, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("Must specify a Deployment/DeploymentConfig name argument!")
	}
	p.name = args[0]
	c, cfg := client.NewClient(f)
	ns, _, _ := f.DefaultNamespace()
	oc, _ := client.NewOpenShiftClient(cfg)
	initSchema()

	for {
		pod, err := waitForReadyPodForDeploymentOrDC(c, oc, ns, p.name)
		if err != nil {
			return err
		}
		if pod == "" {
			return fmt.Errorf("No pod found for ")
		}
		err = runCommand("kubectl", "logs", "-f", pod)
		if err != nil {
			return nil
		}
	}
	return nil
}
