package main

import "fmt"

func chatDiv(content string, color string) string {
	return fmt.Sprintf(`<div class="p-2 my-2 rounded bg-%s-100">%s</div>`, color, htmlIfy(content))
}
