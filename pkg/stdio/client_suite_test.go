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
