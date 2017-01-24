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

// Read a sequence of identifiers and indexing operations.
// If the ident chain ends with a :ident part this does not read it.
func (p *parser) ident() Expr {
	p.l.getCurrent(tknName)
	ident := exprLine(&ConstIdent{
		Value: p.l.current.Lexeme,
	}, p.l.current.Line)

	for p.l.checkLook(tknOIndex, tknDot) {
		switch p.l.look.Type {
		case tknOIndex: // [expr]
			p.l.getCurrent(tknOIndex)

			line := p.l.current.Line
			ident = exprLine(&TableAccessor{
				Obj: ident,
				Key: p.expression(),
			}, line)

			p.l.getCurrent(tknCIndex)
		case tknDot: // .ident
			p.l.getCurrent(tknDot)
			line := p.l.current.Line
			p.l.getCurrent(tknName)
			ident = exprLine(&TableAccessor{
				Obj: ident,
				Key: exprLine(&ConstString{
					Value: p.l.current.Lexeme,
				}, p.l.current.Line),
			}, line)
		default:
			panic("IMPOSSIBLE")
		}
	}
	return ident
}

// Handle a function call. The name must be already read (minus a method name if any).
func (p *parser) funcCall(ident Expr) Expr {
	line := p.l.current.Line
	var r, f Expr
	if p.l.checkLook(tknColon) {
		p.l.getCurrent(tknColon)
		p.l.getCurrent(tknName)
		r = ident
		f = exprLine(&ConstString{
			Value: p.l.current.Lexeme,
		}, p.l.current.Line)
	} else {
		f = ident
	}

	args := []Expr{}
	switch p.l.look.Type {
	case tknOBracket:
		args = append(args, p.tblConstruct())
	case tknString:
		p.l.getCurrent(tknString)
		args = append(args, exprLine(&ConstString{
			Value: p.l.current.Lexeme,
		}, p.l.current.Line))
	case tknOParen:
		p.l.getCurrent(tknOParen)
		for !p.l.checkLook(tknCParen) {
			args = append(args, p.expression())
			if !p.l.checkLook(tknSeperator) {
				break
			}
			p.l.getCurrent(tknSeperator)
		}
		p.l.getCurrent(tknCParen)
	default:
		p.l.getCurrent(tknOBracket, tknString, tknOParen) // For the error message
	}

	return exprLine(&FuncCall{
		Receiver: r,
		Function: f,
		Args:     args,
	}, line)
}

func (p *parser) funcDeclBody(hasSelf bool) Expr {
	// Read Parameters
	p.l.getCurrent(tknOParen)
	line := p.l.current.Line
	params := []string{}
	variadic := false
	if hasSelf {
		params = append(params, "self")
	}
	for p.l.checkLook(tknName, tknVariadic) {
		if p.l.checkLook(tknVariadic) {
			p.l.getCurrent(tknVariadic)
			variadic = true
			break
		}
		p.l.getCurrent(tknName)
		params = append(params, p.l.current.Lexeme)

		if !p.l.checkLook(tknSeperator) {
			break
		}
		p.l.getCurrent(tknSeperator)
		if !p.l.checkLook(tknName, tknVariadic) {
			p.l.getCurrent(tknName, tknVariadic) // Error message
		}
	}
	p.l.getCurrent(tknCParen)

	// Read Block
	block := p.block(tknEnd)

	return exprLine(&FuncDecl{
		Params:     params,
		IsVariadic: variadic,
		Block:      block,
	}, line)
}

