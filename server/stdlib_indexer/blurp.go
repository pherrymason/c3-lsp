package main

import (
	"github.com/dave/jennifer/jen"
	s "github.com/pherrymason/c3-lsp/lsp/symbols"
)

const PackageName = "github.com/pherrymason/c3-lsp/lsp/"

func Generate_variable(variable *s.Variable, module *s.Module) jen.Code {
	return jen.
		Qual(PackageName+"symbols", "NewVariableBuilder").
		Call(
			jen.Lit(variable.GetName()),
			jen.Lit(variable.GetType().GetName()),
			jen.Lit(module.GetName()),
			jen.Lit(buildStdDocId(module.GetDocumentURI())),
		).
		Dot("Build").Call()
}

func Generate_struct(strukt *s.Struct, module *s.Module) jen.Code {
	def := jen.
		Qual(PackageName+"symbols", "NewStructBuilder").
		Call(
			jen.Lit(strukt.GetName()),
			jen.Lit(module.GetName()),
			jen.Lit(buildStdDocId(module.GetDocumentURI())),
		)

	for _, member := range strukt.GetMembers() {
		def.Dot("WithStructMember").
			Call(
				jen.Lit(member.GetName()),
				jen.Lit(member.GetType().GetName()),
				jen.Lit(module.GetName()),
				jen.Lit(module.GetDocumentURI()),
			)
	}

	def.
		Dot("WithoutSourceCode").Call().
		Dot("Build").Call()

	return def
}

func Generate_definition(def *s.Def, module *s.Module) jen.Code {
	defDef := jen.
		Qual(PackageName+"symbols", "NewDefBuilder").
		Call(
			jen.Lit(def.GetName()),
			jen.Lit(module.GetName()),
			jen.Lit(module.GetDocumentURI()),
		).
		Dot("WithResolvesTo").
		Call(
			jen.Lit(def.GetResolvesTo()),
		).
		Dot("WithoutSourceCode").Call().
		Dot("Build").Call()

	return defDef
}
