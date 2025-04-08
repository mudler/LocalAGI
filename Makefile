GOCMD?=go
IMAGE_NAME?=webui
ROOT_DIR:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

prepare-tests:
	docker compose up -d

cleanup-tests:
	docker compose down

tests: prepare-tests
	LOCALAGI_MODEL="arcee-agent" LOCALAI_API_URL="http://localhost:8081" LOCALAGI_API_URL="http://localhost:8080" $(GOCMD) run github.com/onsi/ginkgo/v2/ginkgo --fail-fast -v -r ./...

run-nokb:
	$(MAKE) run KBDISABLEINDEX=true

webui/react-ui/dist:
	docker run --entrypoint /bin/bash -v $(ROOT_DIR):/app oven/bun:1 -c "cd /app/webui/react-ui && bun install && bun run build"

.PHONY: build
build: webui/react-ui/dist
	$(GOCMD) build -o localagi ./

.PHONY: run
run: webui/react-ui/dist
	$(GOCMD) run ./

build-image:
	docker build -t $(IMAGE_NAME) -f Dockerfile.webui .

image-push:
	docker push $(IMAGE_NAME)
