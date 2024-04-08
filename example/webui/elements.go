package main

import "fmt"

// TODO: switch to https://github.com/chasefleming/elem-go

func chatDiv(content string, color string) string {
	return fmt.Sprintf(`<div class="p-2 my-2 rounded bg-%s-600">%s</div>`, color, htmlIfy(content))
}

func loader() string {
	return `<div class="loader"></div>`
}

func disabledElement(id string, disabled bool) string {
	if disabled {
		return `<script> document.getElementById('` + id + `').disabled = true;</script>`
	}

	return `<script> document.getElementById('` + id + `').disabled = false;</script>`
}
