package search

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindImplementationsInWorkspace_interface(t *testing.T) {
	state := NewTestState()
	search := NewSearchWithoutLog()

	body := `module app;
interface MyInterface {
    fn void greet();
}

struct Cat (MyInterface) {
    int age;
}

fn void Cat.greet(&self) {}

struct Dog (MyInterface) {
    int age;
}

fn void Dog.greet(&self) {}

fn void main() {
    MyInter|||face value;
}`

	cursorlessBody, position := parseBodyWithCursor(body)
	state.registerDoc("app.c3", cursorlessBody)

	implementations := search.FindImplementationsInWorkspace("app.c3", position, &state.state)

	assert.Len(t, implementations, 2)
	actualNames := []string{implementations[0].GetName(), implementations[1].GetName()}
	assert.ElementsMatch(t, []string{"Cat", "Dog"}, actualNames)
}

func TestFindImplementationsInWorkspace_interface_method(t *testing.T) {
	state := NewTestState()
	search := NewSearchWithoutLog()

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
	state.registerDoc("app.c3", cursorlessBody)

	implementations := search.FindImplementationsInWorkspace("app.c3", position, &state.state)

	assert.Len(t, implementations, 2)
	actualNames := []string{implementations[0].GetName(), implementations[1].GetName()}
	assert.ElementsMatch(t, []string{"Cat.greet", "Dog.greet"}, actualNames)
}
