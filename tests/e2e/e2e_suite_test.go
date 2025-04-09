package e2e_test

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E test suite")
}

var apiURL = os.Getenv("LOCALAI_API_URL")
var localagiURL = os.Getenv("LOCALAGI_API_URL")
