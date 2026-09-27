package scheduler_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestScheduler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Scheduler Suite")
}
