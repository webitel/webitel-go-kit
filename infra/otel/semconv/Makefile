DOCKER_USER=$(shell id -u):$(shell id -g)

# The weaver docker image to use for generate.
WEAVER_IMAGE := otel/weaver:v0.26.1@sha256:9094862c0ab261bdbcb079bb981f9a573b3659b130a6d2ab8616eca6ba37aaec

TAG ?= v0.2.0
REGISTRY ?= https://github.com/webitel/opentelemetry-semantic-conventions/archive/refs/tags/$(TAG).zip[model]

# A local registry directory is mounted into the container.
ifneq ($(wildcard $(REGISTRY)),)
REGISTRY_MOUNT := --mount 'type=bind,source=$(abspath $(REGISTRY)),target=/home/weaver/registry,readonly'
REGISTRY_ARG := /home/weaver/registry
else
REGISTRY_ARG := $(REGISTRY)
endif

.PHONY: generate
generate:
	# Ensure the target directory for source code is available.
	mkdir -p $(CURDIR)/${TAG}
	# Note: We mount a home directory for downloading/storing the semconv repository.
	# Weaver will automatically clean the cache when finished, but the directories will remain.
	mkdir -p ~/.weaver
	docker run --rm \
		-u $(DOCKER_USER) \
		--env HOME=/tmp/weaver \
		--mount 'type=bind,source=$(CURDIR)/templates,target=/home/weaver/templates,readonly' \
		--mount 'type=bind,source=$(CURDIR)/${TAG},target=/home/weaver/target' \
		--mount 'type=bind,source=$(HOME)/.weaver,target=/tmp/weaver/.weaver' \
		$(REGISTRY_MOUNT) \
		$(WEAVER_IMAGE) registry generate \
		--registry='$(REGISTRY_ARG)' \
		--templates=/home/weaver/templates \
		--param tag=$(TAG) \
		go \
		/home/weaver/target
	gofmt -w $(CURDIR)/${TAG}
