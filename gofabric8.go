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
package main

import (
	"io"
	"os"
	"runtime"

	commands "github.com/fabric8io/gofabric8/cmds"
	"github.com/fabric8io/gofabric8/util"
	"github.com/fabric8io/gofabric8/version"
	"github.com/kubernetes/minikube/pkg/minikube/config"
	"github.com/minishift/minishift/pkg/minikube/update"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

const (
	batchFlag = "batch"

	githubOrg          = "fabric8io"
	githubRepo         = "gofabric8"
	binaryName         = githubRepo
	lastUpdateCheck    = "last_update_check"
	hiddenFolder       = "/.fabric8/"
	versionConsoleFlag = "version-console"
	workProject        = "work-project"
)

func runHelp(cmd *cobra.Command, args []string) {
	cmd.Help()
}

func NewGoFabric8Command(f cmdutil.Factory, in io.Reader, out, err io.Writer) *cobra.Command {
	cmds := &cobra.Command{
		Use:   "gofabric8",
		Short: "gofabric8 is used to validate & deploy fabric8 components on to your Kubernetes or OpenShift environment",
		Long: `gofabric8 is used to validate & deploy fabric8 components on to your Kubernetes or OpenShift environment
								Find more information at http://fabric8.io.`,
		Run: runHelp,
	}

	cmds.PersistentFlags().String(workProject, "autodetect", "The work project")
	cmds.PersistentFlags().String(versionConsoleFlag, "latest", "fabric8 version")
	cmds.PersistentFlags().BoolP("yes", "y", false, "Assume yes")
	cmds.PersistentFlags().BoolP(batchFlag, "b", false, "Run in batch mode to avoid prompts. Can also be enabled via `export FABRIC8_BATCH=true`")

	f.BindFlags(cmds.PersistentFlags())

	updated := false
	oldHandler := cmds.PersistentPreRun
	cmds.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if !updated {
			updated = true
			flag := cmds.Flags().Lookup(batchFlag)
			batch := false
			if flag != nil {
				batch = flag.Value.String() == "true"
			}
			batchFlag := os.Getenv("FABRIC8_BATCH")
			if batchFlag == "true" {
				batch = true
			}

			if !batch {
				home := os.Getenv("HOME")
				if home == "" {
					util.Fatalf("No user home environment variable found for OS %s", runtime.GOOS)
				}
				writeFileLocation := home + hiddenFolder
				err := os.MkdirAll(writeFileLocation, 0700)
				if err != nil {
					util.Errorf("Unable to create directory to store update file %s %v\n", writeFileLocation, err)
				}
				localVersion, err := version.GetSemverVersion()
				if err != nil {
					util.Errorf("Unable to get local version %v", err)
				}
				viper.SetDefault(config.WantUpdateNotification, true)
				viper.SetDefault(config.ReminderWaitPeriodInHours, 24)
				update.MaybeUpdate(os.Stdout, githubOrg, githubRepo, binaryName, writeFileLocation+lastUpdateCheck, localVersion)

			}
		}
		if oldHandler != nil {
			oldHandler(cmd, args)
		}
	}

	cmds.AddCommand(commands.NewCmdBddEnv(f))
	cmds.AddCommand(commands.NewCmdCopyEndpoints(f))
	cmds.AddCommand(commands.NewCmdCheShell(f))
	cmds.AddCommand(commands.NewCmdCleanUp(f, out))
	cmds.AddCommand(commands.NewCmdConsole(f))
	cmds.AddCommand(commands.NewCmdDeploy(f))
	cmds.AddCommand(commands.NewCmdDockerEnv(f))
	cmds.AddCommand(commands.NewCmdIngress(f))
	cmds.AddCommand(commands.NewCmdInstall(f))
	cmds.AddCommand(commands.NewCmdIP(f))
	cmds.AddCommand(commands.NewCmdLog(f))
	cmds.AddCommand(commands.NewCmdPackages(f))
	cmds.AddCommand(commands.NewCmdPackageVersions(f))
	cmds.AddCommand(commands.NewCmdPull(f))
	cmds.AddCommand(commands.NewCmdRoutes(f))
	cmds.AddCommand(commands.NewCmdRun(f))
	cmds.AddCommand(commands.NewCmdE2eConsole(f))
	cmds.AddCommand(commands.NewCmdE2eEnv(f))
	cmds.AddCommand(commands.NewCmdE2ESecret(f))
	cmds.AddCommand(commands.NewCmdE2ETest(f))
	cmds.AddCommand(commands.NewCmdSecrets(f))
	cmds.AddCommand(commands.NewCmdService(f))
	cmds.AddCommand(commands.NewCmdStart(f))
	cmds.AddCommand(commands.NewCmdStatus(f))
	cmds.AddCommand(commands.NewCmdStop(f))
	cmds.AddCommand(commands.NewCmdTest(f))
	cmds.AddCommand(commands.NewCmdValidate(f))
	cmds.AddCommand(commands.NewCmdUpgrade(f))
	cmds.AddCommand(commands.NewCmdVersion())
	cmds.AddCommand(commands.NewCmdVolumes(f))
	cmds.AddCommand(commands.NewCmdWaitFor(f))
	cmds.AddCommand(commands.NewCmdTenant(f))
	cmds.AddCommand(commands.NewCmdErasePVC(f))

	getcmd := commands.NewCmdGet(f, out)
	cmds.AddCommand(getcmd)

	createcmd := commands.NewCmdCreate(f, out)
	cmds.AddCommand(createcmd)

	deletecmd := commands.NewCmdDelete(f, out)
	cmds.AddCommand(deletecmd)

	cmds.AddCommand(commands.NewCmdCompletion(f, out))

	return cmds
}

func main() {
	cmd := NewGoFabric8Command(cmdutil.NewFactory(nil), os.Stdin, os.Stdout, os.Stderr)
	cmd.Execute()
}
