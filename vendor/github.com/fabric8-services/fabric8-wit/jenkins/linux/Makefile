# This Makefile is deprecated. Everything is now available through the main Makefile.

.PHONY: all
.ONESHELL:
all:
	cd ../../
	make docker-start
	-make docker-deps
	-make docker-generate
	-make docker-build
	-make docker-test-unit
	-make integration-test-env-prepare
	-make docker-test-integration
	-make integration-test-env-tear-down
	make docker-rm
