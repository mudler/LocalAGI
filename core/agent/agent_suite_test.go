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

var testModel = os.Getenv("LOCALAGI_MODEL")
var apiURL = os.Getenv("LOCALAI_API_URL")
var apiKeyURL = os.Getenv("LOCALAI_API_KEY")

func init() {
	if testModel == "" {
		testModel = "hermes-2-pro-mistral"
	}
	if apiURL == "" {
		apiURL = "http://192.168.68.113:8080"
	}
}
