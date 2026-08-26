package conversations_test

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/mudler/LocalAGI/core/conversations"
)

var _ = Describe("Pruner", func() {
	var dir string

	BeforeEach(func() {
		dir = GinkgoT().TempDir()
	})

	writeAged := func(agent string, age time.Duration) string {
		name := agent + "-" + time.Now().Add(-age).Format(layout) + ".json"
		Expect(os.WriteFile(filepath.Join(dir, name), []byte("[]"), 0644)).To(Succeed())
		return name
	}

	exists := func(name string) bool {
		_, err := os.Stat(filepath.Join(dir, name))
		return err == nil
	}

	It("sweeps once immediately on Start without waiting for the first tick", func() {
		stale := writeAged("Lena", 48*time.Hour)

		p := conversations.NewPruner(dir, conversations.RetentionPolicy{
			MaxAge: 24 * time.Hour,
		}, time.Hour)
		p.Start()
		DeferCleanup(p.Stop)

		Eventually(func() bool { return exists(stale) }, "2s", "20ms").Should(BeFalse())
	})

	It("keeps sweeping on each tick", func() {
		p := conversations.NewPruner(dir, conversations.RetentionPolicy{
			MaxAge: 24 * time.Hour,
		}, 20*time.Millisecond)
		p.Start()
		DeferCleanup(p.Stop)

		// Land a stale dump after the initial sweep has already run.
		time.Sleep(50 * time.Millisecond)
		late := writeAged("Lena", 48*time.Hour)

		Eventually(func() bool { return exists(late) }, "2s", "20ms").Should(BeFalse())
	})

	It("does not remove anything when the policy is disabled", func() {
		old := writeAged("Lena", 10000*time.Hour)

		p := conversations.NewPruner(dir, conversations.RetentionPolicy{}, 20*time.Millisecond)
		p.Start()
		DeferCleanup(p.Stop)

		Consistently(func() bool { return exists(old) }, "300ms", "20ms").Should(BeTrue())
	})

	It("stops sweeping after Stop", func() {
		p := conversations.NewPruner(dir, conversations.RetentionPolicy{
			MaxAge: 24 * time.Hour,
		}, 20*time.Millisecond)
		p.Start()
		p.Stop()

		survivor := writeAged("Lena", 48*time.Hour)
		Consistently(func() bool { return exists(survivor) }, "300ms", "20ms").Should(BeTrue())
	})

	It("tolerates Stop being called more than once", func() {
		p := conversations.NewPruner(dir, conversations.RetentionPolicy{
			MaxAge: 24 * time.Hour,
		}, 20*time.Millisecond)
		p.Start()
		p.Stop()

		Expect(p.Stop).ToNot(Panic())
	})

	It("tolerates Stop without Start", func() {
		p := conversations.NewPruner(dir, conversations.RetentionPolicy{
			MaxAge: 24 * time.Hour,
		}, 20*time.Millisecond)

		Expect(p.Stop).ToNot(Panic())
	})
})

var _ = Describe("Pruner without a directory", func() {
	// A pool created with conversation logging disabled has no directory to
	// sweep; the sweeper must stay dormant rather than error on every tick.
	It("never starts when the directory is empty", func() {
		p := conversations.NewPruner("", conversations.RetentionPolicy{
			MaxAge: 24 * time.Hour,
		}, 20*time.Millisecond)

		Expect(p.Start).ToNot(Panic())
		Expect(p.Running()).To(BeFalse())
		p.Stop()
	})

	It("reports running while a configured sweeper is active", func() {
		p := conversations.NewPruner(GinkgoT().TempDir(), conversations.RetentionPolicy{
			MaxAge: 24 * time.Hour,
		}, 20*time.Millisecond)
		p.Start()
		DeferCleanup(p.Stop)

		Expect(p.Running()).To(BeTrue())
	})
})
