package views

import (
	_ "embed"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

//go:embed login.html
var loginHTML []byte

func RenderLogin(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/html")
	return c.Status(http.StatusUnauthorized).Send(loginHTML)
}
