/*
Copyright 2016-2017 by Milo Christiansen

This software is provided 'as-is', without any express or implied warranty. In
no event will the authors be held liable for any damages arising from the use of
this software.

Permission is granted to anyone to use this software for any purpose, including
commercial applications, and to alter it and redistribute it freely, subject to
the following restrictions:

1. The origin of this software must not be misrepresented; you must not claim
that you wrote the original software. If you use this software in a product, an
acknowledgment in the product documentation would be appreciated but is not
required.

2. Altered source versions must be plainly marked as such, and must not be
misrepresented as being the original software.

3. This notice may not be removed or altered from any source distribution.
*/

package ast

// Assign represents an assignment statement.
type Assign struct {
	stmtBase `json:"Assign"`

	// Is this a local variable declaration statement?
	LocalDecl bool

	// Special case handling for "local function f() end", this should be treated like "local f; f = function() end".
	LocalFunc bool

	Targets []Expr
	Values  []Expr // If len == 0 no values were given, if len == 1 then the value may be a multi-return function call.
}

// FuncCall is declared in the expression parts file (it is both an Expr and a Stmt).

// DoBlock represents a do block (do ... end).
type DoBlock struct {
	stmtBase `json:"DoBlock"`

	Block []Stmt
}

// If represents an if statement.
// 'elseif' statements are encoded as nested if statements.
type If struct {
	stmtBase `json:"If"`

	Cond Expr
	Then []Stmt
	Else []Stmt
}

// WhileLoop represents a while loop.
type WhileLoop struct {
	stmtBase `json:"WhileLoop"`

	Cond  Expr
	Block []Stmt
}

// RepeatUntilLoop represents a repeat-until loop.
type RepeatUntilLoop struct {
	stmtBase `json:"RepeatUntilLoop"`

	Cond  Expr
	Block []Stmt
}

// ForLoopNumeric represents a numeric for loop.
type ForLoopNumeric struct {
	stmtBase `json:"ForLoopNumeric"`

	Counter string

	Init  Expr
	Limit Expr
	Step  Expr

	Block []Stmt
}

// ForLoopGeneric represents a generic for loop.
type ForLoopGeneric struct {
	stmtBase `json:"ForLoopGeneric"`

	Locals []string
	Init   []Expr // This will always be adjusted to three return results, but AFAIK there is no actual limit on expression count.

	Block []Stmt
}

type Goto struct {
	stmtBase `json:"Goto"`

	// True if this Goto is actually a break statement. There is no matching label.
	// If Label is not "break" then this is actually a continue statement (a custom
	// extension that the default lexer/parser does not use).
	IsBreak bool
	Label   string
}

type Label struct {
	stmtBase `json:"Label"`

	Label string
}

type Return struct {
	stmtBase `json:"Return"`

	Items []Expr
}
