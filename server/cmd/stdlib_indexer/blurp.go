package main

import (
	"strings"

	"github.com/dave/jennifer/jen"
	"github.com/pherrymason/c3-lsp/pkg/cast"
	"github.com/pherrymason/c3-lsp/pkg/symbols"
	s "github.com/pherrymason/c3-lsp/pkg/symbols"
)

const InternalPackageName = "github.com/pherrymason/c3-lsp/internal/lsp/"
const PackageName = "github.com/pherrymason/c3-lsp/pkg/"

func Generate_variable(variable *s.Variable, module *s.Module) jen.Code {
	varDef := jen.
		Qual(PackageName+"symbols", "NewVariableBuilder").
		Call(
			jen.Lit(variable.GetName()),
			Generate_type(variable.GetType(), module.GetName()),
			jen.Lit(module.GetName()),
			jen.Lit(module.GetDocumentURI()),
		)

	if variable.Arg.VarArg {
		varDef.
			Dot("IsVarArg").
			Call()
	}

	if variable.Arg.Default.IsSome() {
		varDef.
			Dot("WithArgDefault").
			Call(
				jen.Lit(variable.Arg.Default.Get()),
			)
	}

	if variable.GetDocComment() != nil {
		varDef.
			Dot("WithDocs").
			Call(jen.Lit(variable.GetDocComment().GetBody()))
	}

	varDef.
		Dot("Build").
		Call()

	return varDef
}

func Generate_struct(strukt *s.Struct, module *s.Module) jen.Code {
	def := jen.
		Qual(PackageName+"symbols", "NewStructBuilder").
		Call(
			jen.Lit(strukt.GetName()),
			jen.Lit(module.GetName()),
			jen.Lit(module.GetDocumentURI()),
		)

	for _, member := range strukt.GetMembers() {
		def.Dot("WithStructMember").
			Call(
				jen.Lit(member.GetName()),
				Generate_type(member.GetType(), module.GetName()),
				jen.Lit(module.GetName()),
				jen.Lit(module.GetDocumentURI()),
			)
	}

	if strukt.GetDocComment() != nil {
		def.
			Dot("WithDocs").
			Call(jen.Lit(strukt.GetDocComment().GetBody()))
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
			Generate_type(cast.ToPtr(bitstruct.Type()), module.GetName()),
			jen.Lit(module.GetName()),
			jen.Lit(module.GetDocumentURI()),
		)

	for _, member := range bitstruct.Members() {
		def.Dot("WithStructMember").
			Call(
				jen.Lit(member.GetName()),
				Generate_type(member.GetType(), module.GetName()),
				jen.Lit(module.GetName()),
				jen.Lit(module.GetDocumentURI()),
			)
	}

	if bitstruct.GetDocComment() != nil {
		def.
			Dot("WithDocs").
			Call(jen.Lit(bitstruct.GetDocComment().GetBody()))
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
		)

	if def.ResolvesToType() {
		defDef.
			Dot("WithResolvesToType").
			Call(
				Generate_type(def.ResolvedType(), module.GetName()),
			)
	} else {
		defDef.
			Dot("WithResolvesTo").
			Call(
				jen.Lit(def.GetResolvesTo()),
			)
	}

	if def.GetDocComment() != nil {
		defDef.
			Dot("WithDocs").
			Call(jen.Lit(def.GetDocComment().GetBody()))
	}

	defDef.
		Dot("WithoutSourceCode").Call().
		Dot("Build").Call()

	return defDef
}

func Generate_distinct(distinct *s.Distinct, module *s.Module) jen.Code {
	distinctDef := jen.
		Qual(PackageName+"symbols", "NewDistinctBuilder").
		Call(
			jen.Lit(distinct.GetName()),
			jen.Lit(module.GetName()),
			jen.Lit(module.GetDocumentURI()),
		).
		Dot("WithInline").
		Call(
			jen.Lit(distinct.IsInline()),
		).
		Dot("WithBaseType").
		Call(
			Generate_type(distinct.GetBaseType(), module.GetName()),
		)

	if distinct.GetDocComment() != nil {
		distinctDef.
			Dot("WithDocs").
			Call(jen.Lit(distinct.GetDocComment().GetBody()))
	}

	distinctDef.
		Dot("WithoutSourceCode").Call().
		Dot("Build").Call()

	return distinctDef
}

