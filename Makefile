GOCMD=go

tests:
	$(GOCMD) run github.com/onsi/ginkgo/v2/ginkgo --flake-attempts 5 --fail-fast -v -r ./...