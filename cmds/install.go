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
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
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
	githubURL                = "https://github.com/"
	fabric8io                = "fabric8io"
	funktion                 = "funktion"
	funktionOperator         = "funktion-operator"
	minishiftFlag            = "minishift"
	minishiftOwner           = "jimmidyson"
	minishift                = "minishift"
	minikube                 = "minikube"
	minishiftDownloadURL     = "https://github.com/jimmidyson/"
	kubectl                  = "kubectl"
	kubernetes               = "kubernetes"
	oc                       = "oc"
	ghDownloadURL            = "https://storage.googleapis.com/"
	ocTools                  = "openshift-origin-client-tools"
	stableVersionURL         = "https://storage.googleapis.com/kubernetes-release/release/stable.txt"
)

var (
	githubClient *github.Client
)

type downloadProperties struct {
	clientBinary   string
	kubeDistroOrg  string
	kubeDistroRepo string
	kubeBinary     string
	extraPath      string
	downloadURL    string
	isMiniShift    bool
}

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

	writeFileLocation := getFabric8BinLocation()

	err := os.MkdirAll(writeFileLocation, 0700)
	if err != nil {
		util.Errorf("Unable to create directory to download files %s %v\n", writeFileLocation, err)
	}

	err = downloadDriver()
	if err != nil {
		util.Warnf("Unable to download driver %v\n", err)
	}

	d := getDownloadProperties(isMinishift)
	err = downloadKubernetes(d)
	if err != nil {
		util.Warnf("Unable to download kubernetes distro %v\n", err)
	}

	err = downloadKubectlClient()
	if err != nil {
		util.Warnf("Unable to download client %v\n", err)
	}

	if d.isMiniShift {
		err = downloadOpenShiftClient()
		if err != nil {
			util.Warnf("Unable to download client %v\n", err)
		}
	}

	err = downloadFunktion()
	if err != nil {
		util.Warnf("Unable to download funktion operator %v\n", err)
	}
}
func downloadDriver() (err error) {
	if runtime.GOOS == "darwin" {
		util.Infof("fabric8 recommends OSX users use the xhyve driver\n")
		_, err := exec.LookPath("brew")
		if err != nil {
			util.Fatalf("brew command is not available, see https://brew.sh")
		}

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

func downloadKubernetes(d downloadProperties) (err error) {
	os := runtime.GOOS
	arch := runtime.GOARCH

	if runtime.GOOS == "windows" {
		d.kubeBinary += ".exe"
	}

	_, err = exec.LookPath(d.kubeBinary)
	if err != nil {
		// fix minishift version to 0.9.0 until we can address issues running on 1.x
		latestVersion := "0.9.0"
		if !d.isMiniShift {
			semverVersion, err := getLatestVersionFromGitHub(d.kubeDistroOrg, d.kubeDistroRepo)
			latestVersion = semverVersion.String()
			if err != nil {
				util.Errorf("Unable to get latest version for %s/%s %v", d.kubeDistroOrg, d.kubeDistroRepo, err)
				return err
			}
		}

		kubeURL := fmt.Sprintf(d.downloadURL+d.kubeDistroRepo+"/releases/"+d.extraPath+"v%s/%s-%s-%s", latestVersion, d.kubeDistroRepo, os, arch)
		if runtime.GOOS == "windows" {
			kubeURL += ".exe"
		}
		util.Infof("Downloading %s...\n", kubeURL)

		fullPath := filepath.Join(getFabric8BinLocation(), d.kubeBinary)
		err = downloadFile(fullPath, kubeURL)
		if err != nil {
			util.Errorf("Unable to download file %s/%s %v", fullPath, kubeURL, err)
			return err
		}
		util.Successf("Downloaded %s\n", fullPath)
	} else {
		util.Successf("%s is already available on your PATH\n", d.kubeBinary)
	}

	return nil
}

func downloadKubectlClient() (err error) {

	os := runtime.GOOS
	arch := runtime.GOARCH

	kubectlBinary := kubectl
	if runtime.GOOS == "windows" {
		kubectlBinary += ".exe"
	}

	_, err = exec.LookPath(kubectlBinary)
	if err != nil {
		latestVersion, err := getLatestVersionFromKubernetesReleaseUrl()
		if err != nil {
			return fmt.Errorf("Unable to get latest version for %s/%s %v", kubernetes, kubernetes, err)
		}

		clientURL := fmt.Sprintf("https://storage.googleapis.com/kubernetes-release/release/v%s/bin/%s/%s/%s", latestVersion, os, arch, kubectlBinary)

		util.Infof("Downloading %s...\n", clientURL)

		fullPath := filepath.Join(getFabric8BinLocation(), kubectlBinary)
		err = downloadFile(fullPath, clientURL)
		if err != nil {
			util.Errorf("Unable to download file %s/%s %v", fullPath, clientURL, err)
			return err
		}
		util.Successf("Downloaded %s\n", fullPath)
	} else {
		util.Successf("%s is already available on your PATH\n", kubectlBinary)
	}

	return nil
}

func downloadOpenShiftClient() (err error) {
	var arch string

	ocBinary := "oc"
	if runtime.GOOS == "windows" {
		ocBinary += ".exe"
	}

	_, err = exec.LookPath(ocBinary)
	if err != nil {

		// need to fix the version we download as not able to work out the oc sha in the URL yet
		sha := "dad658de7465ba8a234a4fb40b5b446a45a4cee1"
		latestVersion := "1.3.1"

		clientURL := fmt.Sprintf("https://github.com/openshift/origin/releases/download/v%s/openshift-origin-client-tools-v%s-%s", latestVersion, latestVersion, sha)

		extension := ".zip"
		switch runtime.GOOS {
		case "windows":
			clientURL += "-windows.zip"
		case "darwin":
			clientURL += "-mac.zip"
		default:
			switch runtime.GOARCH {
			case "amd64":
				arch = "64bit"
			case "386":
				arch = "32bit"
			}
			extension = ".tar.gz"
			clientURL += fmt.Sprintf("-%s-%s.tar.gz", runtime.GOOS, arch)
		}

		util.Infof("Downloading %s...\n", clientURL)

		writeFileLocation := getFabric8BinLocation()
		fullPath := filepath.Join(getFabric8BinLocation(), oc+extension)
		dotPath := filepath.Join(getFabric8BinLocation(), ".")

		err = downloadFile(fullPath, clientURL)
		if err != nil {
			util.Errorf("Unable to download file %s/%s %v", writeFileLocation+oc, clientURL, err)
			return err
		}

		switch extension {
		case ".zip":
			err = unzip(fullPath, dotPath)
			if err != nil {
				util.Errorf("Unable to unzip %s %v", fullPath, err)
				return err
			}
		case ".tar.gz":
			err = untargz(fullPath, dotPath, []string{"oc"})
			if err != nil {
				util.Errorf("Unable to untar %s %v", writeFileLocation+oc+".tar.gz", err)
				return err
			}
			os.Remove(fullPath)
		}

		util.Successf("Downloaded %s\n", oc)
	} else {
		util.Successf("%s is already available on your PATH\n", oc)
	}

	return nil
}

func downloadFunktion() (err error) {
	os := runtime.GOOS
	arch := runtime.GOARCH

	_, err = exec.LookPath(funktion)
	if err != nil {
		latestVersion, err := getLatestVersionFromGitHub(fabric8io, funktionOperator)
		if err != nil {
			util.Errorf("Unable to get latest version for %s/%s %v", fabric8io, funktionOperator, err)
			return err
		}

		funktionURL := fmt.Sprintf(githubURL+fabric8io+"/"+funktionOperator+"/releases/download/v%s/%s-%s-%s", latestVersion, funktionOperator, os, arch)
		if runtime.GOOS == "windows" {
			funktionURL += ".exe"
		}
		util.Infof("Downloading %s...\n", funktionURL)

		fullPath := filepath.Join(getFabric8BinLocation(), funktion)
		err = downloadFile(fullPath, funktionURL)
		if err != nil {
			util.Errorf("Unable to download file %s/%s %v", fullPath, funktionURL, err)
			return err
		}
		util.Successf("Downloaded %s\n", fullPath)
	} else {
		util.Successf("%s is already available on your PATH\n", funktion)
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

// get the latest version from kubernetes, parse it and return it
func getLatestVersionFromKubernetesReleaseUrl() (sem semver.Version, err error) {
	response, err := http.Get(stableVersionURL)
	if err != nil {
		return semver.Version{}, fmt.Errorf("Cannot get url " + stableVersionURL)
	}
	defer response.Body.Close()

	bytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return semver.Version{}, fmt.Errorf("Cannot get url body")
	}

	s := strings.TrimSpace(string(bytes))
	if s != "" {
		return semver.Make(strings.TrimPrefix(s, "v"))
	}

	return semver.Version{}, fmt.Errorf("Cannot get release name")
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
		_, err = exec.LookPath(oc)
		if err != nil {
			return false
		}

	} else {
		// check for minikube
		_, err = exec.LookPath(minikube)
		if err != nil {
			return false
		}
	}

	return true
}

func getDownloadProperties(isMinishift bool) downloadProperties {
	d := downloadProperties{}

	if isMinishift {
		d.clientBinary = oc
		d.extraPath = "download/"
		d.kubeBinary = minishift
		d.downloadURL = minishiftDownloadURL
		d.kubeDistroOrg = minishiftOwner
		d.kubeDistroRepo = minishift
		d.isMiniShift = true

	} else {
		d.clientBinary = kubectl
		d.kubeBinary = minikube
		d.downloadURL = ghDownloadURL
		d.kubeDistroOrg = kubernetes
		d.kubeDistroRepo = minikube
		d.isMiniShift = false
	}
	return d
}

func getFabric8BinLocation() string {
	home := homedir.HomeDir()
	if home == "" {
		util.Fatalf("No user home environment variable found for OS %s", runtime.GOOS)
	}
	return filepath.Join(home, ".fabric8", "bin")
}

// untargz a tarball to a target, from
// http://blog.ralch.com/tutorial/golang-working-with-tar-and-gzipf
func untargz(tarball, target string, onlyFiles []string) error {
	zreader, err := os.Open(tarball)
	if err != nil {
		return err
	}
	defer zreader.Close()

	reader, err := gzip.NewReader(zreader)
	defer reader.Close()
	if err != nil {
		panic(err)
	}

	tarReader := tar.NewReader(reader)

	for {
		inkey := false
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		for _, value := range onlyFiles {
			if value == path.Base(header.Name) {
				inkey = true
				break
			}
		}

		if !inkey {
			continue
		}

		path := filepath.Join(target, path.Base(header.Name))
		info := header.FileInfo()
		if info.IsDir() {
			if err = os.MkdirAll(path, info.Mode()); err != nil {
				return err
			}
			continue
		}

		file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(file, tarReader)
		if err != nil {
			return err
		}
	}

	return nil
}

func unzip(archive, target string) error {
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(target, 0755); err != nil {
		return err
	}

	for _, file := range reader.File {
		path := filepath.Join(target, file.Name)
		if file.FileInfo().IsDir() {
			os.MkdirAll(path, file.Mode())
			continue
		}

		fileReader, err := file.Open()
		if err != nil {
			return err
		}
		defer fileReader.Close()

		targetFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}
		defer targetFile.Close()

		if _, err := io.Copy(targetFile, fileReader); err != nil {
			return err
		}

		// make it executable
		os.Chmod(path, 0755)
		if err != nil {
			return err
		}
	}

	return nil
}
