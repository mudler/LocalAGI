package webui

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestWebUI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "WebUI Suite")
}
