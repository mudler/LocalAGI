package colony

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"strings"
	"time"

	fiber "github.com/gofiber/fiber/v2"
)

func verifyHiveSignature(sigHeader string, body []byte) bool {
	secret := os.Getenv("HIVE_JWT_SECRET")
	if secret == "" {
		return true // permissive dev mode
	}
	if !strings.HasPrefix(sigHeader, "sha256=") {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedHex := hex.EncodeToString(mac.Sum(nil))
	actualHex := sigHeader[7:] // strip "sha256=" prefix
	expectedBytes, err1 := hex.DecodeString(expectedHex)
	actualBytes, err2 := hex.DecodeString(actualHex)
	if err1 != nil || err2 != nil {
		return false
	}
	return hmac.Equal(actualBytes, expectedBytes)
}

var colonyInfo = fiber.Map{
	"colony_id":   "localagi",
	"colony_name": "LocalAGI",
	"role":        "colony",
	"archetype":   "body",
	"layer":       3,
	"entity":      "BODY (The Swarm)",
	"guilds":      []string{"swarm", "workflow", "agent"},
	"hive":        "sovereign-hive",
	"queen":       "https://github.com/TehutiRaEl/-sovereign-hive-meta",
	"version":     "1.0.0",
}

func RegisterFiberRoutes(app *fiber.App) {
	app.Get("/colony/info", handleInfo)
	app.Get("/colony/health", handleHealth)
	app.Get("/colony/agents", handleAgents)
	app.Post("/colony/events", handleEvents)
	app.Get("/colony/manifest", handleManifest)
	app.Get("/colony/capabilities", handleCapabilities)
}

var startTime = time.Now()

func soulHash() string {
	data, err := os.ReadFile("soul.md")
	if err != nil {
		return "none"
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])[:16]
}

// handleCapabilities mirrors THEHIVE's /colony/capabilities shape:
// colony.json identity + live status/uptime/soul hash + endpoint pointers.
func handleCapabilities(c *fiber.Ctx) error {
	caps := fiber.Map{}
	for k, v := range colonyInfo {
		caps[k] = v
	}
	if data, err := os.ReadFile("colony.json"); err == nil {
		var identity map[string]interface{}
		if json.Unmarshal(data, &identity) == nil {
			for k, v := range identity {
				caps[k] = v
			}
		}
	}
	// Sonnet's structured capability catalog (PR #6) — colony.json identity
	// may override it by carrying its own "capabilities" key
	if _, ok := caps["capabilities"]; !ok {
		caps["capabilities"] = []fiber.Map{
			{"name": "task_execution", "description": "Execute multi-step agent tasks autonomously"},
			{"name": "skill_management", "description": "Load, register, and invoke skill modules"},
			{"name": "knowledge_integration", "description": "Query and update the local knowledge base"},
			{"name": "agent_orchestration", "description": "Coordinate multiple sub-agents toward a goal"},
		}
	}
	caps["protocol"] = "sovereign-hive-v11"
	caps["status"] = "healthy"
	caps["uptime_s"] = float64(int(time.Since(startTime).Seconds()*10)) / 10
	caps["soul_md_hash"] = soulHash()
	caps["health_endpoint"] = "/colony/health"
	caps["capabilities_endpoint"] = "/colony/capabilities"
	return c.JSON(caps)
}

func handleInfo(c *fiber.Ctx) error {
	return c.JSON(colonyInfo)
}

func handleHealth(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":    "ok",
		"colony_id": "localagi",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func handleAgents(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"colony_id": "localagi",
		"agents":    []string{"task-executor", "skill-manager", "knowledge-agent", "integration-agent"},
	})
}

func handleEvents(c *fiber.Ctx) error {
	body := c.Body()
	if !verifyHiveSignature(c.Get("X-Hive-Signature"), body) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":     "Invalid hive signature",
			"colony_id": "localagi",
		})
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON"})
	}
	h := fnv.New32a()
	b, _ := json.Marshal(payload)
	h.Write(b)
	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"accepted":  true,
		"event_id":  fmt.Sprintf("%08x", h.Sum32()),
		"colony_id": "localagi",
	})
}

func handleManifest(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"colony_id": "localagi",
		"endpoints": fiber.Map{
			"info":         "/colony/info",
			"health":       "/colony/health",
			"agents":       "/colony/agents",
			"events":       "/colony/events",
			"manifest":     "/colony/manifest",
			"capabilities": "/colony/capabilities",
		},
		"constitution": "https://raw.githubusercontent.com/TehutiRaEl/THEHIVE/main/soul.md",
	})
}
