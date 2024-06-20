package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/pherrymason/c3-lsp/pkg/document"
	"github.com/pherrymason/c3-lsp/pkg/fs"
	p "github.com/pherrymason/c3-lsp/pkg/parser"
	"github.com/pherrymason/c3-lsp/pkg/symbols_table"
	"github.com/tliron/commonlog"

	"github.com/dave/jennifer/jen"
)

func main() {
	path := "./../../../assets/c3c"
	c3cVersion := getC3Version(path)

	fmt.Printf("Version detected %s\n", c3cVersion)

	files, _ := fs.ScanForC3(fs.GetCanonicalPath(path + "/lib/std"))

	commonlog.Configure(2, nil)
	logger := commonlog.GetLogger("")
	parser := p.NewParser(logger)

	symbolsTable := symbols_table.NewSymbolsTable()
	for _, filePath := range files {
		//s.server.Log.Debug(fmt.Sprint("Parsing ", filePath))

		content, _ := os.ReadFile(filePath)
		doc := document.NewDocumentFromString(filePath, string(content))
		parsedModules, pendingTypes := parser.ParseSymbols(&doc)

		symbolsTable.Register(parsedModules, pendingTypes)
	}

	// generate code
	generateCode(&symbolsTable, c3cVersion)
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
	return strings.ReplaceAll(path, "../../../assets/c3c/lib/", "c3_stdlib::")
}

func generateCode(symbolsTable *symbols_table.SymbolsTable, c3Version string) {
	f := jen.NewFile("stdlib")
	versionIdentifier := "v" + strings.ReplaceAll(c3Version, ".", "")

	dict := jen.Dict{}

	uniqueModuleNames := map[string]bool{}
	for _, ps := range symbolsTable.All() {
		for _, mod := range ps.Modules() {
			if mod.IsPrivate() {
				continue
			}

			_, ok := uniqueModuleNames[mod.GetName()]
			if !ok {
				uniqueModuleNames[mod.GetName()] = true

				dict[jen.Lit(mod.GetName())] =
					jen.
						Qual(PackageName+"symbols", "NewModuleBuilder").
						Call(
							jen.Lit(mod.GetName()),
							jen.Op("&").Id("docId"),
						).
						Dot("WithoutSourceCode").Call().
						Dot("Build").Call()

			}
		}
	}

	stmts := []jen.Code{
		jen.Id("docId").Op(":=").Lit("_stdlib"),
		jen.Id("moduleCollection").Op(":=").Map(jen.String()).Add(jen.Op("*")).Qual(PackageName+"symbols", "Module").Values(dict),
		jen.Id("parsedModules").Op(":=").Qual(PackageName+"symbols_table", "NewParsedModules").Call(jen.Op("&").Id("docId")),
		jen.For(
			jen.Id("_").Op(",").Id("mod").Op(":=").Range().
				Id("moduleCollection"),
		).Block(
			//jen.Qual("fmt", "Println").Call(jen.Id("i")),
			jen.Id("parsedModules").Dot("RegisterModule").Call(jen.Id("mod")),
		),
		jen.Var().Id("module").Add(jen.Op("*")).Qual(PackageName+"symbols", "Module"),
	}

	for _, ps := range symbolsTable.All() {
		for _, mod := range ps.Modules() {
			if mod.IsPrivate() {
				continue
			}
			modDefinition := jen.Id("module")
			somethingAdded := false

			for _, variable := range mod.Variables {
				somethingAdded = true
				// Generate variable
				varDef := Generate_variable(variable, mod)
				modDefinition.
					Dot("AddVariable").Call(varDef)
			}

			for _, strukt := range mod.Structs {
				somethingAdded = true
				structDef := Generate_struct(strukt, mod)
				modDefinition.
					Dot("AddStruct").Call(structDef)
			}
			for _, bitstruct := range mod.Bitstructs {
				somethingAdded = true
				bitstructDef := Generate_bitstruct(bitstruct, mod)
				modDefinition.
					Dot("AddBitstruct").Call(bitstructDef)
			}

			for _, def := range mod.Defs {
				somethingAdded = true
				defDef := Generate_definition(def, mod)
				modDefinition.
					Dot("AddDef").Call(defDef)
			}

			for _, enum := range mod.Enums {
				somethingAdded = true
				enumDef := Generate_enum(enum, mod)
				modDefinition.
					Dot("AddEnum").Call(enumDef)
			}

			for _, fault := range mod.Faults {
				somethingAdded = true
				enumDef := Generate_fault(fault, mod)
				modDefinition.
					Dot("AddFault").Call(enumDef)
			}

			for _, fun := range mod.ChildrenFunctions {
				somethingAdded = true
				// Generate functions
				funDef := Generate_function(fun, mod)

				modDefinition.
					Dot("AddFunction").Call(funDef)
			}

			if somethingAdded {
				stmts = append(
					stmts,
					jen.Line(),
					jen.Comment("Define module "+mod.GetName()),
					jen.Id("module").Op("=").
						Id("moduleCollection").Index(jen.Lit(mod.GetName())),
					modDefinition,
				)
			}
		}
	}

	stmts = append(stmts, jen.Return(jen.Id("parsedModules")))

	f.Func().
		Id("Load_"+versionIdentifier+"_stdlib").
		Params().
		Qual(PackageName+"symbols_table", "UnitModules").
		Block(stmts...)

	err := f.Save("../../internal/lsp/language/stdlib/" + versionIdentifier + ".go")
	if err != nil {
		log.Fatal(err)
	}
}
