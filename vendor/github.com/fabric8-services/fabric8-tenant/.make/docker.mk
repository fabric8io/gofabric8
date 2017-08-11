DOCKER_IMAGE_CORE := $(PROJECT_NAME)
DOCKER_IMAGE_DEPLOY := $(PROJECT_NAME)-deploy

# If running in Jenkins we don't allow for interactively running the container
ifneq ($(BUILD_TAG),)
	DOCKER_RUN_INTERACTIVE_SWITCH :=
else
	DOCKER_RUN_INTERACTIVE_SWITCH := -i
endif

# The workspace environment is set by Jenkins and defaults to /tmp if not set
WORKSPACE ?= /tmp
DOCKER_BUILD_DIR := $(WORKSPACE)/$(PROJECT_NAME)-build

# The BUILD_TAG environment variable will be set by jenkins
# to reflect jenkins-${JOB_NAME}-${BUILD_NUMBER}
BUILD_TAG ?= $(PROJECT_NAME)-local-build
DOCKER_CONTAINER_NAME := $(BUILD_TAG)

# Where is the GOPATH inside the build container?
GOPATH_IN_CONTAINER=/tmp/go
PACKAGE_PATH=$(GOPATH_IN_CONTAINER)/src/$(PACKAGE_NAME)

.PHONY: docker-image-builder
## Builds the docker image used to build the software.
docker-image-builder:
	@echo "Building docker image $(DOCKER_IMAGE_CORE)"
	docker build -t $(DOCKER_IMAGE_CORE) -f $(CUR_DIR)/Dockerfile.builder $(CUR_DIR)

.PHONY: docker-image-deploy
## Creates a runnable image using the artifacts from the bin directory.
docker-image-deploy:
	docker build -t $(DOCKER_IMAGE_DEPLOY) -f $(CUR_DIR)/Dockerfile.deploy $(CUR_DIR)

.PHONY: docker-publish-deploy
## Tags the runnable image and pushes it to the docker hub.
docker-publish-deploy:
	docker tag $(DOCKER_IMAGE_DEPLOY) fabric8-services/${PROJECT_NAME}:latest
	docker push fabric8-services/${PROJECT_NAME}:latest

.PHONY: docker-build-dir
## Creates the docker build directory.
docker-build-dir:
	@echo "Creating build directory $(BUILD_DIR)"
	mkdir -p $(DOCKER_BUILD_DIR)

CLEAN_TARGETS += clean-docker-build-container
.PHONY: clean-docker-build-container
## Removes any existing container used to build the software (if any).
clean-docker-build-container:
	@echo "Removing container named \"$(DOCKER_CONTAINER_NAME)\" (if any)"
ifneq ($(strip $(shell docker ps -qa --filter "name=$(DOCKER_CONTAINER_NAME)" 2>/dev/null)),)
	@docker rm -f $(DOCKER_CONTAINER_NAME)
else
	@echo "No container named \"$(DOCKER_CONTAINER_NAME)\" to remove"
endif

CLEAN_TARGETS += clean-docker-build-dir
.PHONY: clean-docker-build-dir
## Removes the docker build directory.
clean-docker-build-dir:
	@echo "Cleaning build directory $(BUILD_DIR)"
	-rm -rf $(DOCKER_BUILD_DIR)

.PHONY: docker-start
## Starts the docker build container in the background (detached mode).
## After calling this command you can invoke all the make targets from the
## normal Makefile (e.g. deps, generate, build) inside the build container
## by prefixing them with "docker-". For example to execute "make deps"
## inside the build container, just run "make docker-deps".
## To remove the container when no longer needed, call "make docker-rm".
docker-start: docker-build-dir docker-image-builder
ifneq ($(strip $(shell docker ps -qa --filter "name=$(DOCKER_CONTAINER_NAME)" 2>/dev/null)),)
	@echo "Docker container \"$(DOCKER_CONTAINER_NAME)\" already exists. To recreate, run \"make docker-rm\"."
else
	docker run \
		--detach=true \
		-t \
		$(DOCKER_RUN_INTERACTIVE_SWITCH) \
		--name="$(DOCKER_CONTAINER_NAME)" \
		-v $(CUR_DIR):$(PACKAGE_PATH):Z \
		-u $(shell id -u $(USER)):$(shell id -g $(USER)) \
		-e GOPATH=$(GOPATH_IN_CONTAINER) \
		-w $(PACKAGE_PATH) \
		$(DOCKER_IMAGE_CORE)
		@echo "Docker container \"$(DOCKER_CONTAINER_NAME)\" created. Continue with \"make docker-deps\"."
endif

.PHONY: docker-rm
## Removes the docker build container, if any (see "make docker-start").
docker-rm:
ifneq ($(strip $(shell docker ps -qa --filter "name=$(DOCKER_CONTAINER_NAME)" 2>/dev/null)),)
	docker rm -f "$(DOCKER_CONTAINER_NAME)"
else
	@echo "No container named \"$(DOCKER_CONTAINER_NAME)\" to remove."
endif

# The targets in the following list all depend on a running database container.
# Make sure you run "make integration-test-env-prepare" before you run any of these targets.
DB_DEPENDENT_DOCKER_TARGETS = docker-test-migration docker-test-integration docker-test-integration-no-coverage docker-coverage-all

$(DB_DEPENDENT_DOCKER_TARGETS):
	$(eval makecommand:=$(subst docker-,,$@))
ifeq ($(strip $(shell docker ps -qa --filter "name=$(DOCKER_CONTAINER_NAME)" 2>/dev/null)),)
	$(error No container name "$(DOCKER_CONTAINER_NAME)" exists to run the build. Try running "make docker-start")
endif
ifeq ($(strip $(shell docker inspect --format '{{ .NetworkSettings.IPAddress }}' make_postgres_integration_test_1 2>/dev/null)),)
	$(error Failed to find PostgreSQL container. Try running "make integration-test-env-prepare")
endif
	$(eval F8_POSTGRES_HOST := $(shell docker inspect --format '{{ .NetworkSettings.IPAddress }}' make_postgres_integration_test_1 2>/dev/null))
	docker exec -t $(DOCKER_RUN_INTERACTIVE_SWITCH) "$(DOCKER_CONTAINER_NAME)" bash -ec 'export F8_POSTGRES_HOST=$(F8_POSTGRES_HOST); export F8_POSTGRES_DATABASE=postgres; make $(makecommand)'

# This is a wildcard target to let you call any make target from the normal makefile
# but it will run inside the docker container. This target will only get executed if
# there's no specialized form available. For example if you call "make docker-start"
# not this target gets executed but the "docker-start" target.
docker-%:
	$(eval makecommand:=$(subst docker-,,$@))
ifeq ($(strip $(shell docker ps -qa --filter "name=$(DOCKER_CONTAINER_NAME)" 2>/dev/null)),)
	$(error No container name "$(DOCKER_CONTAINER_NAME)" exists to run the command "make $(makecommand)")
endif
	docker exec -t $(DOCKER_RUN_INTERACTIVE_SWITCH) "$(DOCKER_CONTAINER_NAME)" bash -ec 'make $(makecommand)'
