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
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/fabric8io/gofabric8/client"
	"github.com/fabric8io/gofabric8/util"
	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

var (
	removedExportedLine = []string{
		"selfLink", "resourceVersion", "uid", "creationTimestamp",
		"kubectl.kubernetes.io/last-applied-configuration:",
		"control-plane.alpha.kubernetes.io/leader:",
		"pv.kubernetes.io/",
		"volume.beta.kubernetes.io/",
		"volumeName"}
)

type erasePVCFlags struct {
	cmd    *cobra.Command
	args   []string
	userNS string

	volumeName string
}

// NewCmdErasePVC Erase PVC https://github.com/fabric8io/gofabric8/issues/598
func NewCmdErasePVC(f cmdutil.Factory) *cobra.Command {
	p := &erasePVCFlags{}
	cmd := &cobra.Command{
		Use:   "erase-pvc",
		Short: "Erase PVC",
		Long:  `Erase PVC`,

		Run: func(cmd *cobra.Command, args []string) {
			p.cmd = cmd
			p.args = args
			p.userNS = cmd.Flags().Lookup(namespaceFlag).Value.String()

			if len(p.args) != 1 {
				util.Fatal("We need a PVC to delete as argument.\n")
			}
			p.volumeName = p.args[0]

			handleError(p.erasePVC(f))
		},
	}
	cmd.PersistentFlags().StringP(namespaceFlag, "n", "", "The namespace where the PVC is located. Defaults to the current namespace")
	return cmd
}

// writeLines writes the lines to the given file.
func writeLines(lines []string, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
	return w.Flush()
}

func (p *erasePVCFlags) erasePVC(f cmdutil.Factory) (err error) {
	var userNS string

	if p.userNS != "" {
		userNS = p.userNS
	} else {
		c, cfg := client.NewClient(f)
		ns, _, _ := f.DefaultNamespace()
		oc, _ := client.NewOpenShiftClient(cfg)
		initSchema()

		userNS, err = detectCurrentUserNamespace(ns, c, oc)
		cmdutil.CheckErr(err)
	}

	cmd := []string{"get", "-o", "yaml", "-n", userNS, "pvc", p.volumeName}
	output, err := runCommandWithOutput("kubectl", cmd...)
	if err != nil {
		util.Fatal("Error while running cmd: " + strings.Join(cmd, " ") + " Error: " + err.Error() + " Output: " + output + "\n")
	}

	inStatus := false
	scanner := bufio.NewScanner(strings.NewReader(output))
	var outputYAML []string
	for scanner.Scan() {
		text := scanner.Text()
		nsLine := strings.TrimSpace(text)

		stop := false
		for _, l := range removedExportedLine {
			if strings.HasPrefix(nsLine, l) {
				stop = true
			}
		}
		if stop {
			continue
		}
		if text == "status:" {
			inStatus = true
			continue
		}

		if inStatus && string(text[0]) != " " {
			inStatus = false
		} else if inStatus {
			continue
		}
		outputYAML = append(outputYAML, text)
	}
	tmpfile, err := ioutil.TempFile("", "gofabric8")
	cmdutil.CheckErr(err)

	err = writeLines(outputYAML, tmpfile.Name())
	cmdutil.CheckErr(err)

	cmd = []string{"delete", "-n", userNS, "pvc", p.volumeName}
	output, err = runCommandWithOutput("kubectl", cmd...)
	if err != nil {
		util.Fatal("Error while running cmd: " + strings.Join(cmd, " ") + " Error: " + err.Error() + " Output: " + output + "\n")
	}

	cmd = []string{"create", "-n", userNS, "-f", tmpfile.Name()}
	output, err = runCommandWithOutput("kubectl", cmd...)
	if err != nil {
		util.Fatal("Error while running cmd: " + strings.Join(cmd, " ") + " Error: " + err.Error() + " Output: " + output + "\n")
	}

	util.Success("Volume: " + p.volumeName + " has been recreated.\n")
	os.Remove(tmpfile.Name())

	return
}
