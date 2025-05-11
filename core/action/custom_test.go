package action_test

import (
	"context"

	. "github.com/mudler/LocalAGI/core/action"
	"github.com/mudler/LocalAGI/core/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sashabaranov/go-openai/jsonschema"
)

var _ = Describe("Agent custom action", func() {
	Context("custom action", func() {
		It("initializes correctly", func() {

			testCode := `

import (
	"encoding/json"
)
type Params struct {
	Foo string
}

func Run(config map[string]interface{}) (string, map[string]interface{}, error) {

p := Params{}
b, err := json.Marshal(config)
	if err != nil {
		return "",map[string]interface{}{}, err
	}
if err := json.Unmarshal(b, &p); err != nil {
	return "",map[string]interface{}{}, err
}

return p.Foo,map[string]interface{}{}, nil
}

func Definition() map[string][]string {
return map[string][]string{
	"foo": []string{
		"string",
		"The foo value",
		},
	}
}

func RequiredFields() []string {
return []string{"foo"}
}

			`

			customAction, err := NewCustom(
				map[string]string{
					"code":        testCode,
					"name":        "test",
					"description": "A test action",
				},
				"",
			)
			Expect(err).ToNot(HaveOccurred())

			definition := customAction.Definition()
			Expect(definition).To(Equal(types.ActionDefinition{
				Properties: map[string]jsonschema.Definition{
					"foo": {
						Type:        jsonschema.String,
						Description: "The foo value",
					},
				},
				Required:    []string{"foo"},
				Name:        "test",
				Description: "A test action",
			}))

			runResult, err := customAction.Run(context.Background(), nil, types.ActionParams{
				"Foo": "bar",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(runResult.Result).To(Equal("bar"))

		})
	})
})
