package search_v2

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindImplementationsInWorkspace_delegates_to_v1(t *testing.T) {
	state := NewTestState()
	search := NewSearchV2WithoutLog()

	body := `module app;
interface MyInterface {
    fn void gre|||et();
}

struct Cat (MyInterface) {
    int age;
}

fn void Cat.greet(&self) {}

struct Dog (MyInterface) {
    int age;
}

fn void Dog.greet(&self) {}`

	cursorlessBody, position := parseBodyWithCursor(body)
	state.RegisterDoc("app.c3", cursorlessBody)

	implementations := search.FindImplementationsInWorkspace("app.c3", position, &state.State)

	assert.Len(t, implementations, 2)
	actualNames := []string{implementations[0].GetName(), implementations[1].GetName()}
	assert.ElementsMatch(t, []string{"Cat.greet", "Dog.greet"}, actualNames)
}
