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
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"

	"github.com/fabric8io/almighty-core/log"
	"github.com/fabric8io/fabric8-init-tenant/openshift"
	"github.com/fabric8io/gofabric8/util"

	"github.com/spf13/cobra"

	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"

	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

type cmdTenant struct {
	cmd  *cobra.Command
	args []string

	apiserver   string
	username    string
	token       string
	templateDir string
	teamVersion string
}

func NewCmdTenant(f *cmdutil.Factory) *cobra.Command {
	p := &cmdTenant{}
	cmd := &cobra.Command{
		Use:   "tenant",
		Short: "Initialises/Upgrades a Tenant (a user/team) with a set of namespaces along with Jenkins and Che",
		Run: func(cmd *cobra.Command, args []string) {
			p.cmd = cmd
			p.args = args
			handleError(p.run(f))

		},
	}
	flags := cmd.Flags()
	flags.StringVarP(&p.apiserver, "apiserver", "a", "", "the URL of the Kubernetes API Server to use. Defaults to the current kubectl/oc context cluster's URL")
	flags.StringVarP(&p.username, "username", "u", "", "the username to communicate with the API server. Defaults to the current kubectl/oc context's user name")
	flags.StringVarP(&p.token, "token", "t", "", "the token to communicate with the API server. Defaults to the current kubectl/oc context cluster's token")
	flags.StringVarP(&p.templateDir, "template-dir", "d", "", "the directory to look for templates")
	flags.StringVarP(&p.teamVersion, "team-version", "v", "", "the version to use for the team templates")
	return cmd
}

func (p *cmdTenant) run(f *cmdutil.Factory) error {
	apiserver := p.apiserver
	token := p.token
	username := p.username

	clientConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	).RawConfig()

	if err != nil {
		util.Warnf("Failed to load client config: %s\n", err)
	} else {
		currentContext := clientConfig.CurrentContext
		context := clientConfig.Contexts[currentContext]
		if context != nil {
			if len(apiserver) == 0 {
				clusterName := context.Cluster
				if len(clusterName) > 0 {
					cluster := clientConfig.Clusters[clusterName]
					if cluster != nil {
						apiserver = cluster.Server

					}
				}
			}
			user := context.AuthInfo
			if len(user) > 0 {
				userInfo := clientConfig.AuthInfos[user]
				if userInfo != nil {
					if len(token) == 0 {
						token = userInfo.Token
					}
					if len(username) == 0 {
						username = userInfo.Username
					}
				}
				if len(username) == 0 {
					arr := strings.Split(user, "/")
					if len(arr) > 0 {
						username = arr[0]
					}
				}
			}
		}
	}

	util.Infof("\nInitialising tenant at server: %s for user: %s\n", apiserver, username)

	if len(apiserver) == 0 {
		return fmt.Errorf("No value detected or configured for `apiserver`")
	}
	if len(username) == 0 {
		return fmt.Errorf("No value detected or configured for `username`")
	}
	if len(token) == 0 {
		return fmt.Errorf("No value detected or configured for `token`")
	}

	osConfig := openshift.Config{
		MasterURL:   apiserver,
		MasterUser:  username,
		Token:       token,
		TemplateDir: p.templateDir,
		TeamVersion: p.teamVersion,
		HttpTransport: &http.Transport{
			// we need to disable TLS verify on minishift
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		LogCallback: logMessage,
	}
	templateVars := map[string]string{}

	log.InitializeLogger(false, "debug")
	return openshift.InitTenant(osConfig, defaultCallback, username, token, templateVars)
}

func logMessage(message string) {
	util.Info(message + "\n")
}

func defaultCallback(statusCode int, method string, request, response map[interface{}]interface{}) (string, map[interface{}]interface{}) {
	metadata := map[interface{}]interface{}{}
	value := request["metadata"]
	m, ok := value.(map[interface{}]interface{})
	if ok {
		metadata = m
	}
	util.Infof("status %d on %s/%s %s message: %s\n", statusCode, metadata["namespace"], metadata["name"], request["kind"], response["message"])
	return method, response
}
