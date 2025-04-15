package actions_test

import (
	"context"
	"os"
	"strings"

	"github.com/mudler/LocalAGI/services/actions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GithubRepositoryGetAllContent", func() {
	var (
		action *actions.GithubRepositoryGetAllContent
		ctx    context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()

		// Check for required environment variables
		token := os.Getenv("GITHUB_TOKEN")
		repo := os.Getenv("TEST_REPOSITORY")
		owner := os.Getenv("TEST_OWNER")

		// Skip tests if any required environment variable is missing
		if token == "" || repo == "" || owner == "" {
			Skip("Skipping GitHub repository get all content tests: required environment variables not set")
		}

		config := map[string]string{
			"token":            token,
			"repository":       repo,
			"owner":            owner,
			"customActionName": "test_get_all_content",
		}

		action = actions.NewGithubRepositoryGetAllContent(config)
	})

	Describe("Getting repository content", func() {
		It("should successfully get content from root directory with proper file markers", func() {
			params := map[string]interface{}{
				"path": ".",
			}

			result, err := action.Run(ctx, params)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Result).NotTo(BeEmpty())

			// Verify file markers
			Expect(result.Result).To(ContainSubstring("--- START FILE:"))
			Expect(result.Result).To(ContainSubstring("--- END FILE:"))

			// Verify markers are properly paired
			startCount := strings.Count(result.Result, "--- START FILE:")
			endCount := strings.Count(result.Result, "--- END FILE:")
			Expect(startCount).To(Equal(endCount), "Number of start and end markers should match")
		})

		It("should handle non-existent path", func() {
			params := map[string]interface{}{
				"path": "non-existent-path",
			}

			_, err := action.Run(ctx, params)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Action Definition", func() {
		It("should return correct action definition", func() {
			def := action.Definition()
			Expect(def.Name.String()).To(Equal("test_get_all_content"))
			Expect(def.Description).To(ContainSubstring("Get all content of a GitHub repository recursively"))
			Expect(def.Properties).To(HaveKey("path"))
		})

		It("should handle custom action name", func() {
			config := map[string]string{
				"token":            "test-token",
				"customActionName": "custom_action_name",
			}
			action := actions.NewGithubRepositoryGetAllContent(config)
			def := action.Definition()
			Expect(def.Name.String()).To(Equal("custom_action_name"))
		})
	})

	Describe("Configuration", func() {
		It("should handle missing repository and owner in config", func() {
			config := map[string]string{
				"token": "test-token",
			}
			action := actions.NewGithubRepositoryGetAllContent(config)
			def := action.Definition()
			Expect(def.Properties).To(HaveKey("repository"))
			Expect(def.Properties).To(HaveKey("owner"))
		})

		It("should handle provided repository and owner in config", func() {
			config := map[string]string{
				"token":      "test-token",
				"repository": "test-repo",
				"owner":      "test-owner",
			}
			action := actions.NewGithubRepositoryGetAllContent(config)
			def := action.Definition()
			Expect(def.Properties).NotTo(HaveKey("repository"))
			Expect(def.Properties).NotTo(HaveKey("owner"))
		})
	})
})
