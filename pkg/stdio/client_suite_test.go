package stdio

import (
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
}

var _ = AfterSuite(func() {
	client := NewClient(baseURL)
	client.StopGroup("test-group")
})
