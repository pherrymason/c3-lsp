package symbols

import (
	"fmt"
	"strings"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

type FunctionType int

const (
	UserDefined = iota
	Method
	Macro
)

type Function struct {
	fType          FunctionType
	returnType     Type
	argumentIds    []string // Used to list which variables are defined in function signature. They are fully defined in Variables
	typeIdentifier string

	Variables map[string]*Variable

	BaseIndexable
}

func NewFunction(name string, returnType Type, argumentIds []string, module string, docId string, idRange Range, docRange Range) Function {
	return newFunctionType(UserDefined, "", name, returnType, argumentIds, module, docId, idRange, docRange, protocol.CompletionItemKindFunction)
}

func NewTypeFunction(typeIdentifier string, name string, returnType Type, argumentIds []string, module string, docId string, idRange Range, docRange Range, kind protocol.CompletionItemKind) Function {
	return newFunctionType(Method, typeIdentifier, name, returnType, argumentIds, module, docId, idRange, docRange, kind)
}

func NewTypeMacro(typeIdentifier string, name string, argumentIds []string, originalReturnType *Type, module string, docId string, idRange Range, docRange Range, kind protocol.CompletionItemKind) Function {
	var returnType Type
	if originalReturnType == nil {
		returnType = NewTypeFromString("", module)
	} else {
		returnType = *originalReturnType
	}
	return newFunctionType(Macro, typeIdentifier, name, returnType, argumentIds, module, docId, idRange, docRange, kind)
}

func NewMacro(name string, argumentIds []string, originalReturnType *Type, module string, docId string, idRange Range, docRange Range) Function {
	var returnType Type
	if originalReturnType == nil {
		returnType = NewTypeFromString("", module)
	} else {
		returnType = *originalReturnType
	}
	return newFunctionType(Macro, "", name, returnType, argumentIds, module, docId, idRange, docRange, protocol.CompletionItemKindFunction)
}

func newFunctionType(fType FunctionType, typeIdentifier string, name string, returnType Type, argumentIds []string, module string, docId string, identifierRangePosition Range, docRange Range, kind protocol.CompletionItemKind) Function {
	return Function{
		fType:          fType,
		returnType:     returnType,
		argumentIds:    argumentIds,
		typeIdentifier: typeIdentifier,
		BaseIndexable: NewBaseIndexable(
			name,
			module,
			docId,
			identifierRangePosition,
			docRange,
			kind,
		),
		Variables: make(map[string]*Variable),
	}
}

func (f Function) Id() string {
	return f.documentURI + f.module.GetName()
}

func (f Function) FunctionType() FunctionType {
	return f.fType
}

func (f Function) GetName() string {
	if f.typeIdentifier == "" {
		return f.name
	}

	return f.typeIdentifier + "." + f.name
}

func (f Function) GetMethodName() string {
	return f.name
}

func (f Function) GetFullName() string {
	if f.typeIdentifier == "" {
		return f.GetName()
	}

	return f.typeIdentifier + "." + f.name
}

func (f Function) GetFQN() string {
	return fmt.Sprintf("%s::%s", f.module.GetName(), f.GetName())
}

func (f Function) GetTypeIdentifier() string {
	return f.typeIdentifier
}

func (f Function) GetKind() protocol.CompletionItemKind {
	switch f.fType {
	case Method:
		return protocol.CompletionItemKindMethod

	default:
		return protocol.CompletionItemKindFunction
	}
}

func (f *Function) GetReturnType() *Type {
	return &f.returnType
}

func (f Function) ArgumentIds() []string {
	return f.argumentIds
}

func (f *Function) GetArguments() []*Variable {
	arguments := []*Variable{}
	for _, arg := range f.ArgumentIds() {
		arguments = append(arguments, f.Variables[arg])
	}

	return arguments
}

// Display the function's signature to show in types, on hover etc.
//
// Returns either a string like `fn void abc(int param)` if `includeName` is
// `true`, or `fn void(int param)` otherwise. If this is a macro, returns
// `macro void abc(int param; @body)` (with the correct keyword, and correctly
// displaying the trailing body parameter) instead.
func (f *Function) DisplaySignature(includeName bool) string {
	var declKeyword string
	if f.fType == Macro {
		declKeyword = "macro"
	} else {
		declKeyword = "fn"
	}

	name := ""
	if includeName {
		name = " " + f.GetFullName()
	}

	returnTypeData := f.GetReturnType()
	returnType := ""
	if returnTypeData != nil {
		returnType = returnTypeData.String()
	}
	if returnType != "" {
		returnType = " " + returnType
	}

	args := ""
	for _, arg := range f.argumentIds {
		variable := f.Variables[arg]
		if f.fType == Macro && strings.HasPrefix(variable.name, "@") {
			// Trailing @body in a macro
			// TODO: Maybe store this information properly, without needing string
			// manipulation at some point
			// However, this will do for now
			bodyParams := strings.TrimPrefix(variable.Type.String(), "fn void")
			if bodyParams == "()" {
				// Show '@body' instead of '@body()'
				bodyParams = ""
			}

			args += fmt.Sprintf("; %s%s", variable.name, bodyParams)
		} else {
			comma := ""
			if args != "" {
				comma = ", "
			}

			argName := variable.name
			if variable.idRange == (Range{}) {
				// Originally, it had an empty name
				argName = ""
			}

			argDefault := ""
			if variable.Arg.Default.IsSome() {
				argDefault = " = " + variable.Arg.Default.Get()
			}

			varArg := ""
			if variable.Arg.VarArg {
				// ...args
				varArg = "..."
			}

			argType := variable.Type.String()
			if argType != "" && argName != "" {
				if varArg == "" {
					// int args (no var args)
					argType += " "
				} else {
					// int[]... args (space required)
					varArg += " "
				}
			}

			if varArg != "" && argType != "" {
				// fix: int[]... args
				// -> int... args
				argType = strings.TrimSuffix(argType, "[]")
			}

			if varArg != "" && argName == "" && argType == "any*" {
				// Special case for C-style var args (fn name(...))
				//
				// fn name(any*...) -> fn name(...)
				argType = ""
			}

			args += fmt.Sprintf("%s%s%s%s%s", comma, argType, varArg, argName, argDefault)
		}
	}

	return fmt.Sprintf("%s%s%s(%s)", declKeyword, returnType, name, args)
}

func (f *Function) AddVariables(variables []*Variable) {
	for _, variable := range variables {
		f.Variables[variable.name] = variable
		f.Insert(variable)
	}
}

func (f *Function) AddVariable(variable *Variable) {
	f.Variables[variable.name] = variable
	f.Insert(variable)
}

func (f *Function) SetDocRange(docRange Range) {
	f.docRange = docRange
}

func (f *Function) SetStartPosition(position Position) {
	f.docRange.Start = position
}

func (f *Function) SetEndPosition(position Position) {
	f.docRange.End = position
}

func (f Function) GetHoverInfo() string {
	return f.DisplaySignature(true)
}

func (f Function) GetCompletionDetail() string {
	// Since this is just the type of the function, we don't include its name
	return f.DisplaySignature(false)
}
