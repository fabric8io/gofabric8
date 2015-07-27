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
tag := $(shell cat .openshift-version)

build: *.go */*.go
	godep go build -o build/gofabric8 -a gofabric8.go

install: *.go */*.go
	GOBIN=${GOPATH}/bin godep go install -a gofabric8.go

update-deps:
	echo $(tag) > .openshift-version && \
		pushd $(GOPATH)/src/github.com/openshift/origin && \
		git fetch origin && \
		git checkout -B $(tag) refs/tags/$(tag) && \
		godep restore && \
		popd && \
		godep save ./... && \
		godep update ...
