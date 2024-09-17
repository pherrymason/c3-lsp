package utils

func getFeatureFlags() map[string]bool {
	return map[string]bool{
		"SIZE_ON_HOVER": false,
	}
}

// Function to check if a specific feature is enabled
func IsFeatureEnabled(feature string) bool {
	flags := getFeatureFlags()
	if enabled, exists := flags[feature]; exists {
		return enabled
	}
	// Default behavior if feature flag is not found
	return false
}
