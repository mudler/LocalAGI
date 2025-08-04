package services

import (
	"fmt"
	"os"

	"github.com/mudler/LocalAGI/pkg/xlog"
)

func memoryPath(agentName string, actionsConfigs map[string]string) string {
	// Compose memory file path based on stateDir and agentName, using a subdirectory
	memoryFilePath := "memory.json"
	if actionsConfigs != nil {
		if stateDir, ok := actionsConfigs[ConfigStateDir]; ok && stateDir != "" {
			memoryDir := fmt.Sprintf("%s/memory", stateDir)
			err := os.MkdirAll(memoryDir, 0755) // ensure the directory exists
			if err != nil {
				xlog.Error("Error creating memory directory", "error", err)
				return memoryFilePath
			}
			memoryFilePath = fmt.Sprintf("%s/%s.json", memoryDir, agentName)
		} else {
			memoryFilePath = fmt.Sprintf("%s.memory.json", agentName)
		}
	}

	return memoryFilePath
}
