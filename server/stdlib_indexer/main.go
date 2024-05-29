package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/pherrymason/c3-lsp/fs"
	"github.com/pherrymason/c3-lsp/lsp/document"
	p "github.com/pherrymason/c3-lsp/lsp/parser"
	s "github.com/pherrymason/c3-lsp/lsp/symbols"
	"github.com/tliron/commonlog"

	"github.com/dave/jennifer/jen"
)

func main() {
	path := "./../../assets/c3c"
	c3cVersion := getC3Version(path)

	fmt.Printf("Version detected %s\n", c3cVersion)

	files, _ := fs.ScanForC3(fs.GetCanonicalPath(path + "/lib/std"))

	commonlog.Configure(2, nil)
	logger := commonlog.GetLogger("")
	parser := p.NewParser(logger)

	var stdLibModules []*s.Module
	for _, filePath := range files {
		//s.server.Log.Debug(fmt.Sprint("Parsing ", filePath))

		content, _ := os.ReadFile(filePath)
		doc := document.NewDocumentFromString(filePath, string(content))
		parsedModules := parser.ParseSymbols(&doc)

		stdLibModules = append(stdLibModules, parsedModules.Modules()...)
	}

	// generate code
	generateCode(stdLibModules, c3cVersion)
}

func getC3Version(path string) string {
	versionFile := path + "/src/version.h"
	content, err := os.ReadFile(versionFile)
	if err != nil {
		panic(fmt.Sprintf("Could not find c3c version: Could not open %s file: %s", versionFile, err))
	}

	text := string(content)
	versionRegex := regexp.MustCompile(`#define\s+COMPILER_VERSION\s+"([^"]+)"`)
	versionMatch := versionRegex.FindStringSubmatch(text)
	if len(versionMatch) > 1 {
		return versionMatch[1]
	}

	panic("Could not find c3c version: Did not find COMPILER_VERSION in versino.h")
}

func buildStdDocId(path string) string {
	return strings.ReplaceAll(path, "../../assets/c3c/lib/", "c3_stdlib::")
}

func generateCode(modules []*s.Module, c3Version string) {
	f := jen.NewFile("stdlib")
	versionIdentifier := "v" + strings.ReplaceAll(c3Version, ".", "")

	stmts := []jen.Code{}
	stmts = append(stmts,
		jen.Id("parsedModules").Op(":=").Qual(PackageName+"unit_modules", "NewParsedModules").Call(jen.Lit("_stdlib")),
		//jen.Var().Id("modules").Index().Add(jen.Op("*")).Qual(PackageName+"symbols", "Module"),
		jen.Var().Id("module").Add(jen.Op("*")).Qual(PackageName+"symbols", "Module"),
	)

	for _, mod := range modules {
		if mod.IsPrivate() {
			continue
		}

		modDefinition := jen.Id("module").Op("=").
			Qual(PackageName+"symbols", "NewModuleBuilder").
			Call(
				jen.Lit(mod.GetName()),
				jen.Lit(buildStdDocId(mod.GetDocumentURI())),
			).
			Dot("WithoutSourceCode").Call().
			Dot("Build").Call()

		for _, variable := range mod.Variables {
			// Generate variable
			varDef := Generate_variable(variable, mod)
			modDefinition.
				Dot("AddVariable").Call(varDef)
		}

		for _, strukt := range mod.Structs {
			structDef := Generate_struct(strukt, mod)
			modDefinition.
				Dot("AddStruct").Call(structDef)
		}
		for _, bitstruct := range mod.Bitstructs {
			bitstructDef := Generate_bitstruct(bitstruct, mod)
			modDefinition.
				Dot("AddBitstruct").Call(bitstructDef)
		}

		for _, def := range mod.Defs {
			defDef := Generate_definition(def, mod)
			modDefinition.
				Dot("AddDef").Call(defDef)
		}

		for _, enum := range mod.Enums {
			enumDef := Generate_enum(enum, mod)
			modDefinition.
				Dot("AddEnum").Call(enumDef)
		}

		for _, fault := range mod.Faults {
			enumDef := Generate_fault(fault, mod)
			modDefinition.
				Dot("AddFault").Call(enumDef)
		}

		for _, fun := range mod.ChildrenFunctions {
			// Generate functions
			funDef := Generate_function(fun, mod)

			modDefinition.
				Dot("AddFunction").Call(funDef)
		}

		stmts = append(
			stmts,
			jen.Line(),
			jen.Comment("Define module "+mod.GetName()),
			modDefinition,
			jen.Id("parsedModules").Dot("RegisterModule").Call(jen.Id("module")),
		)
	}

	stmts = append(stmts, jen.Return(jen. /*.Add(jen.Op("&"))*/ Id("parsedModules")))

	f.Func().
		Id("Load_"+versionIdentifier+"_stdlib").
		Params().
		/*Add(jen.Op("*")).*/ Qual(PackageName+"unit_modules", "UnitModules").
		Block(stmts...)

	err := f.Save("../lsp/language/stdlib/" + versionIdentifier + ".go")
	if err != nil {
		log.Fatal(err)
	}
}
