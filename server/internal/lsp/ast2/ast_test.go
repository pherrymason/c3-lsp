package ast2

import (
	"fmt"
	"go/token"
	"testing"
)

func Test_walk(t *testing.T) {
	file := &File{
		Name: Ident{0, "main.c3"},
		Decls: []Decl{
			&VarDecl{Name: Ident{0, "x"}, Values: []Expr{
				&BasicLit{
					ValuePos: 2,
					Kind:     token.INT,
					Value:    "2",
				},
			}},
		},
	}

	if file.FileStart == 0 {

	}
	Walky(file)
}

func Walky(n Node) {
	switch n.(type) {
	case *File:
		fmt.Println("Is a file")
		for _, decls := range n.(*File).Decls {
			Walky(decls)
		}

	case *VarDecl:
		fmt.Println("It's a vardecl")
	default:
		fmt.Println("I dont know")
	}
}
