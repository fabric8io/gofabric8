# gofabric8

[![Go Report Card](https://goreportcard.com/badge/github.com/fabric8io/gofabric8)](https://goreportcard.com/report/github.com/fabric8io/gofabric8)
[![APACHEv2 License](https://img.shields.io/badge/license-APACHEv2-blue.svg)](https://github.com/fabric8io/gofabric8/blob/master/LICENSE)

gofabric8 is used to validate & deploy fabric8 components on to your Kubernetes
or OpenShift environment

Find more information at http://fabric8.io.

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->


- [Getting started](#getting-started)
  - [Install gofabric8](#install-gofabric8)
  - [Install dependencies](#install-dependencies)
    - [Install minikube](#install-minikube)
    - [Install minishift](#install-minishift)
  - [Install the fabric8 microservices platform](#install-the-fabric8-microservices-platform)
  - [Reusing the Docker daemon](#reusing-the-docker-daemon)
  - [Run different versions](#run-different-versions)
  - [Usage](#usage)
- [Shell completion](#shell-completion)
- [Development](#development)
  - [Prerequisites](#prerequisites)
  - [Developing](#developing)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Getting started

### Install gofabric8

Get the [latest](https://github.com/fabric8io/gofabric8/releases/latest/)
`gofabric8` or use the following script to download it.

```
curl -sS https://get.fabric8.io/download.txt | bash
```

Add '~/.fabric8/bin' to your path so you can execute the new binaries, for
example: edit your ~/.zshrc or ~/.bashrc and append to the end of the file

```
export PATH=$PATH:~/.fabric8/bin
source ~/.zshrc or ~/.bashrc
```

### Install dependencies

The `gofabric8 install` installs the dependencies to locally run the fabric8
microservices platform - either [minishift][minishift] and [openshift
client][oc] or [minikube][minikube] and [kubectl][kubectl] along with necessary
drivers. The binaries are downloaded to `~/.fabric8/bin`.

#### Install minikube

```
gofabric8 install
```

#### Install minishift

```
gofabric8 install --minishift
```

### Install the fabric8 microservices platform

To install the [fabric8 microservices platform](http://fabric8.io/) then run the following:

```sh
gofabric8 deploy
```

If you are deploying to a remote OpenShift instance make sure to set the domain
so we can generate routes to access applications

```
gofabric8 deploy --domain=your.domain.io
```

### Reusing the Docker daemon

When developing locally and using a single VM its really handy to reuse the
Docker daemon inside the VM; as this means you don't have to build on your host
machine and push the image into a docker registry - you can just build inside
the same docker daemon as minikube which speeds up local experiments.

To be able to work with the docker daemon on your mac/linux host use the
docker-env command in your shell:

```
eval $(gofabric8 docker-env)
```

You should now be able to use docker on the command line on your host mac/linux
machine talking to the docker daemon inside the minikube VM:

```
docker ps
```

Remember to turn off the `imagePullPolicy:Always`, as otherwise kubernetes won't
use images you built locally.

### Run different versions

When deploying, by default the latest release version is used. In order to
deploy a specific version you can use the various`--version-xxxx` flags as
detailed under

```
gofabric8 deploy help
```

### Usage

```
gofabric8 is used to validate & deploy fabric8 components on to your Kubernetes or OpenShift environment.
Find more information at http://fabric8.io.

Usage:
  gofabric8 [flags]
  gofabric8 [command]

Available Commands:
  console     Open the fabric8 console
  deploy      Deploy fabric8 to your Kubernetes or OpenShift environment
  docker-env  Sets up docker env variables; Usage 'eval $(gofabric8 docker-env)'
  ingress     Creates any missing Ingress resources for services
  install     Installs the dependencies to locally run the fabric8 microservices platform
  pull        Pulls the docker images for the given templates
  routes      Creates any missing Routes for services
  run         Runs a fabric8 microservice from one of the installed templates
  secrets     Set up Secrets on your Kubernetes or OpenShift environment
  service     Opens the specified Kubernetes service in your browser
  start       Starts a local cloud development environment
  validate    Validate your Kubernetes or OpenShift environment
  version     Display version & exit
  volumes     Creates a persisent volume for any pending persistance volume claims

Flags:
      --as string                      Username to impersonate for the operation.
  -b, --batch                          Run in batch mode to avoid prompts
      --certificate-authority string   Path to a cert. file for the certificate authority.
      --client-certificate string      Path to a client certificate file for TLS.
      --client-key string              Path to a client key file for TLS.
      --cluster string                 The name of the kubeconfig cluster to use
      --context string                 The name of the kubeconfig context to use
      --fabric8-version string         fabric8 version (default "latest")
      --insecure-skip-tls-verify       If true, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure.
      --kubeconfig string              Path to the kubeconfig file to use for CLI requests.
      --log-flush-frequency duration   Maximum number of seconds between log flushes (default 5s)
      --match-server-version           Require server version to match client version
      --namespace string               If present, the namespace scope for this CLI request.
      --password string                Password for basic authentication to the API server.
  -s, --server string                  The address and port of the Kubernetes API server
      --token string                   Bearer token for authentication to the API server.
      --user string                    The name of the kubeconfig user to use
      --username string                Username for basic authentication to the API server.
  -y, --yes                            assume yes

Use "gofabric8 [command] --help" for more information about a command.
```

## Shell completion

``gofabric8`` provides shell completions, so you can easily complete commands while in the shell.

Simply add this to your ``~/.bashrc`` if you use bash :

```sh
  source <(gofabric8 completion bash)
```

or to your ``~/.zshrc`` if you use zsh ::

```sh
  source <(gofabric8 completion zsh)
```

## Development

### Prerequisites

Install [go version 1.7.4](https://golang.org/doc/install)

### Developing

```sh
git clone git@github.com:fabric8io/gofabric8.git $GOPATH/src/github.com/fabric8io/gofabric8
make
```

Make changes to *.go files, rerun `make` and execute the generated binary

e.g.

```sh
./build/gofabric8 deploy

```

[kubectl]: https://kubernetes.io/docs/reference/kubectl/overview/
[minikube]: https://github.com/kubernetes/minikube
[minishift]: https://github.com/minishift/minishift
[oc]: https://docs.openshift.org/latest/cli_reference/index.html
