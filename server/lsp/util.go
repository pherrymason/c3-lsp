package lsp

func boolPtr(v bool) *bool {
	b := v
	return &b
}
