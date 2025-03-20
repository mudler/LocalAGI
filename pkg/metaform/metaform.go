package metaform

// Option represents a selectable option for FieldOption type
type Option struct {
  Value string `json:"value"`
  Label string `json:"label"`
}

type FieldKind string

const (
  FieldString FieldKind = "string"
  FieldNumber FieldKind = "number"
  FieldOptions FieldKind = "options"
)

type Field struct {
  Kind FieldKind `json:"kind"`
  Name string `json:"name"`
  Label string `json:"label"`
  Required bool `json:"required"`
  Placeholder string `json:"placeholder,omitempty"`
  Options []Option `json:"options,omitempty"`
}

type Form struct {
  Fields []Field
}
