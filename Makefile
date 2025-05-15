GOCMD?=go
IMAGE_NAME?=webui
MCPBOX_IMAGE_NAME?=mcpbox
ROOT_DIR:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

prepare-tests: build-mcpbox
	docker compose up -d --build
	docker run -d -v /var/run/docker.sock:/var/run/docker.sock --privileged -p 9090:8080 --rm -ti $(MCPBOX_IMAGE_NAME)

cleanup-tests:
	docker compose down

tests: prepare-tests
	LOCALAGI_MCPBOX_URL="http://localhost:9090" LOCALAGI_MODEL="qwen3-8b" LOCALAI_API_URL="http://localhost:8081" LOCALAGI_API_URL="http://localhost:8080" $(GOCMD) run github.com/onsi/ginkgo/v2/ginkgo --fail-fast -v -r ./...

run-nokb:
	$(MAKE) run KBDISABLEINDEX=true

webui/react-ui/dist:
	docker run --entrypoint /bin/bash -v $(ROOT_DIR):/app oven/bun:1 -c "cd /app/webui/react-ui && bun install && bun run build"

.PHONY: build
build: webui/react-ui/dist
	$(GOCMD) build -o localagi ./

.PHONY: run
run: webui/react-ui/dist
	LOCALAGI_MCPBOX_URL="http://localhost:9090" $(GOCMD) run ./

build-image:
	docker build -t $(IMAGE_NAME) -f Dockerfile.webui .

image-push:
	docker push $(IMAGE_NAME)

build-mcpbox:
	docker build -t $(MCPBOX_IMAGE_NAME) -f Dockerfile.mcpbox .

run-mcpbox:
	docker run -v /var/run/docker.sock:/var/run/docker.sock --privileged -p 9090:8080 -ti mcpbox