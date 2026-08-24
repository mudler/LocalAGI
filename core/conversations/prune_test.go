package conversations_test

import (
	"os"
	"path/filepath"
	"sort"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/mudler/LocalAGI/core/conversations"
)

// layout mirrors the one saveConversation uses to name files.
const layout = "2006-01-02-15-04-05"

var _ = Describe("Prune", func() {
	var dir string
	var now time.Time

	BeforeEach(func() {
		dir = GinkgoT().TempDir()
		now = time.Date(2026, 8, 24, 12, 0, 0, 0, time.Local)
	})

	// write creates a conversation dump named after agent, aged `age` before now.
	write := func(agent string, age time.Duration) string {
		name := agent + "-" + now.Add(-age).Format(layout) + ".json"
		Expect(os.WriteFile(filepath.Join(dir, name), []byte("[]"), 0644)).To(Succeed())
		return name
	}

	remaining := func() []string {
		entries, err := os.ReadDir(dir)
		Expect(err).ToNot(HaveOccurred())
		names := []string{}
		for _, e := range entries {
			names = append(names, e.Name())
		}
		sort.Strings(names)
		return names
	}

	Context("age retention", func() {
		It("removes files older than MaxAge and keeps newer ones", func() {
			old := write("Lena", 48*time.Hour)
			recent := write("Lena", 1*time.Hour)

			removed, err := conversations.Prune(dir, conversations.RetentionPolicy{
				MaxAge: 24 * time.Hour,
			}, now)

			Expect(err).ToNot(HaveOccurred())
			Expect(removed).To(Equal(1))
			Expect(remaining()).To(Equal([]string{recent}))
			Expect(remaining()).ToNot(ContainElement(old))
		})

		It("keeps every file when MaxAge is zero", func() {
			write("Lena", 10000*time.Hour)

			removed, err := conversations.Prune(dir, conversations.RetentionPolicy{
				MaxAge: 0,
			}, now)

			Expect(err).ToNot(HaveOccurred())
			Expect(removed).To(Equal(0))
			Expect(remaining()).To(HaveLen(1))
		})
	})

	Context("per-agent count retention", func() {
		It("keeps only the newest MaxPerAgent files for an agent", func() {
			write("Lena", 4*time.Hour)
			write("Lena", 3*time.Hour)
			second := write("Lena", 2*time.Hour)
			newest := write("Lena", 1*time.Hour)

			removed, err := conversations.Prune(dir, conversations.RetentionPolicy{
				MaxPerAgent: 2,
			}, now)

			Expect(err).ToNot(HaveOccurred())
			Expect(removed).To(Equal(2))
			Expect(remaining()).To(ConsistOf(second, newest))
		})

		It("counts each agent separately", func() {
			lena := write("Lena", 1*time.Hour)
			jordan := write("Jordan", 1*time.Hour)

			removed, err := conversations.Prune(dir, conversations.RetentionPolicy{
				MaxPerAgent: 1,
			}, now)

			Expect(err).ToNot(HaveOccurred())
			Expect(removed).To(Equal(0))
			Expect(remaining()).To(ConsistOf(lena, jordan))
		})

		It("treats a decision- prefixed agent as its own bucket", func() {
			conv := write("Lena", 1*time.Hour)
			decisionOld := write("decision-Lena", 3*time.Hour)
			decisionNew := write("decision-Lena", 2*time.Hour)

			removed, err := conversations.Prune(dir, conversations.RetentionPolicy{
				MaxPerAgent: 1,
			}, now)

			Expect(err).ToNot(HaveOccurred())
			Expect(removed).To(Equal(1))
			Expect(remaining()).To(ConsistOf(conv, decisionNew))
			Expect(remaining()).ToNot(ContainElement(decisionOld))
		})

		It("keeps every file when MaxPerAgent is zero", func() {
			write("Lena", 3*time.Hour)
			write("Lena", 2*time.Hour)
			write("Lena", 1*time.Hour)

			removed, err := conversations.Prune(dir, conversations.RetentionPolicy{
				MaxPerAgent: 0,
			}, now)

			Expect(err).ToNot(HaveOccurred())
			Expect(removed).To(Equal(0))
			Expect(remaining()).To(HaveLen(3))
		})
	})

	Context("both policies together", func() {
		It("removes a file that violates either limit", func() {
			write("Lena", 100*time.Hour) // too old
			write("Lena", 3*time.Hour)   // over count
			kept := write("Lena", 1*time.Hour)

			removed, err := conversations.Prune(dir, conversations.RetentionPolicy{
				MaxAge:      24 * time.Hour,
				MaxPerAgent: 1,
			}, now)

			Expect(err).ToNot(HaveOccurred())
			Expect(removed).To(Equal(2))
			Expect(remaining()).To(Equal([]string{kept}))
		})
	})

	Context("safety", func() {
		It("never touches files that do not match the conversation naming scheme", func() {
			foreign := "pool.json"
			Expect(os.WriteFile(filepath.Join(dir, foreign), []byte("{}"), 0644)).To(Succeed())
			noTimestamp := "Lena-notadate.json"
			Expect(os.WriteFile(filepath.Join(dir, noTimestamp), []byte("[]"), 0644)).To(Succeed())

			removed, err := conversations.Prune(dir, conversations.RetentionPolicy{
				MaxAge:      time.Nanosecond,
				MaxPerAgent: 1,
			}, now)

			Expect(err).ToNot(HaveOccurred())
			Expect(removed).To(Equal(0))
			Expect(remaining()).To(ConsistOf(foreign, noTimestamp))
		})

		It("ignores subdirectories", func() {
			Expect(os.Mkdir(filepath.Join(dir, "nested"), 0755)).To(Succeed())

			removed, err := conversations.Prune(dir, conversations.RetentionPolicy{
				MaxAge: time.Nanosecond,
			}, now)

			Expect(err).ToNot(HaveOccurred())
			Expect(removed).To(Equal(0))
			Expect(remaining()).To(ConsistOf("nested"))
		})

		It("returns an error when the directory cannot be read", func() {
			_, err := conversations.Prune(filepath.Join(dir, "missing"), conversations.RetentionPolicy{
				MaxAge: time.Hour,
			}, now)

			Expect(err).To(HaveOccurred())
		})
	})
})
