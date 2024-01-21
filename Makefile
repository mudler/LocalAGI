GOCMD=go

tests:
	$(GOCMD) run github.com/onsi/ginkgo/v2/ginkgo --fail-fast -v -r ./...