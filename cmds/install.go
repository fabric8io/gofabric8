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
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"golang.org/x/oauth2"

	"github.com/blang/semver"
	"github.com/fabric8io/gofabric8/util"
	"github.com/google/go-github/github"
	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/util/homedir"
)

const (
	docker                   = "docker"
	dockerMachine            = "docker-machine"
	dockerMachineDownloadURL = "https://github.com/docker/machine/releases/download/"
	minishiftFlag            = "minishift"
	minishiftOwner           = "jimmidyson"
	minishift                = "minishift"
	minishiftDownloadURL     = "https://github.com/jimmidyson/"
	kubectl                  = "kubectl"
	kubernetes               = "kubernetes"
	oc                       = "oc"
	binLocation              = "/fabric8/bin/"
)

var (
	clientBinary    = kubectl
	kubeDistroOrg   = "kubernetes"
	kubeDistroRepo  = "minikube"
	kubeBinary      = "minikube"
	kubeDownloadURL = "https://storage.googleapis.com/"
	downloadPath    = ""

	writeFileLocation string
	githubClient      *github.Client
)

// NewCmdInstall installs the dependencies to run the fabric8 microservices platform
func NewCmdInstall(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Installs the dependencies to locally run the fabric8 microservices platform",
		Long:  `Installs the dependencies to locally run the fabric8 microservices platform`,

		Run: func(cmd *cobra.Command, args []string) {
			isMinishift := cmd.Flags().Lookup(minishiftFlag).Value.String() == "true"
			install(isMinishift)
		},
	}
	cmd.PersistentFlags().Bool(minishiftFlag, false, "Install minishift rather than minikube")
	return cmd
}

func install(isMinishift bool) {
	if runtime.GOOS == "windows" {
		util.Errorf("%s is not yet supported by gofabric8 install", runtime.GOOS)
	}

	home := homedir.HomeDir()
	if home == "" {
		util.Fatalf("No user home environment variable found for OS %s", runtime.GOOS)
	}
	writeFileLocation = home + binLocation

	err := os.MkdirAll(writeFileLocation, 0700)
	if err != nil {
		util.Errorf("Unable to create directory to download files %s %v\n", writeFileLocation, err)
	}

	err = downloadDriver()
	if err != nil {
		util.Warnf("Unable to download driver %v\n", err)
	}

	err = downloadKubernetes(isMinishift)
	if err != nil {
		util.Warnf("Unable to download kubernetes distro %v\n", err)
	}

	err = downloadClient(isMinishift)
	if err != nil {
		util.Warnf("Unable to download client %v\n", err)
	}
}
func downloadDriver() (err error) {

	if runtime.GOOS == "darwin" {
		util.Infof("fabric8 recommends OSX users use the xhyve driver\n")
		info, err := exec.Command("brew", "info", "docker-machine-driver-xhyve").Output()

		if err != nil || strings.Contains(string(info), "Not installed") {
			e := exec.Command("brew", "install", "docker-machine-driver-xhyve")
			e.Stdout = os.Stdout
			e.Stderr = os.Stderr
			err = e.Run()
			if err != nil {
				return err
			}

			out, err := exec.Command("brew", "--prefix").Output()
			if err != nil {
				return err
			}

			brewPrefix := strings.TrimSpace(string(out))

			file := string(brewPrefix) + "/opt/docker-machine-driver-xhyve/bin/docker-machine-driver-xhyve"
			e = exec.Command("sudo", "chown", "root:wheel", file)
			e.Stdout = os.Stdout
			e.Stderr = os.Stderr
			err = e.Run()
			if err != nil {
				return err
			}

			e = exec.Command("sudo", "chmod", "u+s", file)
			e.Stdout = os.Stdout
			e.Stderr = os.Stderr
			err = e.Run()
			if err != nil {
				return err
			}

			util.Success("xhyve driver installed\n")
		} else {
			util.Success("xhyve driver already installed\n")
		}

	} else if runtime.GOOS == "linux" {
		return errors.New("Driver install for " + runtime.GOOS + " not yet supported")
	}
	return nil
}

