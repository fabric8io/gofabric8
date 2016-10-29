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
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"path/filepath"

	"github.com/fabric8io/gofabric8/util"
	"github.com/kardianos/osext"
	"github.com/spf13/cobra"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

const (
	memory   = "memory"
	vmDriver = "vm-driver"
	cpus     = "cpus"
	console  = "console"
	ipaas    = "ipaas"
)

// NewCmdStart starts a local cloud environment
func NewCmdStart(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Starts a local cloud development environment",
		Long:  `Starts a local cloud development environment`,

		Run: func(cmd *cobra.Command, args []string) {

			flag := cmd.Flags().Lookup(minishift)
			isOpenshift := false
			if flag != nil {
				isOpenshift = flag.Value.String() == "true"
			}

			flag = cmd.Flags().Lookup(ipaas)
			isIPaaS := false
			if flag != nil && flag.Value.String() == "true" {
				isOpenshift = true
				isIPaaS = true
			}

			if !isInstalled(isOpenshift) {
				install(isOpenshift)
			}
			kubeBinary := minikube
			if isOpenshift {
				kubeBinary = minishift
			}

			if runtime.GOOS == "windows" && !strings.HasSuffix(kubeBinary, ".exe") {
				kubeBinary += ".exe"
			}

			binaryFile := resolveBinaryLocation(kubeBinary)

			// check if already running
			out, err := exec.Command(binaryFile, "status").Output()
			if err != nil {
				util.Fatalf("Unable to get status %v", err)
			}

			if err == nil && strings.Contains(string(out), "Running") {
				// already running
				util.Successf("%s already running\n", kubeBinary)

				kubectlBinaryFile := resolveBinaryLocation(kubectl)

				// setting context
				if kubeBinary == minikube {
					e := exec.Command(kubectlBinaryFile, "config", "use-context", kubeBinary)
					e.Stdout = os.Stdout
					e.Stderr = os.Stderr
					err = e.Run()
					if err != nil {
						util.Errorf("Unable to start %v", err)
					}
				} else {
					// minishift context has changed, we need to work it out now
					util.Info("minishift is already running, you can switch to the context\n")
				}

			} else {
				args := []string{"start"}

				vmDriverValue := cmd.Flags().Lookup(vmDriver).Value.String()
				if len(vmDriverValue) == 0 {
					switch runtime.GOOS {
					case "darwin":
						vmDriverValue = "xhyve"
					case "windows":
						vmDriverValue = "hyperv"
					case "linux":
						vmDriverValue = "kvm"
					default:
						vmDriverValue = "virtualbox"
					}

				}
				args = append(args, "--vm-driver="+vmDriverValue)

				// set memory flag
				memoryValue := cmd.Flags().Lookup(memory).Value.String()
				args = append(args, "--memory="+memoryValue)

				// set cpu flag
				cpusValue := cmd.Flags().Lookup(cpus).Value.String()
				args = append(args, "--cpus="+cpusValue)

				// start the local VM
				logCommand(binaryFile, args)
				e := exec.Command(binaryFile, args...)
				e.Stdout = os.Stdout
				e.Stderr = os.Stderr
				err = e.Run()
				if err != nil {
					util.Errorf("Unable to start %v", err)
				}
			}

			if isOpenshift {
				// deploy fabric8
				e := exec.Command("oc", "login", "--username="+minishiftDefaultUsername, "--password="+minishiftDefaultPassword)
				e.Stdout = os.Stdout
				e.Stderr = os.Stderr
				err = e.Run()
				if err != nil {
					util.Errorf("Unable to login %v", err)
				}

			}

			// now check that fabric8 is running, if not deploy it
			c, err := keepTryingToGetClient(f)
			if err != nil {
				util.Fatalf("Unable to connect to %s %v", kubeBinary, err)
			}

			// deploy fabric8 if its not already running
			ns, _, _ := f.DefaultNamespace()
			_, err = c.Services(ns).Get("fabric8")
			if err != nil {

				// deploy fabric8
				d := GetDefaultFabric8Deployment()
				flag := cmd.Flags().Lookup(console)
				if isIPaaS {
					d.packageName = "ipaas"
				} else if flag != nil && flag.Value.String() == "true" {
					d.packageName = "console"
				} else {
					d.packageName = cmd.Flags().Lookup(packageFlag).Value.String()
				}
				d.pv = true
				deploy(f, d)

			} else {
				openService(ns, "fabric8", c, false)
			}
		},
	}
	cmd.PersistentFlags().BoolP(minishift, "", false, "start the openshift flavour of Kubernetes")
	cmd.PersistentFlags().BoolP(console, "", false, "start only the fabric8 console")
	cmd.PersistentFlags().BoolP(ipaas, "", false, "start the fabric8 iPaaS")
	cmd.PersistentFlags().StringP(memory, "", "4096", "amount of RAM allocated to the VM")
	cmd.PersistentFlags().StringP(vmDriver, "", "", "the VM driver used to spin up the VM. Possible values (hyperv, xhyve, kvm, virtualbox, vmwarefusion)")
	cmd.PersistentFlags().StringP(cpus, "", "1", "number of CPUs allocated to the VM")
	cmd.PersistentFlags().String(packageFlag, "platform", "The name of the package to startup such as 'platform', 'console', 'ipaas'. Otherwise specify a URL or local file of the YAML to install")
	return cmd
}

func logCommand(executable string, args []string) {
	util.Infof("running: %s %s\n", executable, strings.Join(args, " "))
}

// lets find the executable on the PATH or in the fabric8 directory
func resolveBinaryLocation(executable string) string {
	path, err := exec.LookPath(executable)
	if err != nil || fileNotExist(path) {
		home := os.Getenv("HOME")
		if home == "" {
			util.Error("No $HOME environment variable found")
		}
		writeFileLocation := getFabric8BinLocation()

		// lets try in the fabric8 folder
		path = filepath.Join(writeFileLocation, executable)
		if fileNotExist(path) {
			path = executable
			// lets try in the folder where we found the gofabric8 executable
			folder, err := osext.ExecutableFolder()
			if err != nil {
				util.Errorf("Failed to find executable folder: %v\n", err)
			} else {
				path = filepath.Join(folder, executable)
				if fileNotExist(path) {
					util.Infof("Could not find executable at %v\n", path)
					path = executable
				}
			}
		}
	}
	util.Infof("using the executable %s\n", path)
	return path
}

func findExecutable(file string) error {
	d, err := os.Stat(file)
	if err != nil {
		return err
	}
	if m := d.Mode(); !m.IsDir() {
		return nil
	}
	return os.ErrPermission
}

func fileNotExist(path string) bool {
	return findExecutable(path) != nil
}

func keepTryingToGetClient(f *cmdutil.Factory) (*client.Client, error) {
	timeout := time.After(2 * time.Minute)
	tick := time.Tick(1 * time.Second)
	// Keep trying until we're timed out or got a result or got an error
	for {
		select {
		// Got a timeout! fail with a timeout error
		case <-timeout:
			return nil, errors.New("timed out")
		// Got a tick, try and get teh client
		case <-tick:
			c, _ := getClient(f)
			// return if we have a client
			if c != nil {
				return c, nil
			}
			util.Info("Cannot connect to api server, retrying...\n")
			// retry
		}
	}
}

func getClient(f *cmdutil.Factory) (*client.Client, error) {
	var err error
	cfg, err := f.ClientConfig()
	if err != nil {
		return nil, err
	}
	c, err := client.New(cfg)
	if err != nil {
		return nil, err
	}
	return c, nil
}
