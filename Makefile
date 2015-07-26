SHELL := /bin/bash
tag := $(shell cat .openshift-version)

build: *.go */*.go
	godep go build -o build/gofabric8 gofabric8.go

update-deps:
	echo $(tag) > .openshift-version && \
		pushd $(GOPATH)/src/github.com/openshift/origin && \
		git fetch origin && \
		git checkout -B $(tag) refs/tags/$(tag) && \
		godep restore && \
		popd && \
		godep save ./... && \
		godep update ...
