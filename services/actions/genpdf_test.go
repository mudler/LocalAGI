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

var _ = Describe("GenPDFAction", func() {
	var (
		tmpDir      string
		action      *actions.GenPDFAction
		ctx         context.Context
		sharedState *types.AgentSharedState
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "genpdf_test_*")
		Expect(err).ToNot(HaveOccurred())

		action = actions.NewGenPDF(map[string]string{
			"outputDir": tmpDir,
		})

		ctx = context.Background()
		sharedState = &types.AgentSharedState{}
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	It("generates PDF with title and content", func() {
		result, err := action.Run(ctx, sharedState, types.ActionParams{
			"title":   "Test Document",
			"content": "This is test content for the PDF.",
		})

		Expect(err).ToNot(HaveOccurred())
		Expect(result.Result).To(ContainSubstring("PDF generated and saved to:"))
		Expect(result.Metadata).To(HaveKey(actions.MetadataPDFs))

		paths := result.Metadata[actions.MetadataPDFs].([]string)
		Expect(paths).To(HaveLen(1))
		Expect(paths[0]).To(BeAnExistingFile())
	})

	It("requires content parameter", func() {
		_, err := action.Run(ctx, sharedState, types.ActionParams{
			"title": "Test",
		})

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("content is required"))
	})

	It("uses custom filename when provided", func() {
		result, err := action.Run(ctx, sharedState, types.ActionParams{
			"content":  "Test content",
			"filename": "custom_name",
		})

		Expect(err).ToNot(HaveOccurred())
		paths := result.Metadata[actions.MetadataPDFs].([]string)
		Expect(filepath.Base(paths[0])).To(Equal("custom_name.pdf"))
	})

	It("generates PDF with content only (no title)", func() {
		result, err := action.Run(ctx, sharedState, types.ActionParams{
			"content": "Just some content without a title.",
		})

		Expect(err).ToNot(HaveOccurred())
		Expect(result.Result).To(ContainSubstring("PDF generated and saved to:"))
		paths := result.Metadata[actions.MetadataPDFs].([]string)
		Expect(paths).To(HaveLen(1))
		Expect(paths[0]).To(BeAnExistingFile())
	})

	It("automatically adds .pdf extension if missing", func() {
		result, err := action.Run(ctx, sharedState, types.ActionParams{
			"content":  "Test content",
			"filename": "my_document",
		})

		Expect(err).ToNot(HaveOccurred())
		paths := result.Metadata[actions.MetadataPDFs].([]string)
		Expect(filepath.Base(paths[0])).To(Equal("my_document.pdf"))
	})

	It("does not double-add .pdf extension", func() {
		result, err := action.Run(ctx, sharedState, types.ActionParams{
			"content":  "Test content",
			"filename": "document.pdf",
		})

		Expect(err).ToNot(HaveOccurred())
		paths := result.Metadata[actions.MetadataPDFs].([]string)
		Expect(filepath.Base(paths[0])).To(Equal("document.pdf"))
	})

	It("requires outputDir to be configured", func() {
		actionNoDir := actions.NewGenPDF(map[string]string{})
		_, err := actionNoDir.Run(ctx, sharedState, types.ActionParams{
			"content": "Test content",
		})

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("outputDir is required"))
	})

	It("cleans output directory on start if cleanOnStart is enabled", func() {
		// Create a test file in the directory
		testFile := filepath.Join(tmpDir, "test.txt")
		err := os.WriteFile(testFile, []byte("test"), 0644)
		Expect(err).ToNot(HaveOccurred())
		Expect(testFile).To(BeAnExistingFile())

		// Create a new action with cleanOnStart enabled
		_ = actions.NewGenPDF(map[string]string{
			"outputDir":    tmpDir,
			"cleanOnStart": "true",
		})

		// The test file should be deleted
		Expect(testFile).ToNot(BeAnExistingFile())
	})

	It("does not clean output directory if cleanOnStart is disabled", func() {
		// Create a test file in the directory
		testFile := filepath.Join(tmpDir, "test.txt")
		err := os.WriteFile(testFile, []byte("test"), 0644)
		Expect(err).ToNot(HaveOccurred())
		Expect(testFile).To(BeAnExistingFile())

		// Create a new action with cleanOnStart disabled (default)
		_ = actions.NewGenPDF(map[string]string{
			"outputDir": tmpDir,
		})

		// The test file should still exist
		Expect(testFile).To(BeAnExistingFile())
	})

	It("prevents path traversal in filename", func() {
		result, err := action.Run(ctx, sharedState, types.ActionParams{
			"content":  "Test content",
			"filename": "../../../etc/passwd",
		})

		Expect(err).ToNot(HaveOccurred())
		paths := result.Metadata[actions.MetadataPDFs].([]string)
		// Should only use the base filename, not the path
		Expect(filepath.Base(paths[0])).To(Equal("passwd.pdf"))
		// Should be in the tmpDir, not in /etc
		Expect(filepath.Dir(paths[0])).To(Equal(tmpDir))
	})

	It("generates PDF with markdown content and renders structure", func() {
		content := "# Section\n\n**Bold** and *italic* and `code`.\n\n- Item one\n- Item two"
		result, err := action.Run(ctx, sharedState, types.ActionParams{
			"content": content,
		})

		Expect(err).ToNot(HaveOccurred())
		Expect(result.Result).To(ContainSubstring("PDF generated and saved to:"))
		paths := result.Metadata[actions.MetadataPDFs].([]string)
		Expect(paths).To(HaveLen(1))
		Expect(paths[0]).To(BeAnExistingFile())
		info, err := os.Stat(paths[0])
		Expect(err).ToNot(HaveOccurred())
		Expect(info.Size()).To(BeNumerically(">", 0))
	})

	It("generates PDF with special characters", func() {
		content := "Café, \"quotes\", 2–3"
		result, err := action.Run(ctx, sharedState, types.ActionParams{
			"content": content,
		})

		Expect(err).ToNot(HaveOccurred())
		Expect(result.Result).To(ContainSubstring("PDF generated and saved to:"))
		paths := result.Metadata[actions.MetadataPDFs].([]string)
		Expect(paths).To(HaveLen(1))
		Expect(paths[0]).To(BeAnExistingFile())
		info, err := os.Stat(paths[0])
		Expect(err).ToNot(HaveOccurred())
		Expect(info.Size()).To(BeNumerically(">", 0))
	})

	It("generates PDF with markdown table", func() {
		content := "| A | B |\n|---|---|\n| 1 | 2 |\n| 3 | 4 |"
		result, err := action.Run(ctx, sharedState, types.ActionParams{
			"content": content,
		})

		Expect(err).ToNot(HaveOccurred())
		Expect(result.Result).To(ContainSubstring("PDF generated and saved to:"))
		paths := result.Metadata[actions.MetadataPDFs].([]string)
		Expect(paths).To(HaveLen(1))
		Expect(paths[0]).To(BeAnExistingFile())
		info, err := os.Stat(paths[0])
		Expect(err).ToNot(HaveOccurred())
		Expect(info.Size()).To(BeNumerically(">", 0))
	})
})
