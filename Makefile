GOCMD?=go
IMAGE_NAME?=webui

tests:
	$(GOCMD) run github.com/onsi/ginkgo/v2/ginkgo --fail-fast -v -r ./...

webui:
	cd example/webui && $(GOCMD) run ./

webui-image:
	docker build -t $(IMAGE_NAME) -f Dockerfile.webui .