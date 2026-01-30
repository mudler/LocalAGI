package services

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mudler/xlog"
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

// memoryIndexPath returns the directory path for the Bleve index (used by memory actions).
func memoryIndexPath(agentName string, actionsConfigs map[string]string) string {
	indexPath := "memory.bleve"
	if actionsConfigs != nil {
		if stateDir, ok := actionsConfigs[ConfigStateDir]; ok && stateDir != "" {
			memoryDir := fmt.Sprintf("%s/memory", stateDir)
			if err := os.MkdirAll(memoryDir, 0755); err != nil {
				xlog.Error("Error creating memory directory", "error", err)
				return indexPath
			}
			indexPath = filepath.Join(memoryDir, agentName+".bleve")
		} else {
			indexPath = agentName + ".memory.bleve"
		}
	}
	if dir := filepath.Dir(indexPath); dir != "." {
		os.MkdirAll(dir, 0755)
	}
	return indexPath
}
