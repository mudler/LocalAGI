package stdio

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSTDIOTransport(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "STDIOTransport test suite")
}

var baseURL string

func init() {
	baseURL = os.Getenv("STDIO_SERVER_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
}

var _ = AfterSuite(func() {
	client := NewClient(baseURL)
	client.StopGroup("test-group")
})