func (p *parser) tblConstruct() Expr {
	vals, keys := []Expr{}, []Expr{}

	p.l.getCurrent(tknOBracket)
	line := p.l.current.Line

	for !p.l.checkLook(tknCBracket) {
		switch p.l.look.Type {
		case tknName:
			if p.l.exlook.Type != tknSet {
				keys = append(keys, nil)
				break
			}
			p.l.getCurrent(tknName)
			keys = append(keys, exprLine(&ConstString{Value: p.l.current.Lexeme}, p.l.current.Line))
			p.l.getCurrent(tknSet)
		case tknOIndex:
			p.l.getCurrent(tknOIndex)
			keys = append(keys, p.expression())
			p.l.getCurrent(tknCIndex)
			p.l.getCurrent(tknSet)
		default:
			keys = append(keys, nil)
		}
		vals = append(vals, p.expression())

		if !p.l.checkLook(tknSeperator, tknUnnecessary) {
			break
		}
		p.l.getCurrent(tknSeperator, tknUnnecessary)
	}

	p.l.getCurrent(tknCBracket)

	return exprLine(&TableConstructor{
		Keys: keys,
		Vals: vals,
	}, line)
}

func (p *parser) expression() Expr {
	return p.subexpr(0)
	//return p.valOr()
}

/*
Operators...

	^ (right associative)
	not   #     -     ~ (the unary operators)
	*     /     //    %
	+     -
	.. (right associative)
	<<    >>
	&
	~
	|
	<     >     <=    >=    ~=    ==
	and
	or

1+2+3
(1+2)+3

"a".."b".."c"
"a"..("b".."c")

"a".."b"..1+2+3
"a"..("b"..((1+2)+3))
*/

// Operator priorities
var priorities = [...]struct {
	left  int
	right int
}{
	{10, 10}, // OpAdd
	{10, 10}, // OpSub
	{11, 11}, // OpMul
	{11, 11}, // OpMod
	{14, 13}, // OpPow (right associative)
	{11, 11}, // OpDiv
	{11, 11}, // OpIDiv
	{6, 6},   // OpBinAND
	{4, 4},   // OpBinOR
	{5, 5},   // OpBinXOR
	{7, 7},   // OpBinShiftL
	{7, 7},   // OpBinShiftR
	{12, 12}, // OpUMinus
	{12, 12}, // OpBinNot
	{12, 12}, // OpNot
	{12, 12}, // OpLength
	{9, 8},   // OpConcat (right associative)

	{3, 3}, // OpEqual
	{3, 3}, // OpNotEqual
	{3, 3}, // OpLessThan
	{3, 3}, // OpGreaterThan
	{3, 3}, // OpLessOrEqual
	{3, 3}, // OpGreaterOrEqual

	{2, 2}, // OpAnd
	{1, 1}, // OpOr
}

var tknToBinOp = map[int]opTyp{
	tknAnd:    OpAnd,
	tknOr:     OpOr,
	tknAdd:    OpAdd,
	tknSub:    OpSub,
	tknMul:    OpMul,
	tknDiv:    OpDiv,
	tknIDiv:   OpIDiv,
	tknMod:    OpMod,
	tknPow:    OpPow,
	tknEQ:     OpEqual,
	tknGT:     OpGreaterThan,
	tknGE:     OpGreaterOrEqual,
	tknLT:     OpLessThan,
	tknLE:     OpLessOrEqual,
	tknNE:     OpNotEqual,
	tknShiftL: OpBinShiftL,
	tknShiftR: OpBinShiftR,
	tknBXOr:   OpBinXOR,
	tknBOr:    OpBinOR,
	tknBAnd:   OpBinAND,
	tknConcat: OpConcat,
}

var tknToUnOp = map[int]opTyp{
	tknSub:  OpUMinus,
	tknBXOr: OpBinNot,
	tknNot:  OpNot,
	tknLen:  OpLength,
}

func (p *parser) subexpr(limit int) Expr {
	// Grab the starting left hand side of the expression
	var e1 Expr
	op, ok := tknToUnOp[p.l.look.Type]
	if ok {
		p.l.advance()
		line := p.l.current.Line
		e1 = exprLine(&Operator{Op: op, Right: p.subexpr(12)}, line)
	} else {
		e1 = p.value()
	}

	// Then grab the right hand side. The old right then becomes the new left until we cannot find
	// anything with a priority higher than the limit anymore.
	op, ok = tknToBinOp[p.l.look.Type]
	for ok && priorities[op].left > limit {
		p.l.advance()
		line := p.l.current.Line
		e1 = exprLine(&Operator{Op: op, Left: e1, Right: p.subexpr(priorities[op].right)}, line)

		op, ok = tknToBinOp[p.l.look.Type]
	}
	return e1
}

