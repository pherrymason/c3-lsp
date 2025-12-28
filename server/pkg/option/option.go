package option

import (
	"encoding/json"
	"fmt"
)

type Option[T any] struct {
	value  T
	isFull bool
}

func Some[T any](value T) Option[T] {
	return Option[T]{value: value, isFull: true}
}

func None[T any]() Option[T] {
	return Option[T]{isFull: false}
}

func (s Option[T]) Get() T {
	if s.isFull {
		return s.value
	} else {
		panic("cannot get from None type")
	}
}

func (s *Option[T]) GetOrElse(other T) T {
	if s.isFull {
		return s.value
	} else {
		return other
	}
}

func (s *Option[_]) IsNone() bool {
	return !s.isFull
}

func (s *Option[_]) IsSome() bool {
	return s.isFull
}

func (s Option[T]) String() string {
	if s.isFull {
		return fmt.Sprintf("Some(%v)", s.value)
	} else {
		return "None"
	}
}

func (o Option[T]) MarshalJSON() ([]byte, error) {
	if o.isFull {
		return json.Marshal(o.value)
	}
	return []byte("null"), nil
}

func (o *Option[T]) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*o = None[T]()
		return nil
	}

	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	*o = Some(value)
	return nil
}
