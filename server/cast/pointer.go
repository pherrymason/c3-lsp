package cast

func BoolPtr(v bool) *bool {
	b := v
	return &b
}
