package agent_test

import (
	"net/url"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAgent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Agent test suite")
}

var (
	testModel      = os.Getenv("LOCALAGI_MODEL")
	apiURL         = os.Getenv("LOCALAI_API_URL")
	apiKey         = os.Getenv("LOCALAI_API_KEY")
	useRealLocalAI bool
	clientTimeout  = "10m"
)

func isValidURL(u string) bool {
	parsed, err := url.ParseRequestURI(u)
	return err == nil && parsed.Scheme != "" && parsed.Host != ""
}

func init() {
	useRealLocalAI = isValidURL(apiURL) && apiURL != "" && testModel != ""
}
