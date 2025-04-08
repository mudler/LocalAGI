package config

type FieldType string

const (
	FieldTypeNumber   FieldType = "number"
	FieldTypeText     FieldType = "text"
	FieldTypeTextarea FieldType = "textarea"
	FieldTypeCheckbox FieldType = "checkbox"
	FieldTypeSelect   FieldType = "select"
)

type Tags struct {
	Section string `json:"section,omitempty"`
}

type FieldOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type Field struct {
	Name         string        `json:"name"`
	Type         FieldType     `json:"type"`
	Label        string        `json:"label"`
	DefaultValue any           `json:"defaultValue"`
	Placeholder  string        `json:"placeholder,omitempty"`
	HelpText     string        `json:"helpText,omitempty"`
	Required     bool          `json:"required,omitempty"`
	Disabled     bool          `json:"disabled,omitempty"`
	Options      []FieldOption `json:"options,omitempty"`
	Min          float32       `json:"min,omitempty"`
	Max          float32       `json:"max,omitempty"`
	Step         float32       `json:"step,omitempty"`
	Tags         Tags          `json:"tags,omitempty"`
}

type FieldGroup struct {
	Name   string  `json:"name"`
	Label  string  `json:"label"`
	Fields []Field `json:"fields"`
}
