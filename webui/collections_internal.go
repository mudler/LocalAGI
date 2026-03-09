package webui

import (
	"github.com/mudler/LocalAGI/core/agent"
	"github.com/mudler/LocalAGI/core/state"
	"github.com/mudler/LocalAGI/webui/collections"
)

// CollectionsRAGProviderFromState delegates to the collections sub-package.
func CollectionsRAGProviderFromState(cs *CollectionsState) func(collectionName string) (agent.RAGDB, state.KBCompactionClient, bool) {
	return collections.RAGProviderFromState(cs)
}

// CollectionsRAGProvider returns a provider that the pool can use when no LocalRAG URL is set.
func (app *App) CollectionsRAGProvider() func(collectionName string) (agent.RAGDB, state.KBCompactionClient, bool) {
	return CollectionsRAGProviderFromState(app.collectionsState)
}