func Generate_enum(enum *s.Enum, module *s.Module) jen.Code {
	enumDef := jen.
		Qual(PackageName+"symbols", "NewEnumBuilder").
		Call(
			jen.Lit(enum.GetName()),
			jen.Lit(enum.GetType()),
			jen.Lit(module.GetName()),
			jen.Lit(module.GetDocumentURI()),
		)

	for _, enumerator := range enum.GetEnumerators() {
		var assvalues []jen.Code
		for _, asv := range enumerator.AssociatedValues {
			assvalues = append(
				assvalues,
				jen.Add(jen.Op("*")).Qual(PackageName+"symbols", "NewVariableBuilder").
					Call(
						jen.Lit(asv.GetName()),
						Generate_type(asv.GetType(), asv.GetModuleString()),
						jen.Lit(asv.GetModuleString()),
						jen.Lit(module.GetDocumentURI()),
					).
					Dot("Build").Call(),
			)
		}

		associativeValues := jen.Index().Qual(PackageName+"symbols", "Variable").Values(assvalues...)

		enumDef.
			Dot("WithEnumerator").
			Call(
				jen.Qual(PackageName+"symbols", "NewEnumeratorBuilder").
					Call(
						jen.Lit(enumerator.GetName()),
						jen.Lit(module.GetDocumentURI()),
					).
					Dot("WithAssociativeValues").Call(associativeValues).
					Dot("WithEnumName").Call(jen.Lit(enum.GetName())).
					Dot("Build").Call(),
			)
	}

	if enum.GetDocComment() != nil {
		enumDef.
			Dot("WithDocs").
			Call(jen.Lit(enum.GetDocComment().GetBody()))
	}

	enumDef.Dot("Build").Call()

	return enumDef
}

func Generate_fault(fault *s.Fault, module *s.Module) jen.Code {
	faultDef := jen.
		Qual(PackageName+"symbols", "NewFaultBuilder").
		Call(
			jen.Lit(fault.GetName()),
			jen.Lit(fault.GetType()),
			jen.Lit(module.GetName()),
			jen.Lit(module.GetDocumentURI()),
		)

	for _, enumerator := range fault.GetConstants() {
		faultDef.
			Dot("WithConstant").
			Call(
				jen.Qual(PackageName+"symbols", "NewFaultConstantBuilder").
					Call(
						jen.Lit(enumerator.GetName()),
						jen.Lit(module.GetName()),
						jen.Lit(enumerator.GetDocumentURI()),
					).
					Dot("WithFaultName").Call(jen.Lit(fault.GetName())).
					Dot("Build").Call(),
			)
	}

	if fault.GetDocComment() != nil {
		faultDef.
			Dot("WithDocs").
			Call(jen.Lit(fault.GetDocComment().GetBody()))
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
				Generate_type(fun.GetReturnType(), mod.GetName()),
				jen.Lit(mod.GetName()),
				jen.Lit(mod.GetDocumentURI()),
			).
			Dot("WithTypeIdentifier").
			Call(jen.Lit(fun.GetTypeIdentifier()))
	} else {
		funDef = jen.
			Qual(PackageName+"symbols", "NewFunctionBuilder").
			Call(
				jen.Lit(fun.GetFullName()),
				Generate_type(fun.GetReturnType(), mod.GetName()),
				jen.Lit(mod.GetName()),
				jen.Lit(mod.GetDocumentURI()),
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

	if fun.GetDocComment() != nil {
		funDef.Dot("WithDocs").Call(Generate_doc_comment(fun.GetDocComment()))
	}

	funDef.
		Dot("WithoutSourceCode").Call().
		Dot("Build").Call()

	return funDef
}

func Generate_type(type_ *s.Type, mod string) *jen.Statement {
	builderName := "NewTypeBuilder"
	typeModule := type_.GetModule()

	if type_.IsBaseTypeLanguage() {
		// Use this shorthand just to reduce generated code by a bit
		builderName = "NewBaseTypeBuilder"
	}

	if type_.IsGenericArgument() {
		builderName = "NewGenericTypeBuilder"

		// Temporary fix for generic types' modules being misdetected
		typeModule = mod
	}

	ptrs := strings.Repeat("*", type_.GetPointerCount())

	typeDef := jen.
		Qual(PackageName+"symbols", builderName).
		Call(
			jen.Lit(type_.GetName()+ptrs),
			jen.Lit(typeModule),
		)

	if type_.IsOptional() {
		typeDef = typeDef.
			Dot("IsOptional").
			Call()
	}

	if type_.IsCollection() {
		colSize := type_.GetCollectionSize()
		if colSize.IsSome() {
			typeDef = typeDef.
				Dot("IsCollectionWithSize").
				Call(jen.Lit(colSize.Get()))
		} else {
			typeDef = typeDef.
				Dot("IsUnsizedCollection").
				Call()
		}
	}

	if type_.HasGenericArguments() {
		generatedGenericArgs := []jen.Code{}
		genericArgs := type_.GetGenericArguments()

		for i := range genericArgs {
			generatedGenericArgs = append(generatedGenericArgs, Generate_type(&genericArgs[i], mod))
		}

		typeDef = typeDef.
			Dot("WithGenericArguments").
			Call(generatedGenericArgs...)
	}

	typeDef.
		Dot("Build").
		Call()

	return typeDef
}

func Generate_doc_comment(docComment *s.DocComment) *jen.Statement {
	docDef := jen.
		Qual(PackageName+"symbols", "NewDocCommentBuilder").
		Call(
			jen.Lit(docComment.GetBody()),
		)

	if docComment.HasContracts() {
		for _, contract := range docComment.GetContracts() {
			docDef.
				Dot("WithContract").
				Call(
					jen.Lit(contract.GetName()),
					jen.Lit(contract.GetBody()),
				)
		}
	}

	docDef.
		Dot("Build").
		Call()

	return docDef
}
