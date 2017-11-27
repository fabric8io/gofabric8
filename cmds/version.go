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
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/fabric8io/gofabric8/util"
	"github.com/fabric8io/gofabric8/version"
	"github.com/spf13/cobra"
)

var versionInfoTmpl = `
gofabric8, version {{.version}} (branch: {{.branch}}, revision: {{.revision}})
  build date:       {{.buildDate}}
  go version:       {{.goVersion}}
  {{if .ocVersion}}oc version:       '{{.ocVersion}}'{{end}}
  {{if .remoteServerURL}}Remote URL:       '{{.remoteServerURL}}'{{end}}
  {{if .remoteServerOpenShiftVersion}}Remote OpenShift:        '{{.remoteServerOpenShiftVersion}}'{{end}}
  {{if .remoteServerKubernetesVersion}}Remote Kubernetes:       '{{.remoteServerKubernetesVersion}}'{{end}}
  {{if .minikubeVersion}}Minikube:         '{{.minikubeVersion}}'{{end}}
  {{if .minishiftVersion}}Minishift:        '{{.minishiftVersion}}'{{end}}
`

func NewCmdVersion() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Display version & exit",
		Long:  `display version & exit`,
		Run: func(cmd *cobra.Command, args []string) {
			t := template.Must(template.New("version").Parse(versionInfoTmpl))

			// If we are remote print information
			minitype, _, err := util.GetMiniType()
			ocPath, err := getBinary(ocBinary)
			if err == nil {
				output, err := runCommandWithOutput(ocPath, "version")
				if err != nil {
					util.Fatalf("Error while running cmd: %s Output: %s\n", ocPath, output)
				}
				scanner := bufio.NewScanner(strings.NewReader(output))
				for scanner.Scan() {
					text := strings.TrimSpace(scanner.Text())
					if strings.HasPrefix(text, "oc") {
						text = strings.Replace(text, "oc", "", -1)
						version.Map["ocVersion"] = strings.TrimSpace(text)
					}

					if strings.HasPrefix(text, "Server") {
						text = strings.Replace(text, "Server", "", -1)
						version.Map["remoteServerURL"] = strings.TrimSpace(text)
					}
					if strings.HasPrefix(text, "openshift") && version.Map["remoteServerURL"] != "" {
						text = strings.Replace(text, "openshift", "", -1)
						version.Map["remoteServerOpenShiftVersion"] = strings.TrimSpace(text)
					}
					if strings.HasPrefix(text, "kubernetes") && version.Map["remoteServerURL"] != "" {
						text = strings.Replace(text, "kubernetes", "", -1)
						version.Map["remoteServerKubernetesVersion"] = strings.TrimSpace(text)
					}
				}

			}

			if minitype == util.Minikube {
				minikubePath, err := getBinary(minikube)
				if err == nil {
					output, err := runCommandWithOutput(minikubePath, "version")
					if err == nil {
						version.Map["minikubeVersion"] = strings.TrimSpace(strings.Replace(output, "minikube version:", "", -1))
					}
				}
			}

			if minitype == util.Minishift {
				minishiftPath, err := getBinary(minishift)
				if err == nil {
					output, err := runCommandWithOutput(minishiftPath, "version")
					if err == nil {
						version.Map["minishiftVersion"] = strings.TrimSpace(strings.Replace(output, "minishift", "", -1))
					}
				}
			}

			var buf bytes.Buffer
			if err := t.ExecuteTemplate(&buf, "version", version.Map); err != nil {
				panic(err)
			}
			fmt.Fprintln(os.Stdout, strings.TrimSpace(buf.String()))

		},
	}
	return cmd
}
