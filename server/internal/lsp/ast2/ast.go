package ast2

import "go/token"

type Pos int

type Node interface {
	Pos() token.Pos // position of first character belonging to the node
	End() token.Pos // position of first character immediately after the node
}
type Expr interface {
	Node
	exprNode()
}

// All statement nodes implement the Stmt interface.
type Stmt interface {
	Node
	stmtNode()
}

// All declaration nodes implement the Decl interface.
type Decl interface {
	Node
	//declNode()
}

type Ident struct {
	NamePos token.Pos // identifier position
	Name    string    // identifier name
}

type File struct {
	Name  Ident  // package name
	Decls []Decl // top-level declarations; or nil

	FileStart, FileEnd token.Pos // start and end of entire file
}

func (f *File) Pos() token.Pos { return f.FileStart }
func (f *File) End() token.Pos { return f.FileEnd }

type VarDecl struct {
	Name     Ident
	Type     Ident
	Values   []Expr
	Position token.Pos
}

// func (VarDecl) declNode()      {}
func (*VarDecl) Pos() token.Pos { return token.NoPos }
func (*VarDecl) End() token.Pos { return token.NoPos }

type BasicLit struct {
	ValuePos token.Pos   // literal position
	Kind     token.Token // token.INT, token.FLOAT, token.IMAG, token.CHAR, or token.STRING
	Value    string      // literal string; e.g. 42, 0x7f, 3.14, 1e-9, 2.4i, 'a', '\x7f', "foo" or `\m\n\o`
}

func (*BasicLit) exprNode()      {}
func (*BasicLit) Pos() token.Pos { return token.NoPos }
func (*BasicLit) End() token.Pos { return token.NoPos }

func Walk() {
	//x := ast.Ident{NamePos: token.NoPos}
}
