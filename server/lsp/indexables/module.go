package indexables

type Module struct {
	Name  string
	Alias string
}

type ModulePath struct {
	tokens []string
}

func NewModulePath(path []string) ModulePath {
	return ModulePath{
		tokens: path,
	}
}

func (mp *ModulePath) AddPath(path string) {
	mp.tokens = append(mp.tokens, path)
}

func (mp ModulePath) GetName() string {
	return concatPaths(mp.tokens, "::")
}

func (mp ModulePath) Has() bool {
	return len(mp.tokens) > 0
}

func concatPaths(slice []string, delimiter string) string {
	result := ""
	n := len(slice)

	// Reverse the slice
	for i := 0; i < n/2; i++ {
		slice[i], slice[n-1-i] = slice[n-1-i], slice[i]
	}

	for i, str := range slice {
		if i > 0 {
			result += delimiter
		}
		result += str
	}
	return result
}
