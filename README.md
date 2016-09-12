# gofabric8 installer

gofabric8 is used to validate &amp; deploy fabric8 components on to your Kubernetes or OpenShift environment

Find more information at http://fabric8.io.

## Getting started

### Install / Update & run

```
FABRIC8_OS=darwin;FABRIC8_VERSION=0.4.64;curl -L -o gofabric8 https://github.com/fabric8io/gofabric8/releases/download/v$FABRIC8_VERSION/gofabric8-$FABRIC8_OS-amd64 && chmod +x gofabric8
gofabric8 version
```

Linux
```
FABRIC8_OS=linux;FABRIC8_VERSION=0.4.64;wget -O gofabric8 https://github.com/fabric8io/gofabric8/releases/download/v$FABRIC8_VERSION/gofabric8-$FABRIC8_OS-amd64; chmod +x gofabric8
gofabric8 version
```

See [latest release](https://github.com/fabric8io/gofabric8/releases/latest/) for more distros

### Install the fabric8 microservices platform

To install the [fabric8 microservices platform](http://fabric8.io/) then run the following:

```sh
gofabric8 deploy --domain=your.domain.io
```

### Run different versions

When deploying, by default the latest release version is used.  In order to deploy a specific version you can use the various`--version-xxxx` flags as detailed under 

```
gofabric8 deploy help
```

### Usage

```
gofabric8 help
gofabric8 is used to validate & deploy fabric8 components on to your Kubernetes or OpenShift environment
       								Find more information at http://fabric8.io.

Usage:
  gofabric8 [flags]
  gofabric8 [command]

Available Commands:
  validate    Validate your Kubernetes or OpenShift environment
  deploy      Deploy fabric8 to your Kubernetes or OpenShift environment
  run         Runs a fabric8 microservice from one of the installed templates
  pull        Pulls the docker images for the given templates
  ingress     Creates any missing Ingress resources for services
  routes      Creates any missing Routes for services
  secrets     Set up Secrets on your Kubernetes or OpenShift environment
  service     Opens the specified Kubernetes service in your browser
  volumes     Creates a persisent volume for any pending persistance volume claims
  version     Display version & exit

Flags:
      --as string                      Username to impersonate for the operation.
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

## Development

### Prerequisites

Install [go version 1.4](https://golang.org/doc/install)

### Developing

```sh
git clone git@github.com:fabric8io/gofabric8.git $GOPATH/src/github.com/fabric8io/gofabric8
make
```

Make changes to *.go files, rerun `make` and execute the generated binary

e.g.

```sh
./build/gofabric8 -s https://172.28.128.4:8443 --domain=vagrant.f8 -y --namespace="fabric8" deploy

```
