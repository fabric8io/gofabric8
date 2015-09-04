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
NAME=gofabric8
VERSION=$(shell cat VERSION)
OPENSHIFT_TAG := $(shell cat .openshift-version)

build: *.go */*.go
	CGO_ENABLED=0 godep go build -o build/$(NAME) -a gofabric8.go

install: *.go */*.go
	GOBIN=${GOPATH}/bin godep go install -a gofabric8.go

update-deps-old:
	echo $(OPENSHIFT_TAG) > .openshift-version && \
		pushd $(GOPATH)/src/github.com/openshift/origin && \
		git fetch origin && \
		git checkout -B $(OPENSHIFT_TAG) refs/tags/$(OPENSHIFT_TAG) && \
		godep restore && \
		popd && \
		godep save ./... && \
		godep update ...

update-deps:
	echo $(OPENSHIFT_TAG) > .openshift-version && \
		pushd $(GOPATH)/src/github.com/openshift/origin && \
		git fetch origin && \
		git checkout -B $(OPENSHIFT_TAG) refs/tags/$(OPENSHIFT_TAG) && \
		godep restore && \
		popd && \
		godep save cmd/generate/generate.go && \
		godep update ... && \
		rm -rf Godeps/_workspace/src/github.com/GoogleCloudPlatform/kubernetes && \
		cp -r $(GOPATH)/src/github.com/openshift/origin/Godeps/_workspace/src/github.com/GoogleCloudPlatform/kubernetes Godeps/_workspace/src/github.com/GoogleCloudPlatform/kubernetes


release:
	rm -rf build release && mkdir build release
	for os in linux darwin ; do \
		CGO_ENABLED=0 GOOS=$$os ARCH=amd64 godep go build -ldflags "-X main.Version $(VERSION)" -o build/$(NAME)-$$os-amd64 -a gofabric8.go ; \
		tar --transform 's|^build/||' --transform 's|-.*||' -czvf release/$(NAME)-$(VERSION)-$$os-amd64.tar.gz build/$(NAME)-$$os-amd64 README.md LICENSE ; \
	done
	CGO_ENABLED=0 GOOS=windows ARCH=amd64 godep go build -ldflags "-X main.Version $(VERSION)" -o build/$(NAME)-$(VERSION)-windows-amd64.exe -a gofabric8.go
	zip release/$(NAME)-$(VERSION)-windows-amd64.zip build/$(NAME)-$(VERSION)-windows-amd64.exe README.md LICENSE && \
		echo -e "@ build/$(NAME)-$(VERSION)-windows-amd64.exe\n@=$(NAME).exe"  | zipnote -w release/$(NAME)-$(VERSION)-windows-amd64.zip
	go get github.com/progrium/gh-release/...
	gh-release create fabric8io/$(NAME) $(VERSION) \
		$(shell git rev-parse --abbrev-ref HEAD) $(VERSION)

clean:
		rm -rf build release

.PHONY: release clean
