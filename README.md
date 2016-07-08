# gofabric8 installer

gofabric8 is used to validate &amp; deploy fabric8 components on to your Kubernetes or OpenShift environment

Find more information at http://fabric8.io.

When deploying, by default the latest release version is used.  In order to deploy a specific version you can use the `--version` flag as detailed below.

## Getting started

### Install / Update & run

Get latest download URL from [gofabric8 releases](https://github.com/fabric8io/gofabric8/releases)

```sh
sudo rm /tmp/gofabric8
sudo rm -rf /usr/bin/gofabric8
mkdir /tmp/gofabric8
curl --retry 999 --retry-max-time 0  -sSL [[ADD DOWNLOAD URL HERE]] | tar xzv -C /tmp/gofabric8
chmod +x /tmp/gofabric8/gofabric8
sudo mv /tmp/gofabric8/* /usr/bin/
```


### Install the fabric8 microservices platform i

To install the [fabric8 microservices platform](http://fabric8.io/) then run the following:

```sh
gofabric8 --domain=vagrant.f8 deploy 
```

### Usage

```
Usage:
  gofabric8 [flags]
  gofabric8 [command]

Available Commands:
  validate    Validate your Kubernetes or OpenShift environment
  deploy      Deploy fabric8 to your Kubernetes or OpenShift environment
  pull        Pulls the docker images for the given templates
  routes      Creates any missing Routes for services
  secrets     Set up Secrets on your Kubernetes or OpenShift environment
  volume      Creates a persisent volume for fabric8 apps needing persistent disk
  version     Display version & exit
  help        Help about any command

Flags:
      --alsologtostderr=false: log to standard error as well as files
			--api-server="vagrant.f8": The server used to connect to kubernetes/openshift api if different from the --domain param
      --api-version="": The API version to use when talking to the server
      --certificate-authority="": Path to a cert. file for the certificate authority.
      --client-certificate="": Path to a client key file for TLS.
      --client-key="": Path to a client key file for TLS.
      --cluster="": The name of the kubeconfig cluster to use
      --context="": The name of the kubeconfig context to use
	-d  --domain="vagrant.f8": The domain fabric8 should be accessible at.
  -h, --help=false: help for gofabric8
      --insecure-skip-tls-verify=false: If true, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure.
      --kubeconfig="": Path to the kubeconfig file to use for CLI requests.
      --log-backtrace-at=:0: when logging hits line file:N, emit a stack trace
      --log-dir=: If non-empty, write log files in this directory
      --log-flush-frequency=5s: Maximum number of seconds between log flushes
      --logtostderr=true: log to standard error instead of files
      --match-server-version=false: Require server version to match client version
      --namespace="": If present, the namespace scope for this CLI request.
      --password="": Password for basic authentication to the API server.
  -s, --server="": The address and port of the Kubernetes API server
      --stderrthreshold=2: logs at or above this threshold go to stderr
      --token="": Bearer token for authentication to the API server.
      --user="": The name of the kubeconfig user to use
      --username="": Username for basic authentication to the API server.
      --v=0: log level for V logs
      --validate=false: If true, use a schema to validate the input before sending it
  -v, --version="latest": fabric8 version
      --vmodule=: comma-separated list of pattern=N settings for file-filtered logging
  -y, --yes=false: assume yes


Use "gofabric8 [command] --help" for more information about a command.
```

## Development

### Prerequisites

Install [go version 1.4](https://golang.org/doc/install)


### Building

```sh
git clone git@github.com:fabric8io/gofabric8.git $GOPATH/src/github.com/fabric8io/gofabric8
./make
```

Make changes to *.go files, rerun `make` and run the generated binary..

e.g.

```sh
./build/gofabric8 -s https://172.28.128.4:8443 --domain=vagrant.f8 -y --namespace="fabric8" deploy

```
