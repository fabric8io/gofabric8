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
	"net"

	"net/url"

	"github.com/fabric8io/gofabric8/client"
	"github.com/fabric8io/gofabric8/util"
	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

const ()

// NewCmdIP returns the IP for the cluster gofabric8 is connected to
func NewCmdIP(f cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ip",
		Short: "Returns the IP for the cluster gofabric8 is connected to",
		Long:  `Returns the IP for the cluster gofabric8 is connected to`,

		Run: func(cmd *cobra.Command, args []string) {
			_, cfg := client.NewClient(f)
			u, err := url.Parse(cfg.Host)
			if err != nil {
				util.Fatalf("%s", err)
			}
			ip, _, err := net.SplitHostPort(u.Host)
			if err != nil {
				util.Errorf("Unable to get IP address of connected cluster: %v", err)
			} else {
				util.Infof("%s", ip)
			}
		},
	}
	return cmd
}