/*
Below this point is the old expression parsing code. I wrote this the way I learned years ago, without
looking at the way it was done in standard Lua. Amazingly it handles everything correctly, except one
case: Any unary operator after a power operator will create a syntax error. Sadly I am not sure if this
code can be easily fixed to handle that case.

Anyway, I then looked at how standard Lua does it. Their method is much shorter, but a little harder to
understand... Oh, well. The above expression code uses something similar to standard Lua now.

Scroll down far enough and you will come to some code that is still in use, just FYI.
*/

// func (p *parser) valOr() Expr {
// 	l := p.valAnd()
// 	for p.l.checkLook(tknOr) {
// 		p.l.getCurrent(tknOr)
// 		line := p.l.current.Line
// 		l = exprLine(&Operator{Op: OpOr, Left: l, Right: p.valAnd()}, line)
// 	}
// 	return l
// }

// func (p *parser) valAnd() Expr {
// 	l := p.valCmp()
// 	for p.l.checkLook(tknAnd) {
// 		p.l.getCurrent(tknAnd)
// 		line := p.l.current.Line
// 		l = exprLine(&Operator{Op: OpAnd, Left: l, Right: p.valCmp()}, line)
// 	}
// 	return l
// }

// func (p *parser) valCmp() Expr {
// 	l := p.valBOr()
// 	for p.l.checkLook(tknEQ, tknGT, tknGE, tknLT, tknLE, tknNE) {
// 		p.l.getCurrent(tknEQ, tknGT, tknGE, tknLT, tknLE, tknNE)
// 		line := p.l.current.Line
// 		switch p.l.current.Type {
// 		case tknEQ:
// 			l = exprLine(&Operator{Op: OpEqual, Left: l, Right: p.valBOr()}, line)
// 		case tknGT:
// 			l = exprLine(&Operator{Op: OpGreaterThan, Left: l, Right: p.valBOr()}, line)
// 		case tknGE:
// 			l = exprLine(&Operator{Op: OpGreaterOrEqual, Left: l, Right: p.valBOr()}, line)
// 		case tknLT:
// 			l = exprLine(&Operator{Op: OpLessThan, Left: l, Right: p.valBOr()}, line)
// 		case tknLE:
// 			l = exprLine(&Operator{Op: OpLessOrEqual, Left: l, Right: p.valBOr()}, line)
// 		case tknNE:
// 			l = exprLine(&Operator{Op: OpNotEqual, Left: l, Right: p.valBOr()}, line)
// 		}
// 	}
// 	return l
// }

// func (p *parser) valBOr() Expr {
// 	l := p.valBXOr()
// 	for p.l.checkLook(tknBOr) {
// 		p.l.getCurrent(tknBOr)
// 		line := p.l.current.Line
// 		l = exprLine(&Operator{Op: OpBinOR, Left: l, Right: p.valBXOr()}, line)
// 	}
// 	return l
// }

// func (p *parser) valBXOr() Expr {
// 	l := p.valBAnd()
// 	for p.l.checkLook(tknBXOr) {
// 		p.l.getCurrent(tknBXOr)
// 		line := p.l.current.Line
// 		l = exprLine(&Operator{Op: OpBinXOR, Left: l, Right: p.valBAnd()}, line)
// 	}
// 	return l
// }

// func (p *parser) valBAnd() Expr {
// 	l := p.valShift()
// 	for p.l.checkLook(tknBAnd) {
// 		p.l.getCurrent(tknBAnd)
// 		line := p.l.current.Line
// 		l = exprLine(&Operator{Op: OpBinAND, Left: l, Right: p.valShift()}, line)
// 	}
// 	return l
// }

