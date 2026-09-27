package webui

import (
	"github.com/mudler/LocalAGI/webui/collections"
)

// Re-export types from the collections sub-package so existing webui code continues to work.
type CollectionSearchResult = collections.SearchResult
type CollectionSourceInfo = collections.SourceInfo
type CollectionsBackend = collections.Backend
type CollectionsState = collections.State
type CollectionList = collections.CollectionList
