package analysis

import (
	"github.com/pherrymason/c3-lsp/internal/lsp/ast"
	"github.com/pherrymason/c3-lsp/internal/lsp/ast/walk"
	"strings"
)

// defValueToCodeVisitor converts a DefValue node to code
type defValueToCodeVisitor struct {
	code              string
	genericParameters []string
}

func (v *defValueToCodeVisitor) Enter(node ast.Node, propertyName string) walk.Visitor {
	switch n := node.(type) {
	case *ast.GenDecl:
		switch spec := n.Spec.(type) {
		case *ast.DefSpec:
			//v.code += spec.Name.Name
			walk.Walk(v, spec.Value, "Value")
			if len(spec.GenericParameters) > 0 {
				v.genericParameters = []string{}
				v.code += "(<"
				for _, gn := range spec.GenericParameters {
					walk.Walk(v, gn, "GenericParameter")
				}
				v.code += strings.Join(v.genericParameters, ", ")
				v.code += ">)"
			}
		default:
			// ignore

		}
	case *ast.Ident:
		v.code += n.Name

	case *ast.TypeInfo:
		if propertyName == "GenericParameter" {
			v.genericParameters = append(v.genericParameters, n.Identifier.Name)
		} else {
			v.code += n.String()
		}
	case *ast.FuncType:
		v.code += "fn "
		if n.ReturnType != nil {
			v.code += n.ReturnType.String() + " "
		}
		v.code += "("
		if len(n.Params) > 0 {
			params := []string{}
			for _, gn := range n.Params {
				params = append(params, gn.Type.String())
			}
			v.code += strings.Join(params, ", ")
		}
		v.code += ")"

	case *ast.FunctionDecl:
		v.code += "fn "
		if n.Signature != nil {
			if n.Signature.ReturnType != nil {
				v.code += n.Signature.ReturnType.String() + " "
			}
			if n.Signature.Name != nil {
				v.code += n.Signature.Name.Name
			}
		}
		v.code += "("
		if len(n.Signature.Parameters) > 0 {
			v.genericParameters = []string{}
			for _, gn := range n.Signature.Parameters {
				walk.Walk(v, gn, "Parameter")
			}
			v.code += strings.Join(v.genericParameters, ", ")
		}
		v.code += ")"
	}

	return nil
}

func (v *defValueToCodeVisitor) Exit(n ast.Node, propertyName string) {}
