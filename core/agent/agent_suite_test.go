package agent_test

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAgent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Agent test suite")
}

var testModel = os.Getenv("LOCALAGENT_MODEL")
var apiModel = os.Getenv("API_MODEL")

func init() {
	if testModel == "" {
		testModel = "hermes-2-pro-mistral"
	}
	if apiModel == "" {
		apiModel = "http://192.168.68.113:8080"
	}
}
