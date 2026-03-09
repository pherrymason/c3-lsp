package project_state

type InvalidationScope struct {
	ChangedModules          []string
	SignatureChangedModules []string
	ImpactedModules         []string
}
