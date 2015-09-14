# gofabric8 installer

gofabric8 is used to validate & deploy fabric8 components on to your Kubernetes or OpenShift environment
								Find more information at http://fabric8.io.

## Getting started

### Download & run

```sh
curl -L https://github.com/fabric8io/gofabric8/releases/download/v0.2/gofabric8-0.2-darwin-amd64.tar.gz | tar -xvz
chmod +x gofabric8
./gofabric8 -s https://172.28.128.4:8443 --domain=vagrant.f8 deploy
```

### Usage

```sh
Usage:
  gofabric8 [flags]
  gofabric8 [command]

Available Commands:
  validate    Validate your Kubernetes or OpenShift environment
  deploy      Deploy fabric8 to your Kubernetes or OpenShift environment
  help        Help about any command

Flags:
      --alsologtostderr=false: log to standard error as well as files
      --api-version="": The API version to use when talking to the server
      --certificate-authority="": Path to a cert. file for the certificate authority.
      --client-certificate="": Path to a client key file for TLS.
      --client-key="": Path to a client key file for TLS.
      --cluster="": The name of the kubeconfig cluster to use
      --context="": The name of the kubeconfig context to use
  -d, --domain="vagrant.f8": The domain name to append to the service name to access web applications
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


Use "gofabric8 help [command]" for more information about a command.
```

## Development

### Pre-requisits

Install [go version 1.4](https://golang.org/doc/install)
Install [godep](https://https://github.com/tools/godep)


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
