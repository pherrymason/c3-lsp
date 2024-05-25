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

	var stmts []jen.Code
	for _, mod := range modules {

		modDefinition := jen.
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

		for _, def := range mod.Defs {
			defDef := Generate_definition(def, mod)
			modDefinition.
				Dot("AddDef").Call(defDef)
		}

		for _, fun := range mod.ChildrenFunctions {
			// Generate functions
			funDef := jen.
				Qual(PackageName+"symbols", "NewFunctionBuilder").
				Call(
					jen.Lit(fun.GetFullName()),
					jen.Lit(fun.GetReturnType()),
					jen.Lit(mod.GetName()),
					jen.Lit(buildStdDocId(mod.GetDocumentURI())),
				)

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

			modDefinition.
				Dot("AddFunction").Call(funDef)
		}

		stmts = append(
			stmts,
			jen.Line(),
			jen.Comment("Define module "+mod.GetName()),
			modDefinition,
		)
	}

	f.Func().
		Id("load_" + versionIdentifier + "_stdlib").
		Params().
		Block(stmts...)

	err := f.Save("stdlib/" + versionIdentifier + ".go")
	if err != nil {
		log.Fatal(err)
	}
}
