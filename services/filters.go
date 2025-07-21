package services

import (
	"github.com/mudler/LocalAGI/core/state"
	"github.com/mudler/LocalAGI/core/types"
	"github.com/mudler/LocalAGI/pkg/config"
	"github.com/mudler/LocalAGI/pkg/xlog"
	"github.com/mudler/LocalAGI/services/filters"
)

func Filters(a *state.AgentConfig) types.JobFilters {
	var result []types.JobFilter
	for _, f := range a.Filters {
		var filter types.JobFilter
		var err error
		switch f.Type {
		case filters.FilterRegex:
			filter, err = filters.NewRegexFilter(f.Config)
			if err != nil {
				xlog.Error("Failed to configure regex", "err", err.Error())
				continue
			}
		case filters.FilterClassifier:
			filter, err = filters.NewClassifierFilter(f.Config, a)
			if err != nil {
				xlog.Error("failed to configure classifier", "err", err.Error())
				continue
			}
		default:
			xlog.Error("Unrecognized filter type", "type", f.Type)
			continue
		}
		result = append(result, filter)
	}
	return result
}

// FiltersConfigMeta returns all filter config metas for UI.
func FiltersConfigMeta() []config.FieldGroup {
	return []config.FieldGroup{
		filters.RegexFilterConfigMeta(),
		filters.ClassifierFilterConfigMeta(),
	}
}
