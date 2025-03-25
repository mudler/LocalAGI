package xstrings_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestXStrings(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "XStrings test suite")
}
