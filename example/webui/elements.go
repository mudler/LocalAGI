package main

import (
	"fmt"
	"strings"

	elem "github.com/chasefleming/elem-go"
	"github.com/chasefleming/elem-go/attrs"
)

func chatDiv(content string, color string) string {
	div := elem.Div(attrs.Props{
		//	attrs.ID:    "container",
		attrs.Class: fmt.Sprintf("p-2 my-2 rounded bg-%s-600", color),
	},
		elem.Raw(htmlIfy(content)),
	)
	return div.Render()
}

func loader() string {
	return elem.Div(attrs.Props{
		attrs.Class: "loader",
	}).Render()
}

func disabledElement(id string, disabled bool) string {
	return elem.Script(nil,
		elem.If(disabled,
			elem.Raw(`document.getElementById('`+id+`').disabled = true`),
			elem.Raw(`document.getElementById('`+id+`').disabled = false`),
		)).Render()
}

func htmlIfy(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", "<br>")
	return s
}
