SHELL := /bin/bash
tag := $(shell cat .openshift-version)

build:
	godep go build -o go-fabric8 -a *.go

update-deps:
	echo $(tag) > .openshift-version && \
		pushd $(GOPATH)/src/github.com/openshift/origin && \
		git fetch origin && \
		git checkout -B $(tag) refs/tags/$(tag) && \
		godep restore && \
		popd && \
		godep save && \
		godep update ...
