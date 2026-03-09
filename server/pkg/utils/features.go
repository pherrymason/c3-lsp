package utils

import (
	"sort"
	"sync"
)

var (
	featureFlagsMu      sync.RWMutex
	runtimeFeatureFlags = map[string]bool{}
)

func getFeatureFlags() map[string]bool {
	return map[string]bool{
		"SIZE_ON_HOVER": false,
		"USE_SEARCH_V2": false,
	}
}

func SetRuntimeFeatureFlags(flags map[string]bool) {
	next := map[string]bool{}
	for key, enabled := range flags {
		if enabled {
			next[key] = true
		}
	}

	featureFlagsMu.Lock()
	runtimeFeatureFlags = next
	featureFlagsMu.Unlock()
}

func EnabledFeatureFlags() []string {
	defaults := getFeatureFlags()
	enabled := map[string]struct{}{}

	for name, isEnabled := range defaults {
		if isEnabled {
			enabled[name] = struct{}{}
		}
	}

	featureFlagsMu.RLock()
	for name, isEnabled := range runtimeFeatureFlags {
		if isEnabled {
			enabled[name] = struct{}{}
		}
	}
	featureFlagsMu.RUnlock()

	flags := make([]string, 0, len(enabled))
	for name := range enabled {
		flags = append(flags, name)
	}
	sort.Strings(flags)

	return flags
}

// Function to check if a specific feature is enabled
func IsFeatureEnabled(feature string) bool {
	featureFlagsMu.RLock()
	enabled, exists := runtimeFeatureFlags[feature]
	featureFlagsMu.RUnlock()
	if exists {
		return enabled
	}

	flags := getFeatureFlags()
	if enabled, exists := flags[feature]; exists {
		return enabled
	}
	// Default behavior if feature flag is not found
	return false
}