// func (p *parser) valShift() Expr {
// 	l := p.valConcat()
// 	for p.l.checkLook(tknShiftL, tknShiftR) {
// 		p.l.getCurrent(tknShiftL, tknShiftR)
// 		line := p.l.current.Line
// 		switch p.l.current.Type {
// 		case tknShiftL:
// 			l = exprLine(&Operator{Op: OpBinShiftL, Left: l, Right: p.valConcat()}, line)
// 		case tknShiftR:
// 			l = exprLine(&Operator{Op: OpBinShiftR, Left: l, Right: p.valConcat()}, line)
// 		}
// 	}
// 	return l
// }

// func (p *parser) valConcat() Expr {
// 	l := p.valAdd()
// 	// No loop!
// 	if p.l.checkLook(tknConcat) {
// 		p.l.getCurrent(tknConcat)
// 		line := p.l.current.Line
// 		// I... Think?
// 		// This would have the effect of treating the remainder of the expression like it was in
// 		// parenthesis, which (if I am thinking correctly) is basically what right associative is...
// 		//return exprLine(&Operator{Op: OpConcat, Left: l, Right: p.expression()}, line)

// 		// Apparently not, maybe this?
// 		return exprLine(&Operator{Op: OpConcat, Left: l, Right: p.valConcat()}, line)
// 	}
// 	return l
// }

// func (p *parser) valAdd() Expr {
// 	l := p.valMul()
// 	for p.l.checkLook(tknAdd, tknSub) {
// 		p.l.getCurrent(tknAdd, tknSub)
// 		line := p.l.current.Line
// 		switch p.l.current.Type {
// 		case tknAdd:
// 			l = exprLine(&Operator{Op: OpAdd, Left: l, Right: p.valMul()}, line)
// 		case tknSub:
// 			l = exprLine(&Operator{Op: OpSub, Left: l, Right: p.valMul()}, line)
// 		}
// 	}
// 	return l
// }

// func (p *parser) valMul() Expr {
// 	l := p.valUnOp()
// 	for p.l.checkLook(tknMul, tknDiv, tknIDiv, tknMod) {
// 		p.l.getCurrent(tknMul, tknDiv, tknIDiv, tknMod)
// 		line := p.l.current.Line
// 		switch p.l.current.Type {
// 		case tknMul:
// 			l = exprLine(&Operator{Op: OpMul, Left: l, Right: p.valUnOp()}, line)
// 		case tknDiv:
// 			l = exprLine(&Operator{Op: OpDiv, Left: l, Right: p.valUnOp()}, line)
// 		case tknIDiv:
// 			l = exprLine(&Operator{Op: OpIDiv, Left: l, Right: p.valUnOp()}, line)
// 		case tknMod:
// 			l = exprLine(&Operator{Op: OpMod, Left: l, Right: p.valUnOp()}, line)
// 		}
// 	}
// 	return l
// }

// func (p *parser) valUnOp() Expr {
// 	switch p.l.look.Type {
// 	case tknNot:
// 		p.l.getCurrent(tknNot)
// 		line := p.l.current.Line
// 		return exprLine(&Operator{Op: OpNot, Right: p.valUnOp()}, line)
// 	case tknLen:
// 		p.l.getCurrent(tknLen)
// 		line := p.l.current.Line
// 		return exprLine(&Operator{Op: OpLength, Right: p.valUnOp()}, line)
// 	case tknBXOr:
// 		p.l.getCurrent(tknBXOr)
// 		line := p.l.current.Line
// 		return exprLine(&Operator{Op: OpBinNot, Right: p.valUnOp()}, line)
// 	case tknSub:
// 		p.l.getCurrent(tknSub)
// 		line := p.l.current.Line
// 		return exprLine(&Operator{Op: OpUMinus, Right: p.valUnOp()}, line)
// 	default:
// 		return p.valPow()
// 	}
// }

