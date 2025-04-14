package actions_test

import (
	"context"
	"os"

	"github.com/mudler/LocalAGI/services/actions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GithubPRCreator", func() {
	var (
		action *actions.GithubPRCreator
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
			Skip("Skipping GitHub PR creator tests: required environment variables not set")
		}

		config := map[string]string{
			"token":            token,
			"repository":       repo,
			"owner":            owner,
			"customActionName": "test_create_pr",
			"defaultBranch":    "main",
		}

		action = actions.NewGithubPRCreator(config)
	})

	Describe("Creating pull requests", func() {
		It("should successfully create a pull request with file changes", func() {
			params := map[string]interface{}{
				"branch":      "test-branch",
				"title":       "Test PR",
				"body":        "This is a test pull request",
				"base_branch": "main",
				"files": []map[string]interface{}{
					{
						"path":    "test.txt",
						"content": "This is a test file",
					},
				},
			}

			result, err := action.Run(ctx, params)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Result).To(ContainSubstring("pull request #"))
		})

		It("should handle missing required fields", func() {
			params := map[string]interface{}{
				"title": "Test PR",
				"body":  "This is a test pull request",
			}

			_, err := action.Run(ctx, params)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Action Definition", func() {
		It("should return correct action definition", func() {
			def := action.Definition()
			Expect(def.Name.String()).To(Equal("test_create_pr"))
			Expect(def.Description).To(ContainSubstring("Create a GitHub pull request with file changes"))
			Expect(def.Properties).To(HaveKey("branch"))
			Expect(def.Properties).To(HaveKey("title"))
			Expect(def.Properties).To(HaveKey("files"))
		})

		It("should handle custom action name", func() {
			config := map[string]string{
				"token":            "test-token",
				"customActionName": "custom_action_name",
			}
			action := actions.NewGithubPRCreator(config)
			def := action.Definition()
			Expect(def.Name.String()).To(Equal("custom_action_name"))
		})
	})

	Describe("Configuration", func() {
		It("should handle missing repository and owner in config", func() {
			config := map[string]string{
				"token": "test-token",
			}
			action := actions.NewGithubPRCreator(config)
			def := action.Definition()
			Expect(def.Properties).To(HaveKey("repository"))
			Expect(def.Properties).To(HaveKey("owner"))
		})

		It("should handle provided repository and owner in config", func() {
			config := map[string]string{
				"token":         "test-token",
				"repository":    "test-repo",
				"defaultBranch": "main",
				"owner":         "test-owner",
			}
			action := actions.NewGithubPRCreator(config)
			def := action.Definition()
			Expect(def.Properties).NotTo(HaveKey("repository"))
			Expect(def.Properties).NotTo(HaveKey("owner"))
		})
	})
})
