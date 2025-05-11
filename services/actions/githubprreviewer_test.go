package actions_test

import (
	"context"
	"os"

	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/services/actions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GithubPRReviewer", func() {
	var (
		reviewer *actions.GithubPRReviewer
		ctx      context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()

		// Check for required environment variables
		token := os.Getenv("GITHUB_TOKEN")
		repo := os.Getenv("TEST_REPOSITORY")
		owner := os.Getenv("TEST_OWNER")
		prNumber := os.Getenv("TEST_PR_NUMBER")

		// Skip tests if any required environment variable is missing
		if token == "" || repo == "" || owner == "" || prNumber == "" {
			Skip("Skipping GitHub PR reviewer tests: required environment variables not set")
		}

		config := map[string]string{
			"token":            token,
			"repository":       repo,
			"owner":            owner,
			"customActionName": "test_review_github_pr",
		}

		reviewer = actions.NewGithubPRReviewer(config)
	})

	Describe("Reviewing a PR", func() {
		It("should successfully submit a review with comments", func() {
			prNumber := os.Getenv("TEST_PR_NUMBER")
			Expect(prNumber).NotTo(BeEmpty())

			params := types.ActionParams{
				"pr_number":      prNumber,
				"review_comment": "Test review comment from integration test",
				"review_action":  "COMMENT",
				"comments": []map[string]interface{}{
					{
						"file":    "README.md",
						"line":    1,
						"comment": "Test line comment from integration test",
					},
				},
			}

			result, err := reviewer.Run(ctx, nil, params)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Result).To(ContainSubstring("reviewed successfully"))
		})

		It("should handle invalid PR number", func() {
			params := types.ActionParams{
				"pr_number":      999999,
				"review_comment": "Test review comment",
				"review_action":  "COMMENT",
			}

			result, err := reviewer.Run(ctx, nil, params)
			Expect(err).To(HaveOccurred())
			Expect(result.Result).To(ContainSubstring("not found"))
		})

		It("should handle invalid review action", func() {
			prNumber := os.Getenv("TEST_PR_NUMBER")
			Expect(prNumber).NotTo(BeEmpty())

			params := types.ActionParams{
				"pr_number":      prNumber,
				"review_comment": "Test review comment",
				"review_action":  "INVALID_ACTION",
			}

			_, err := reviewer.Run(ctx, nil, params)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Action Definition", func() {
		It("should return correct action definition", func() {
			def := reviewer.Definition()
			Expect(def.Name).To(Equal(types.ActionDefinitionName("test_review_github_pr")))
			Expect(def.Description).To(ContainSubstring("Review a GitHub pull request"))
			Expect(def.Properties).To(HaveKey("pr_number"))
			Expect(def.Properties).To(HaveKey("review_action"))
			Expect(def.Properties).To(HaveKey("comments"))
		})
	})
})
