package featureflags

const UseGeneratedAST = "UseGeneratedAST"

var features = map[string]bool{
	UseGeneratedAST: true,
}

func IsActive(feature string) bool {
	return features[feature]
}
