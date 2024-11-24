package featureflags

var features = map[string]bool{
	"UseGeneratedAST": false,
}

func IsActive(feature string) bool {
	return features[feature]
}
