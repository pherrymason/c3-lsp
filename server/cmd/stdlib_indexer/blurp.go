package main

import (
	"github.com/dave/jennifer/jen"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	s "github.com/pherrymason/c3-lsp/pkg/symbols"
)

const InternalPackageName = "github.com/pherrymason/c3-lsp/internal/lsp/"
const PackageName = "github.com/pherrymason/c3-lsp/pkg/"

func Generate_variable(variable *s.Variable, module *s.Module) jen.Code {
	return jen.
		Qual(PackageName+"symbols", "NewVariableBuilder").
		Call(
			jen.Lit(variable.GetName()),
			jen.Lit(variable.GetType().GetName()),
			jen.Lit(module.GetName()),
			//jen.Lit(buildStdDocId(mod.GetDocumentURI())),
			jen.Op("&").Id("docId"),
		).
		Dot("Build").Call()
}

func Generate_struct(strukt *s.Struct, module *s.Module) jen.Code {
	def := jen.
		Qual(PackageName+"symbols", "NewStructBuilder").
		Call(
			jen.Lit(strukt.GetName()),
			jen.Lit(module.GetName()),
			//jen.Lit(buildStdDocId(mod.GetDocumentURI())),
			jen.Op("&").Id("docId"),
		)

	for _, member := range strukt.GetMembers() {
		def.Dot("WithStructMember").
			Call(
				jen.Lit(member.GetName()),
				jen.Lit(member.GetType().GetName()),
				jen.Lit(module.GetName()),
				//jen.Lit(buildStdDocId(mod.GetDocumentURI())),
				jen.Op("&").Id("docId"),
			)
	}

	def.
		Dot("WithoutSourceCode").Call().
		Dot("Build").Call()

	return def
}

func Generate_bitstruct(bitstruct *s.Bitstruct, module *s.Module) jen.Code {
	def := jen.
		Qual(PackageName+"symbols", "NewBitstructBuilder").
		Call(
			jen.Lit(bitstruct.GetName()),
			jen.Lit(bitstruct.Type().GetName()),
			jen.Lit(module.GetName()),
			//jen.Lit(buildStdDocId(mod.GetDocumentURI())),
			jen.Op("&").Id("docId"),
		)

	for _, member := range bitstruct.Members() {
		def.Dot("WithStructMember").
			Call(
				jen.Lit(member.GetName()),
				jen.Lit(member.GetType().GetName()),
				jen.Lit(module.GetName()),
				//jen.Lit(buildStdDocId(mod.GetDocumentURI())),
				jen.Op("&").Id("docId"),
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
			//jen.Lit(buildStdDocId(mod.GetDocumentURI())),
			jen.Op("&").Id("docId"),
		).
		Dot("WithResolvesTo").
		Call(
			jen.Lit(def.GetResolvesTo()),
		).
		Dot("WithoutSourceCode").Call().
		Dot("Build").Call()

	return defDef
}

func Generate_enum(enum *s.Enum, module *s.Module) jen.Code {
	// NewEnumBuilder(name string, baseType string, module string, docId string)
	enumDef := jen.
		Qual(PackageName+"symbols", "NewEnumBuilder").
		Call(
			jen.Lit(enum.GetName()),
			jen.Lit(enum.GetType()),
			jen.Lit(module.GetName()),
			//jen.Lit(buildStdDocId(mod.GetDocumentURI())),
			jen.Op("&").Id("docId"),
		)

	for _, enumerator := range enum.GetEnumerators() {
		var assvalues []jen.Code
		if len(enumerator.GetAssociatedValues()) > 0 {
			for _, asv := range enumerator.GetAssociatedValues() {
				assvalues = append(
					assvalues,
					jen.Add(jen.Op("*")).Qual(PackageName+"symbols", "NewVariableBuilder").
						Call(
							jen.Lit(asv.GetName()),
							jen.Lit(asv.GetType().GetName()),
							jen.Lit(asv.GetModuleString()),
							//jen.Lit(buildStdDocId(mod.GetDocumentURI())),
							jen.Op("&").Id("docId"),
						).
						Dot("Build").Call(),
				)
			}
		}
		associativeValues := jen.Index().Qual(PackageName+"symbols", "Variable").Values(assvalues...)

		enumDef.
			Dot("WithEnumerator").
			Call(
				jen.Qual(PackageName+"symbols", "NewEnumeratorBuilder").
					Call(
						jen.Lit(enumerator.GetName()),
						//jen.Lit(buildStdDocId(mod.GetDocumentURI())),
						jen.Op("&").Id("docId"),
					).
					Dot("WithAssociativeValues").Call(associativeValues).
					Dot("Build").Call(),
			)
	}

	enumDef.Dot("Build").Call()

	return enumDef
}

func Generate_fault(fault *s.Fault, module *s.Module) jen.Code {
	// NewEnumBuilder(name string, baseType string, module string, docId string)
	faultDef := jen.
		Qual(PackageName+"symbols", "NewFaultBuilder").
		Call(
			jen.Lit(fault.GetName()),
			jen.Lit(fault.GetType()),
			jen.Lit(module.GetName()),
			//jen.Lit(buildStdDocId(mod.GetDocumentURI())),
			jen.Op("&").Id("docId"),
		)

	for _, enumerator := range fault.GetConstants() {
		faultDef.
			Dot("WithConstant").
			Call(
				jen.Qual(PackageName+"symbols", "NewFaultConstantBuilder").
					Call(
						jen.Lit(enumerator.GetName()),
						jen.Op("&").Id("docId"), //(enumerator.GetDocumentURI()),
					).
					Dot("Build").Call(),
			)
	}

	faultDef.Dot("Build").Call()

	return faultDef
}

func Generate_function(fun *s.Function, mod *s.Module) jen.Code {
	var funDef *jen.Statement
	if fun.FunctionType() == symbols.Method {
		funDef = jen.
			Qual(PackageName+"symbols", "NewFunctionBuilder").
			Call(
				jen.Lit(fun.GetMethodName()),
				jen.Qual(PackageName+"symbols", "NewTypeFromString").
					Call(
						jen.Lit(fun.GetReturnType().String()),
						jen.Lit(mod.GetName()),
					),
				jen.Lit(mod.GetName()),
				//jen.Lit(buildStdDocId(mod.GetDocumentURI())),
				jen.Op("&").Id("docId"),
			).
			Dot("WithTypeIdentifier").
			Call(jen.Lit(fun.GetTypeIdentifier()))
	} else {
		funDef = jen.
			Qual(PackageName+"symbols", "NewFunctionBuilder").
			Call(
				jen.Lit(fun.GetFullName()),
				jen.Qual(PackageName+"symbols", "NewTypeFromString").
					Call(
						jen.Lit(fun.GetReturnType().String()),
						jen.Lit(mod.GetName()),
					),
				jen.Lit(mod.GetName()),
				//jen.Lit(buildStdDocId(mod.GetDocumentURI())),
				jen.Op("&").Id("docId"),
			)
	}

	for _, arg := range fun.ArgumentIds() {
		variable := fun.Variables[arg]
		varDef := Generate_variable(variable, mod)
		funDef.Dot("WithArgument").Call(varDef)
	}

	if fun.FunctionType() == s.Macro {
		funDef.Dot("IsMacro").Call()
	}

	funDef.
		Dot("WithoutSourceCode").Call().
		Dot("Build").Call()

	return funDef
}
