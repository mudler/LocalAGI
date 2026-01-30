package actions_test

import (
	"context"
	"os"
	"path/filepath"

	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/services/actions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MemoryActions", func() {
	var (
		tmpDir  string
		indexPath string
		aAdd    *actions.AddToMemoryAction
		aList   *actions.ListMemoryAction
		aRemove *actions.RemoveFromMemoryAction
		aSearch *actions.SearchMemoryAction
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "memory_test_*")
		Expect(err).ToNot(HaveOccurred())
		indexPath = filepath.Join(tmpDir, "memory.bleve")
		aAdd, aList, aRemove, aSearch = actions.NewMemoryActions(indexPath, map[string]string{})
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	It("adds and lists entries by name", func() {
		_, err := aAdd.Run(context.TODO(), nil, types.ActionParams{"name": "foo", "content": "bar"})
		Expect(err).ToNot(HaveOccurred())
		_, err = aAdd.Run(context.TODO(), nil, types.ActionParams{"name": "baz", "content": "qux"})
		Expect(err).ToNot(HaveOccurred())
		res, err := aList.Run(context.TODO(), nil, types.ActionParams{})
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Metadata["names"]).To(ContainElements("foo", "baz"))
		Expect(res.Metadata["count"]).To(Equal(2))
	})

	It("removes by id", func() {
		addRes, _ := aAdd.Run(context.TODO(), nil, types.ActionParams{"name": "foo", "content": "bar"})
		id, ok := addRes.Metadata["id"].(string)
		Expect(ok).To(BeTrue())
		_, _ = aAdd.Run(context.TODO(), nil, types.ActionParams{"name": "baz", "content": "qux"})
		_, err := aRemove.Run(context.TODO(), nil, types.ActionParams{"id": id})
		Expect(err).ToNot(HaveOccurred())
		res, _ := aList.Run(context.TODO(), nil, types.ActionParams{})
		Expect(res.Metadata["names"]).To(ConsistOf("baz"))
	})

	It("returns error for missing id on remove", func() {
		_, _ = aAdd.Run(context.TODO(), nil, types.ActionParams{"name": "foo", "content": "bar"})
		_, err := aRemove.Run(context.TODO(), nil, types.ActionParams{})
		Expect(err).To(HaveOccurred())
	})

	It("returns error for unknown id on remove", func() {
		_, err := aRemove.Run(context.TODO(), nil, types.ActionParams{"id": "nonexistent"})
		Expect(err).To(HaveOccurred())
	})

	It("returns error for empty name and content on add", func() {
		_, err := aAdd.Run(context.TODO(), nil, types.ActionParams{"name": "", "content": ""})
		Expect(err).To(HaveOccurred())
	})

	It("search returns matching entries", func() {
		_, _ = aAdd.Run(context.TODO(), nil, types.ActionParams{"name": "meeting", "content": "discussed project X"})
		_, _ = aAdd.Run(context.TODO(), nil, types.ActionParams{"name": "lunch", "content": "ate pizza"})
		res, err := aSearch.Run(context.TODO(), nil, types.ActionParams{"query": "project"})
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Metadata["count"]).To(Equal(1))
		results, ok := res.Metadata["results"].([]actions.MemoryEntry)
		Expect(ok).To(BeTrue())
		Expect(results).To(HaveLen(1))
		Expect(results[0].Name).To(Equal("meeting"))
		Expect(results[0].Content).To(Equal("discussed project X"))
	})

	It("search returns error for empty query", func() {
		_, err := aSearch.Run(context.TODO(), nil, types.ActionParams{"query": ""})
		Expect(err).To(HaveOccurred())
	})
})
