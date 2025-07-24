package actions_test

import (
	"context"
	"os"

	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/services/actions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MemoryActions", func() {
	var (
		tmpFile string
		aAdd    *actions.AddToMemoryAction
		aList   *actions.ListMemoryAction
		aRemove *actions.RemoveFromMemoryAction
	)

	BeforeEach(func() {
		f, err := os.CreateTemp("", "memory_test_*.json")
		Expect(err).ToNot(HaveOccurred())
		tmpFile = f.Name()
		f.Close()
		aAdd, aList, aRemove = actions.NewMemoryActions(tmpFile, map[string]string{})
	})

	AfterEach(func() {
		os.Remove(tmpFile)
	})

	It("adds and lists items", func() {
		_, err := aAdd.Run(context.TODO(), nil, types.ActionParams{"item": "foo"})
		Expect(err).ToNot(HaveOccurred())
		_, err = aAdd.Run(context.TODO(), nil, types.ActionParams{"item": "bar"})
		Expect(err).ToNot(HaveOccurred())
		res, err := aList.Run(context.TODO(), nil, types.ActionParams{})
		Expect(err).ToNot(HaveOccurred())
		Expect(res.Metadata["items"]).To(ContainElements("foo", "bar"))
	})

	It("removes by index", func() {
		_, _ = aAdd.Run(context.TODO(), nil, types.ActionParams{"item": "foo"})
		_, _ = aAdd.Run(context.TODO(), nil, types.ActionParams{"item": "bar"})
		_, err := aRemove.Run(context.TODO(), nil, types.ActionParams{"index": 0})
		Expect(err).ToNot(HaveOccurred())
		res, _ := aList.Run(context.TODO(), nil, types.ActionParams{})
		Expect(res.Metadata["items"]).To(ConsistOf("bar"))
	})

	It("removes by value", func() {
		_, _ = aAdd.Run(context.TODO(), nil, types.ActionParams{"item": "foo"})
		_, _ = aAdd.Run(context.TODO(), nil, types.ActionParams{"item": "bar"})
		_, err := aRemove.Run(context.TODO(), nil, types.ActionParams{"value": "bar"})
		Expect(err).ToNot(HaveOccurred())
		res, _ := aList.Run(context.TODO(), nil, types.ActionParams{})
		Expect(res.Metadata["items"]).To(ConsistOf("foo"))
	})

	It("returns error for out of range index", func() {
		_, _ = aAdd.Run(context.TODO(), nil, types.ActionParams{"item": "foo"})
		_, err := aRemove.Run(context.TODO(), nil, types.ActionParams{"index": 2})
		Expect(err).To(HaveOccurred())
	})

	It("returns error for value not found", func() {
		_, _ = aAdd.Run(context.TODO(), nil, types.ActionParams{"item": "foo"})
		_, err := aRemove.Run(context.TODO(), nil, types.ActionParams{"value": "bar"})
		Expect(err).To(HaveOccurred())
	})

	It("returns error for empty item", func() {
		_, err := aAdd.Run(context.TODO(), nil, types.ActionParams{"item": ""})
		Expect(err).To(HaveOccurred())
	})

	It("returns error if neither index nor value provided", func() {
		_, _ = aAdd.Run(context.TODO(), nil, types.ActionParams{"item": "foo"})
		_, err := aRemove.Run(context.TODO(), nil, types.ActionParams{})
		Expect(err).To(HaveOccurred())
	})
})
