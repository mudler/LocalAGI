package main

import "fmt"

func chatDiv(content string, color string) string {
	return fmt.Sprintf(`<div class="p-2 my-2 rounded bg-%s-600">%s</div>`, color, htmlIfy(content))
}

func loader() string {
	return `<div class="loader"></div>`
}

func inputMessageDisabled(disabled bool) string {
	if disabled {
		return `<script> document.getElementById('inputMessage').disabled = true;</script>`
	}

	return `<script> document.getElementById('inputMessage').disabled = false;</script>`
}