// func (p *parser) valPow() Expr {
// 	l := p.value()
// 	// No loop!
// 	if p.l.checkLook(tknPow) {
// 		p.l.getCurrent(tknPow)
// 		line := p.l.current.Line
// 		// See valConcat.
// 		return exprLine(&Operator{Op: OpPow, Left: l, Right: p.valPow()}, line)
// 	}
// 	return l
// }

// float | int | string | nil | true | false | ... | table constructor | function call | varValue
func (p *parser) value() Expr {
	switch p.l.look.Type {
	case tknOBracket:
		return p.tblConstruct()
	case tknFunction:
		p.l.getCurrent(tknFunction)
		return p.funcDeclBody(false)
	case tknTrue:
		p.l.getCurrent(tknTrue)
		return exprLine(&ConstBool{Value: true}, p.l.current.Line)
	case tknFalse:
		p.l.getCurrent(tknFalse)
		return exprLine(&ConstBool{Value: false}, p.l.current.Line)
	case tknNil:
		p.l.getCurrent(tknNil)
		return exprLine(&ConstNil{}, p.l.current.Line)
	case tknVariadic:
		p.l.getCurrent(tknVariadic)
		return exprLine(&ConstVariadic{}, p.l.current.Line)
	case tknInt:
		p.l.getCurrent(tknInt)
		return exprLine(&ConstInt{Value: p.l.current.Lexeme}, p.l.current.Line)
	case tknFloat:
		p.l.getCurrent(tknFloat)
		return exprLine(&ConstFloat{Value: p.l.current.Lexeme}, p.l.current.Line)
	case tknString:
		p.l.getCurrent(tknString)
		return exprLine(&ConstString{Value: p.l.current.Lexeme}, p.l.current.Line)
	default:
		return p.suffixedValue()
	}
}

// suffixedValue -> primaryValue { '.' ident | '[' exp ']' | ':' ident funcargs | funcargs }
func (p *parser) suffixedValue() Expr {
	l := p.primaryValue()
	for p.l.checkLook(tknOIndex, tknDot, tknColon, tknOParen, tknString, tknOBracket) {
		switch p.l.look.Type {
		case tknOIndex: // [expr]
			p.l.getCurrent(tknOIndex)

			line := p.l.current.Line
			l = exprLine(&TableAccessor{
				Obj: l,
				Key: p.expression(),
			}, line)

			p.l.getCurrent(tknCIndex)
		case tknDot: // .ident or .ident() or .ident:ident()
			p.l.getCurrent(tknDot)
			line := p.l.current.Line
			p.l.getCurrent(tknName)
			if p.l.checkLook(tknColon, tknOParen) {
				l = p.funcCall(exprLine(&TableAccessor{
					Obj: l,
					Key: exprLine(&ConstString{
						Value: p.l.current.Lexeme,
					}, p.l.current.Line),
				}, line))
			} else {
				l = exprLine(&TableAccessor{
					Obj: l,
					Key: exprLine(&ConstString{
						Value: p.l.current.Lexeme,
					}, p.l.current.Line),
				}, line)
			}
		case tknColon, tknOParen, tknString, tknOBracket:
			l = p.funcCall(l)
		}
	}
	return l
}

// primaryValue -> ident | '(' expr ')'
func (p *parser) primaryValue() Expr {
	switch p.l.look.Type {
	case tknName:
		p.l.getCurrent(tknName)
		return exprLine(&ConstIdent{
			Value: p.l.current.Lexeme,
		}, p.l.current.Line)
	case tknOParen:
		p.l.getCurrent(tknOParen)

		line := p.l.current.Line
		l := exprLine(&Parens{
			Inner: p.expression(),
		}, line)

		p.l.getCurrent(tknCParen)
		return l
	default:
		p.l.getCurrent(tknName, tknOParen)
		panic("UNREACHABLE")
	}
}
