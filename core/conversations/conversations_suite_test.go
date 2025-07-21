package conversations_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConversations(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Conversations test suite")
}
