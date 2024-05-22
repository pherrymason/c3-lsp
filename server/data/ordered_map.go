package data

type OrderedMap[T any] struct {
	keys   []string
	values map[string]T
}

func NewOrderedMap[T any]() *OrderedMap[T] {
	return &OrderedMap[T]{
		keys:   []string{},
		values: make(map[string]T),
	}
}

func (om *OrderedMap[T]) Set(key string, value T) {
	if _, exists := om.values[key]; !exists {
		om.keys = append(om.keys, key)
	}
	om.values[key] = value
}

func (om *OrderedMap[T]) Get(key string) (T, bool) {
	value, exists := om.values[key]
	return value, exists
}
func (om *OrderedMap[T]) Has(key string) bool {
	_, exists := om.values[key]
	return exists
}

func (om *OrderedMap[T]) Keys() []string {
	return om.keys
}

func (om *OrderedMap[T]) Values() []T {
	values := []T{}
	for _, key := range om.keys {
		values = append(values, om.values[key])
	}
	return values
}