func downloadKubernetes(isMinishift bool) (err error) {
	os := runtime.GOOS
	arch := runtime.GOARCH
	if isMinishift {
		kubeDistroOrg = minishiftOwner
		kubeDistroRepo = minishift
		kubeDownloadURL = minishiftDownloadURL
		downloadPath = "download/"
		kubeBinary = minishift
	}

	_, err = exec.LookPath(kubeBinary)
	if err != nil {
		latestVersion, err := getLatestVersionFromGitHub(kubeDistroOrg, kubeDistroRepo)
		if err != nil {
			util.Errorf("Unable to get latest version for %s/%s %v", kubeDistroOrg, kubeDistroRepo, err)
			return err
		}

		kubeURL := fmt.Sprintf(kubeDownloadURL+kubeDistroRepo+"/releases/"+downloadPath+"v%s/%s-%s-%s", latestVersion, kubeDistroRepo, os, arch)
		util.Infof("Downloading %s...", kubeURL)

		err = downloadFile(writeFileLocation+kubeBinary, kubeURL)
		if err != nil {
			util.Errorf("Unable to download file %s/%s %v", writeFileLocation+kubeBinary, kubeURL, err)
			return err
		}
		util.Successf("Downloaded %s\n", kubeBinary)
	} else {
		util.Successf("%s is already available on your PATH\n", kubeBinary)
	}

	return nil
}

func downloadClient(isMinishift bool) (err error) {

	os := runtime.GOOS
	arch := runtime.GOARCH

	_, err = exec.LookPath(kubectl)
	if err != nil {
		latestVersion, err := getLatestVersionFromGitHub(kubeDistroOrg, kubernetes)
		if err != nil {
			return fmt.Errorf("Unable to get latest version for %s/%s %v", kubeDistroOrg, kubernetes, err)
		}

		clientURL := fmt.Sprintf("https://storage.googleapis.com/kubernetes-release/release/v%s/bin/%s/%s/%s", latestVersion, os, arch, kubectl)
		util.Infof("Downloading %s...", clientURL)

		err = downloadFile(writeFileLocation+clientBinary, clientURL)
		if err != nil {
			util.Errorf("Unable to download file %s/%s %v", writeFileLocation+clientBinary, clientURL, err)
			return err
		}
		util.Successf("Downloaded %s\n", clientBinary)
	} else {
		util.Successf("%s is already available on your PATH\n", clientBinary)
	}

	if isMinishift {
		clientBinary = oc
		return fmt.Errorf("Openshift client download not yet supported")
	}

	return nil
}

// download here until install and download binaries are supported in minishift
func downloadFile(filepath string, url string) (err error) {

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	// make it executable
	os.Chmod(filepath, 0755)
	if err != nil {
		return err
	}
	return nil
}

// borrowed from minishift until it supports install / download binaries
func getLatestVersionFromGitHub(githubOwner, githubRepo string) (semver.Version, error) {
	if githubClient == nil {
		token := os.Getenv("GH_TOKEN")
		var tc *http.Client
		if len(token) > 0 {
			ts := oauth2.StaticTokenSource(
				&oauth2.Token{AccessToken: token},
			)
			tc = oauth2.NewClient(oauth2.NoContext, ts)
		}
		githubClient = github.NewClient(tc)
	}
	client := githubClient
	var (
		release *github.RepositoryRelease
		resp    *github.Response
		err     error
	)
	release, resp, err = client.Repositories.GetLatestRelease(githubOwner, githubRepo)
	if err != nil {
		return semver.Version{}, err
	}
	defer resp.Body.Close()
	latestVersionString := release.TagName
	if latestVersionString != nil {
		return semver.Make(strings.TrimPrefix(*latestVersionString, "v"))

	}
	return semver.Version{}, fmt.Errorf("Cannot get release name")
}

func isInstalled(isMinishift bool) bool {
	home := homedir.HomeDir()
	if home == "" {
		util.Fatalf("No user home environment variable found for OS %s", runtime.GOOS)
	}

	// check if we can find a local kube config file
	if _, err := os.Stat(home + "/.kube/config"); os.IsNotExist(err) {
		return false
	}

	// check for kubectl
	_, err := exec.LookPath(kubectl)
	if err != nil {
		return false
	}

	if isMinishift {
		// check for minishift
		_, err = exec.LookPath(minishift)
		if err != nil {
			return false
		}
		// check for oc client
		_, err = exec.LookPath("oc")
		if err != nil {
			return false
		}

	} else {
		// check for minikube
		_, err = exec.LookPath(kubeBinary)
		if err != nil {
			return false
		}
	}

	return true
}
