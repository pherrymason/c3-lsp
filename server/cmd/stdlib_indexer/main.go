package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
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

func generateCode(symbolsTable *symbols_table.SymbolsTable, c3Version string) {
	f := jen.NewFile("stdlib")
	versionIdentifier := "v" + strings.ReplaceAll(c3Version, ".", "")

	dict := jen.Dict{}

	uniqueModuleNames := map[string]bool{}
	for docId, ps := range symbolsTable.All() {
		for _, mod := range ps.Modules() {
			if mod.IsPrivate() {
				continue
			}

			// Rewrite its docId
			mod.SetDocumentURI(strings.ReplaceAll(docId, "../../../assets/c3c/lib/std", "<stdlib-path>"))

			_, ok := uniqueModuleNames[mod.GetName()]
			if !ok {
				uniqueModuleNames[mod.GetName()] = true

				dict[jen.Lit(mod.GetName())] =
					jen.
						Qual(PackageName+"symbols", "NewModuleBuilder").
						Call(
							jen.Lit(mod.GetName()),
							jen.Lit(mod.GetDocumentURI()),
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
			jen.Id("parsedModules").Dot("RegisterModule").Call(jen.Id("mod")),
		),
		jen.Var().Id("module").Add(jen.Op("*")).Qual(PackageName+"symbols", "Module"),
	}

	for _, ps := range sortedValues(symbolsTable.All()) {
		for _, mod := range ps.Modules() {
			if mod.IsPrivate() {
				continue
			}
			somethingAdded := false
			modDefinition := jen.Id("module")
			if len(mod.GenericParameters) > 0 {
				somethingAdded = true
				genericParametersDef := jen.Dict{}
				for key, gen := range mod.GenericParameters {
					genericParametersDef[jen.Lit(key)] =
						jen.Qual(PackageName+"symbols", "NewGenericParameter").
							Call(
								jen.Lit(gen.GetName()),
								jen.Lit(mod.GetName()),
								jen.Lit(mod.GetDocumentURI()),
								jen.Qual(PackageName+"symbols", "NewRange").Call(
									jen.Lit(0), jen.Lit(0), jen.Lit(0), jen.Lit(0),
								),
								jen.Qual(PackageName+"symbols", "NewRange").Call(
									jen.Lit(0), jen.Lit(0), jen.Lit(0), jen.Lit(0),
								),
							)
				}

				modDefinition.
					Dot("SetGenericParameters").
					Call(
						jen.Map(jen.String()).Op("*").Qual(PackageName+"symbols", "GenericParameter").
							Values(genericParametersDef),
					)
			}

			for _, variable := range sortedValues(mod.Variables) {
				somethingAdded = true
				// Generate variable
				varDef := Generate_variable(variable, mod)
				modDefinition.
					Dot("AddVariable").Call(varDef)
			}

			for _, strukt := range sortedValues(mod.Structs) {
				somethingAdded = true
				structDef := Generate_struct(strukt, mod)
				modDefinition.
					Dot("AddStruct").Call(structDef)
			}

			for _, bitstruct := range sortedValues(mod.Bitstructs) {
				somethingAdded = true
				bitstructDef := Generate_bitstruct(bitstruct, mod)
				modDefinition.
					Dot("AddBitstruct").Call(bitstructDef)
			}

			for _, def := range sortedValues(mod.Defs) {
				somethingAdded = true
				defDef := Generate_definition(def, mod)
				modDefinition.
					Dot("AddDef").Call(defDef)
			}

			for _, distinct := range sortedValues(mod.Distincts) {
				somethingAdded = true
				distinctDef := Generate_distinct(distinct, mod)
				modDefinition.
					Dot("AddDistinct").Call(distinctDef)
			}

			for _, enum := range sortedValues(mod.Enums) {
				somethingAdded = true
				enumDef := Generate_enum(enum, mod)
				modDefinition.
					Dot("AddEnum").Call(enumDef)
			}

			for _, fault := range sortedValues(mod.Faults) {
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

	err := f.Save("../../internal/lsp/stdlib/" + versionIdentifier + ".go")
	if err != nil {
		log.Fatal(err)
	}
}

func sortedValues[T any](m map[string]T) []T {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	values := make([]T, 0, len(m))
	for _, k := range keys {
		values = append(values, m[k])
	}
	return values
}
