#
# Copyright (C) 2015 Red Hat, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#         http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

SHELL := /bin/bash
NAME := gofabric8
GO := GO15VENDOREXPERIMENT=1 go
VERSION := $(shell cat version/VERSION)
OPENSHIFT_TAG := $(shell cat .openshift-version)
ROOT_PACKAGE := $(shell $(GO) list .)
GO_VERSION := $(shell $(GO) version | sed -e 's/^[^0-9.]*\([0-9.]*\).*/\1/')
PACKAGE_DIRS := $(shell $(GO) list ./... | grep -v /vendor/)
FORMATTED := $(shell $(GO) fmt $(PACKAGE_DIRS))

REV        := $(shell git rev-parse --short HEAD 2> /dev/null  || echo 'unknown')
BRANCH     := $(shell git rev-parse --abbrev-ref HEAD 2> /dev/null  || echo 'unknown')
BUILD_DATE := $(shell date +%Y%m%d-%H:%M:%S)
BUILDFLAGS := -ldflags \
  " -X $(ROOT_PACKAGE)/version.Version=$(VERSION)\
		-X $(ROOT_PACKAGE)/version.Revision='$(REV)'\
		-X $(ROOT_PACKAGE)/version.Branch='$(BRANCH)'\
		-X $(ROOT_PACKAGE)/version.BuildDate='$(BUILD_DATE)'\
		-X $(ROOT_PACKAGE)/version.GoVersion='$(GO_VERSION)'"
CGO_ENABLED = 0

VENDOR_DIR=vendor
INIT_TENANT_DIR=$(VENDOR_DIR)/github.com/fabric8io/fabric8-init-tenant
INIT_TENANT_BINDATA=$(INIT_TENANT_DIR)/template/bindata.go
GO_BINDATA_BIN=$(VENDOR_DIR)/github.com/jteeuwen/go-bindata/go-bindata/go-bindata
FULL_GO_BINDATA_BIN=$(shell pwd)/$(GO_BINDATA_BIN)
GO_BINDATA_DIR=$(VENDOR_DIR)/github.com/jteeuwen/go-bindata/go-bindata/
GO_BINDATA_ASSETFS_BIN=$(VENDOR_DIR)/github.com/elazarl/go-bindata-assetfs/go-bindata-assetfs/go-bindata-assetfs
TEAM_VERSION=$(shell cat TEAM_VERSION)

full-build: $(INIT_TENANT_BINDATA) build
 
build: *.go */*.go fmt
	rm -rf build
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(BUILDFLAGS) -o build/$(NAME) $(NAME).go

test:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test github.com/fabric8io/gofabric8/cmds

install: *.go */*.go
	GOBIN=${GOPATH}/bin $(GO) install $(BUILDFLAGS) $(NAME).go

fmt:
	@([[ ! -z "$(FORMATTED)" ]] && printf "Fixed unformatted files:\n$(FORMATTED)") || true

arm:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=arm $(GO) build $(BUILDFLAGS) -o build/$(NAME)-arm $(NAME).go

win:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=windows GOARCH=amd64 $(GO) build $(BUILDFLAGS) -o build/$(NAME).exe $(NAME).go

bootstrap: vendoring $(INIT_TENANT_BINDATA)

templates: $(INIT_TENANT_BINDATA)

clean-templates:
	rm $(INIT_TENANT_DIR)/template/bindata.go
	rm $(INIT_TENANT_DIR)/template/fabric8-online-che-openshift.yml
	rm $(INIT_TENANT_DIR)/template/fabric8-online-jenkins-openshift.yml
	rm $(INIT_TENANT_DIR)/template/fabric8-online-team-openshift.yml

$(GO_BINDATA_BIN): $(VENDOR_DIR)
	cd $(VENDOR_DIR)/github.com/jteeuwen/go-bindata/go-bindata && go build -v
$(GO_BINDATA_ASSETFS_BIN): $(VENDOR_DIR)
	cd $(VENDOR_DIR)/github.com/elazarl/go-bindata-assetfs/go-bindata-assetfs && go build -v

#$(INIT_TENANT_DIR)/template/bindata.go: $(GO_BINDATA_BIN) $(wildcard $(INIT_TENANT_DIR)/template/*.yml)

$(INIT_TENANT_BINDATA): $(GO_BINDATA_BIN)
	export TEAM_VERSION=$(TEAM_VERSION)
	echo "using team version ${TEAM_VERSION}"
	cd $(INIT_TENANT_DIR) && TEAM_VERSION=$(TEAM_VERSION) go generate template/generate.go
	cd $(INIT_TENANT_DIR) && $(FULL_GO_BINDATA_BIN) -o template/bindata.go \
	-pkg template \
	-prefix '' \
	-nocompress \
	template


vendoring:
	$(GO) get -u github.com/Masterminds/glide
	GO15VENDOREXPERIMENT=1 glide update --strip-vendor --strip-vcs --update-vendored

tools:
	$(GO) get -u get -u github.com/jteeuwen/go-bindata/go-bindata/go-bindata

release: test
	rm -rf build release && mkdir build release
	for os in linux darwin ; do \
		CGO_ENABLED=$(CGO_ENABLED) GOOS=$$os GOARCH=amd64 $(GO) build $(BUILDFLAGS) -o build/$(NAME)-$$os-amd64 $(NAME).go ; \
	done
	CGO_ENABLED=$(CGO_ENABLED) GOOS=windows GOARCH=amd64 $(GO) build $(BUILDFLAGS) -o build/$(NAME)-windows-amd64.exe $(NAME).go
	zip --junk-paths release/$(NAME)-windows-amd64.zip build/$(NAME)-windows-amd64.exe README.md LICENSE
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=arm $(GO) build $(BUILDFLAGS) -o build/$(NAME)-linux-arm $(NAME).go
	cp build/$(NAME)-*-amd64* release
	cp build/$(NAME)-*-arm* release
	go get -u github.com/progrium/gh-release
	gh-release checksums sha256
	gh-release create fabric8io/$(NAME) $(VERSION) $(BRANCH) $(VERSION)


clean:
	rm -rf build release

docker:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=amd64 $(GO) build $(BUILDFLAGS) -o build/$(NAME)-linux-amd64 $(NAME).go
	docker build -t fabric8/gofabric8 .

.PHONY: release clean arm
