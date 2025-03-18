GOCMD?=go
IMAGE_NAME?=webui

prepare-tests:
	docker compose up -d

cleanup-tests:
	docker compose down

tests: prepare-tests
	LOCALAGENT_MODEL="arcee-agent" LOCALAI_API_URL="http://localhost:8081" LOCALAGENT_API_URL="http://localhost:8080" $(GOCMD) run github.com/onsi/ginkgo/v2/ginkgo --fail-fast -v -r ./...

run-nokb:
	$(MAKE) run KBDISABLEINDEX=true

.PHONY: build
build:
	$(GOCMD) build -o localagent ./

.PHONY: run
run:
	$(GOCMD) run ./

build-image:
	docker build -t $(IMAGE_NAME) -f Dockerfile.webui .

image-push:
	docker push $(IMAGE_NAME)
