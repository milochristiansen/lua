/*
Copyright 2016 by Milo Christiansen

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

package lua

import "github.com/milochristiansen/lua/ast"
import "github.com/milochristiansen/lua/luautil"
import "sliceutil" // a quick-and-dirty reflection-based slice library (I use it to make stacks, lazy me).

// TODO: It is possible to have more constants than can be fit into an Bx field, in which case a
// LOADKX instruction should be used.
// In the same vein it is possible to overflow a RK as well, in which case an extra LOADK or LOADKX is needed.

// returns: (0: local, 1: global, 2: upvalue), index
func resolveVar(v string, state *compState) (int, int) {
	idx, local := resolveVarHelper(v, state)
	if local {
		return 0, idx
	} else if idx != -1 {
		return 2, idx
	} else {
		// assume it is a global
		return 1, 0
	}
}

func resolveVarHelper(v string, state *compState) (int, bool) {
	// Try to find an in-scope local first
	for i := len(state.f.localVars) - 1; i >= 0; i-- {
		l := state.f.localVars[i]
		if l.sPC > l.ePC && l.name == v {
			return state.locals[i], true
		}
	}
	// If that fails look for an upvalue
	for i, up := range state.f.upVals {
		if up.name == v {
			return i, false
		}
	}

	if state.p != nil {
		idx, local := resolveVarHelper(v, state.p)
		if idx == -1 {
			return -1, false
		}
		nidx := len(state.f.upVals)
		state.f.upVals = append(state.f.upVals, upDef{
			isLocal: local,
			name:    v,
			index:   idx,
		})
		return nidx, false
	}
	return -1, false
}

type identData struct {
	isUp    bool // If true item is stored in an upval, else a register
	isTable bool // if true the item is a table and keyRK is valid

	state *compState
	reg   int
	line  int

	itemIdx int // The register or upvalue index where the item resides
	keyRK   int // The RK of the table index if needed (isTable is true)
}

func (data identData) Set(sourceRK int) {
	state := data.state
	switch {
	case data.isTable && data.isUp:
		state.addInst(createABC(opSetTableUp, data.itemIdx, data.keyRK, sourceRK), data.line)
	case data.isTable && !data.isUp:
		state.addInst(createABC(opSetTable, data.itemIdx, data.keyRK, sourceRK), data.line)
	case !data.isTable && !data.isUp:
		if sourceRK == data.itemIdx {
			return
		}
		if isK(sourceRK) {
			state.addInst(createABx(opLoadK, data.itemIdx, indexK(sourceRK)), data.line)
			return
		}
		state.addInst(createABC(opMove, data.itemIdx, sourceRK, 0), data.line)
	case !data.isTable && data.isUp:
		// SETUPVAL is inconsistent with just about every other instruction.
		state.addInst(createABC(opSetUpValue, sourceRK, data.itemIdx, 0), data.line)
	default:
		panic("IMPOSSIBLE")
	}
}

// if tryInPlace then instead of moving the item it will be left in place and this will return "true, <index>"
func (data identData) Get(dest int, tryInPlace bool) (bool, int) {
	state := data.state
	switch {
	case data.isTable && data.isUp:
		state.addInst(createABC(opGetTableUp, dest, data.itemIdx, data.keyRK), data.line)
		return false, 0
	case data.isTable && !data.isUp:
		state.addInst(createABC(opGetTable, dest, data.itemIdx, data.keyRK), data.line)
		return false, 0
	case !data.isTable && !data.isUp:
		if dest == data.itemIdx {
			return false, 0
		}
		if tryInPlace {
			return true, data.itemIdx
		}
		state.addInst(createABC(opMove, dest, data.itemIdx, 0), data.line)
		return false, 0
	case !data.isTable && data.isUp:
		state.addInst(createABC(opGetUpValue, dest, data.itemIdx, 0), data.line)
		return false, 0
	default:
		panic("IMPOSSIBLE")
	}
}

// Reduce an identifier to its lowest possible form (aka resolve all table accesses but the last).
// Registers starting from reg may be used to store temporaries, if so the number used will be returned.
// This assumes values above reg will be available as a place to store expression results.
func lowerIdent(n ast.Expr, state *compState, reg int) (identData, int) {
	data := &identData{
		state:   state,
		reg:     reg,
		line:    n.Line(),
		itemIdx: reg, // <- possibly not the final value!
	}

	switch nn := n.(type) {
	case *ast.TableAccessor:
		data.isTable = true
		switch nObj := nn.Obj.(type) {
		case *ast.TableAccessor:
			lowerIdentHelper(nObj, state, data)
			data.keyRK, _ = expr(nn.Key, state, reg+1, false).RK()
			return *data, 1
		case *ast.ConstIdent:
			typ, idx := resolveVar(nObj.Value, state)
			switch typ {
			case 0:
				data.itemIdx = idx
			case 1:
				etyp, eidx := resolveVar("_ENV", state)
				if etyp == 0 {
					state.addInst(createABC(opGetTable, reg, eidx, state.constRK(nObj.Value)), nObj.Line())
				} else {
					//state.addInst(createABC(opGetTableUp, reg, 0 /*_ENV*/, state.constRK(nObj.Value)), nObj.Line())
					state.addInst(createABC(opGetTableUp, reg, eidx, state.constRK(nObj.Value)), nObj.Line())
				}
				idx = reg
			case 2:
				data.itemIdx = idx
				data.isUp = true
			}
			regs, keyReg := 2, reg+1
			if idx != reg {
				keyReg = reg
				regs = 1
			}
			data.keyRK, _ = expr(nn.Key, state, keyReg, false).RK()
			return *data, regs
		case *ast.Parens:
			expr(nObj.Inner, state, data.reg, false).To(false)
			data.keyRK, _ = expr(nn.Key, state, reg+1, false).RK()
			return *data, 1
		case *ast.FuncCall:
			expr(nObj, state, data.reg, false).To(false)
			data.keyRK, _ = expr(nn.Key, state, reg+1, false).RK()
			return *data, 1
		default:
			luautil.Raise("Syntax error", luautil.ErrTypGenSyntax) // TODO: Better errors
		}
		panic("UNREACHABLE")
	case *ast.ConstIdent:
		typ, idx := resolveVar(nn.Value, state)
		switch typ {
		case 0: // local
			data.itemIdx = idx
		case 1: // global (_ENV.<ident>)
			etyp, eidx := resolveVar("_ENV", state)
			if etyp == 0 {
				data.itemIdx = eidx
				data.isUp = false
			} else {
				//data.itemIdx = 0
				data.itemIdx = eidx
				data.isUp = true
			}
			data.isTable = true
			data.keyRK = state.constRK(nn.Value)
		case 2: // upvalue
			data.itemIdx = idx
			data.isUp = true
		}
		return *data, 0
	default:
		panic("IMPOSSIBLE") // I think?
	}
}

func lowerIdentHelper(n *ast.TableAccessor, state *compState, data *identData) {
	switch nObj := n.Obj.(type) {
	case *ast.TableAccessor:
		lowerIdentHelper(nObj, state, data)
		rk, _ := expr(n.Key, state, data.reg+1, false).RK()
		state.addInst(createABC(opGetTable, data.reg, data.reg, rk), n.Key.Line())
	case *ast.ConstIdent:
		typ, idx := resolveVar(nObj.Value, state)
		switch typ {
		case 0:
			rk, _ := expr(n.Key, state, data.reg+1, false).RK()
			state.addInst(createABC(opGetTable, data.reg, idx, rk), n.Key.Line())
		case 1:
			etyp, eidx := resolveVar("_ENV", state)
			if etyp == 0 {
				state.addInst(createABC(opGetTable, data.reg, eidx, state.constRK(nObj.Value)), nObj.Line())
			} else {
				//state.addInst(createABC(opGetTableUp, data.reg, 0 /*_ENV*/, state.constRK(nObj.Value)), nObj.Line())
				state.addInst(createABC(opGetTableUp, data.reg, eidx, state.constRK(nObj.Value)), nObj.Line())
			}
			rk, _ := expr(n.Key, state, data.reg+1, false).RK()
			state.f.code = append(state.f.code, createABC(opGetTable, data.reg, data.reg, rk))
		case 2:
			rk, _ := expr(n.Key, state, data.reg+1, false).RK()
			state.addInst(createABC(opGetTableUp, data.reg, idx, rk), n.Key.Line())
		}
	case *ast.Parens:
		expr(nObj.Inner, state, data.reg, false).To(false)
	case *ast.FuncCall:
		expr(nObj, state, data.reg, false).To(false)
	default:
		panic("IMPOSSIBLE") // I think?
	}
}

func compileCall(call *ast.FuncCall, state *compState, reg, rets int, tail bool) {
	f := reg
	reg++
	params := 0
	if call.Receiver != nil {
		src, _ := expr(call.Receiver, state, f, false).To(true)
		rk, _ := expr(call.Function, state, reg, false).RK()
		state.addInst(createABC(opSelf, f, src, rk), call.Receiver.Line())
		params++
		reg++
	} else {
		expr(call.Function, state, f, false).To(false)
	}

	for i, e := range call.Args {
		exres := expr(e, state, reg, false)
		if i == len(call.Args)-1 && exres.mayMulti {
			exres.setResults(-1)
			params = -2
		}
		exres.To(false)
		reg++
		params++
	}

	if tail {
		state.addInst(createABC(opTailCall, f, params+1, 0), call.Line())
		return
	}
	state.addInst(createABC(opCall, f, params+1, rets+1), call.Line())
}

type exprData struct {
	// If non-nil this expression was a boolean expression, and as such does not produce
	// an actual value without some extra code
	boolean  patchList
	boolRev  bool // If true jump on true instead of false.
	boolCanR bool // If true this boolean expression also sets reg (the value in reg may not be boolean)

	// If true the expression resulted in a value placed in the provided register
	register bool

	// If the expression did not result in a register value and it was not a boolean
	// expression this will be set to a constant index that holds the state.f.
	// I really should use LOADBOOL and LOADNIL where possible, but this way is simpler.
	// TODO: Issue LOADKX instructions where needed!
	constant int

	// True if expression is a single function call or VARARG.
	mayMulti   bool
	patchMulti int // MUST be a CALL or VARARG

	state *compState
	oreg  int
	reg   int
	line  int
}

// -1 for unlimited.
// 0 does nothing if mayMulti is false
// if mayMulti is false items above 1 are taken care of via an inserted LOADNIL
func (e exprData) setResults(c int) {
	state := e.state
	if !e.mayMulti {
		if c <= 1 {
			return
		}
		state.addInst(createABC(opLoadNil, e.reg+1, c-2, 0), e.line)
		return
	}
	if state.f.code[e.patchMulti].getOpCode() == opCall {
		state.f.code[e.patchMulti].setC(c + 1)
	} else {
		state.f.code[e.patchMulti].setB(c + 1)
	}
}

// result register, was the requested register used?
func (e exprData) To(tryInPlace bool) (int, bool) {
	state := e.state
	switch {
	case e.register:
		if e.reg != e.oreg {
			if tryInPlace {
				return e.reg, false
			}
			state.addInst(createABC(opMove, e.oreg, e.reg, 0), e.line)
			return e.oreg, true
		}
		return e.reg, true
	case e.boolCanR:
		e.boolean.patch(state.f, len(state.f.code))
		return e.reg, true
	case e.boolean != nil:
		if e.boolRev {
			state.addInst(createABC(opLoadBool, e.reg, 0, 1), e.line)
			e.boolean.patch(state.f, len(state.f.code))
			state.addInst(createABC(opLoadBool, e.reg, 1, 0), e.line)
		} else {
			state.addInst(createABC(opLoadBool, e.reg, 1, 1), e.line)
			e.boolean.patch(state.f, len(state.f.code))
			state.addInst(createABC(opLoadBool, e.reg, 0, 0), e.line)
		}
		return e.reg, true
	default:
		state.addInst(createABx(opLoadK, e.reg, e.constant), e.line)
		return e.reg, true
	}
}

// RK of result, was the requested register used?
func (e exprData) RK() (int, bool) {
	state := e.state
	switch {
	case e.register:
		if e.reg != e.oreg {
			return e.reg, false
		}
		return e.reg, true
	case e.boolCanR:
		e.boolean.patch(state.f, len(state.f.code))
		return e.reg, true
	case e.boolean != nil:
		if e.boolRev {
			state.addInst(createABC(opLoadBool, e.reg, 0, 1), e.line)
			e.boolean.patch(state.f, len(state.f.code))
			state.addInst(createABC(opLoadBool, e.reg, 1, 0), e.line)
		} else {
			state.addInst(createABC(opLoadBool, e.reg, 1, 1), e.line)
			e.boolean.patch(state.f, len(state.f.code))
			state.addInst(createABC(opLoadBool, e.reg, 0, 0), e.line)
		}
		return e.reg, true
	default:
		return rkAsK(e.constant), false
	}
}

// The caller never needs to know if boolRev is set (since that always comes from the caller they already know)
func (e exprData) Bool() (patchList, bool) {
	state := e.state
	switch {
	case e.register:
		state.addInst(createABC(opTest, e.reg, 0, 0), e.line)
		f := patchList([]int{len(state.f.code)})
		state.addInst(createAsBx(opJump, 0, 0), e.line)
		return f, false
	case e.boolean != nil:
		return e.boolean, false
	default:
		return nil, toBool(state.f.constants[e.constant])
	}
}

// patch, reg ok, const (only if patch == nil, reg ok is always true in this case)
func (e exprData) BoolReg() (patchList, bool, bool) {
	state := e.state
	switch {
	case e.register:
		if e.boolRev {
			state.addInst(createABC(opTest, e.reg, 0, 1), e.line)
			f := patchList([]int{len(state.f.code)})
			state.addInst(createAsBx(opJump, 0, 0), e.line)
			return f, true, false
		}
		state.addInst(createABC(opTest, e.reg, 0, 0), e.line)
		f := patchList([]int{len(state.f.code)})
		state.addInst(createAsBx(opJump, 0, 0), e.line)
		return f, true, false
	case e.boolean != nil:
		return e.boolean, false, false
	default:
		state.addInst(createABx(opLoadK, e.reg, e.constant), e.line)
		return nil, true, toBool(state.f.constants[e.constant])
	}
}

// Handle an expression.
// Assumes that the items above reg are available to use as temporaries.
// boolRev reverses the sense of boolean operators (true jumps and false falls through).
func expr(e ast.Expr, state *compState, reg int, boolRev bool) exprData {
	rtn := exprData{
		state:   state,
		boolRev: boolRev,
		reg:     reg,
		oreg:    reg,
		line:    e.Line(),
	}

	switch ee := e.(type) {
	case *ast.Operator:
		// Operator precedence is already handled by the AST, Yay!
		switch ee.Op {
		// Simple binary operators
		case ast.OpAdd, ast.OpSub, ast.OpMul, ast.OpMod, ast.OpPow, ast.OpDiv, ast.OpIDiv, ast.OpBinAND, ast.OpBinOR, ast.OpBinXOR, ast.OpBinShiftL, ast.OpBinShiftR:
			// TODO: Constant folding
			l, lu := expr(ee.Left, state, reg, false).RK()
			r := reg
			if lu {
				r++
			}
			r, _ = expr(ee.Right, state, r, false).RK()
			state.addInst(createABC(opCode(ee.Op)+OpAdd, reg, l, r), ee.Line())
			rtn.register = true

		// Simple unary operators
		case ast.OpUMinus, ast.OpBinNot, ast.OpNot, ast.OpLength:
			// TODO: Constant folding for OpUMinus and OpBinNot
			v, _ := expr(ee.Right, state, reg, false).RK()
			state.addInst(createABC(opCode(ee.Op)+OpAdd, reg, v, 0), ee.Line())
			rtn.register = true

		// Complex binary operators

		case ast.OpConcat:
			// Combine multiple adjacent concat operators into one operation.
			// This is important to string concatenation performance.
			last := reg
			en, een := e, ee
			ok := true
			for ok && een.Op == ast.OpConcat {
				expr(een.Left, state, last, false).To(false)
				last++
				en = een.Right
				een, ok = en.(*ast.Operator)
			}
			expr(en, state, last, false).To(false)

			state.addInst(createABC(opConcat, reg, reg, last), ee.Line())
			rtn.register = true

		// Simple Logical operators

		case ast.OpEqual:
			sense := 0
			if boolRev {
				sense = 1
			}
			l, lu := expr(ee.Left, state, reg, false).RK()
			r := reg
			if lu {
				r++
			}
			r, _ = expr(ee.Right, state, r, false).RK()
			state.addInst(createABC(OpEqual, sense, l, r), ee.Line())
			rtn.boolean = patchList([]int{len(state.f.code)})
			state.addInst(createAsBx(opJump, 0, 0), ee.Line())
		case ast.OpNotEqual:
			sense := 1
			if boolRev {
				sense = 0
			}
			l, lu := expr(ee.Left, state, reg, false).RK()
			r := reg
			if lu {
				r++
			}
			r, _ = expr(ee.Right, state, r, false).RK()
			state.addInst(createABC(OpEqual, sense, l, r), ee.Line())
			rtn.boolean = patchList([]int{len(state.f.code)})
			state.addInst(createAsBx(opJump, 0, 0), ee.Line())
		case ast.OpLessThan:
			sense := 0
			if boolRev {
				sense = 1
			}
			l, lu := expr(ee.Left, state, reg, false).RK()
			r := reg
			if lu {
				r++
			}
			r, _ = expr(ee.Right, state, r, false).RK()
			state.addInst(createABC(OpLessThan, sense, l, r), ee.Line())
			rtn.boolean = patchList([]int{len(state.f.code)})
			state.addInst(createAsBx(opJump, 0, 0), ee.Line())
		case ast.OpLessOrEqual:
			sense := 0
			if boolRev {
				sense = 1
			}
			l, lu := expr(ee.Left, state, reg, false).RK()
			r := reg
			if lu {
				r++
			}
			r, _ = expr(ee.Right, state, r, false).RK()
			state.addInst(createABC(OpLessOrEqual, sense, l, r), ee.Line())
			rtn.boolean = patchList([]int{len(state.f.code)})
			state.addInst(createAsBx(opJump, 0, 0), ee.Line())
		case ast.OpGreaterThan:
			sense := 1
			if boolRev {
				sense = 0
			}
			l, lu := expr(ee.Left, state, reg, false).RK()
			r := reg
			if lu {
				r++
			}
			r, _ = expr(ee.Right, state, r, false).RK()
			state.addInst(createABC(OpLessOrEqual, sense, l, r), ee.Line())
			rtn.boolean = patchList([]int{len(state.f.code)})
			state.addInst(createAsBx(opJump, 0, 0), ee.Line())
		case ast.OpGreaterOrEqual:
			sense := 1
			if boolRev {
				sense = 0
			}
			l, lu := expr(ee.Left, state, reg, false).RK()
			r := reg
			if lu {
				r++
			}
			r, _ = expr(ee.Right, state, r, false).RK()
			state.addInst(createABC(OpLessThan, sense, l, r), ee.Line())
			rtn.boolean = patchList([]int{len(state.f.code)})
			state.addInst(createAsBx(opJump, 0, 0), ee.Line())

		// The pain in the a** operators
		// TODO: The code generated here is *terrible*.

		case ast.OpAnd:
			//	Get the left value into a register (any register)
			//	if left == reg
			//		TEST reg _ 1
			//		JMP <end> // on false
			//	else
			//		TESTSET reg left 1
			//		JMP <end> // on false
			//	Get the right value into any register
			//	if right == reg
			//		TEST reg _ 1
			//		JMP <end> // on false
			//	else
			//		TESTSET reg right 1
			//		JMP <end> // on false
			//	if jump on true
			//		JMP <target>
			//		<end>
			//	else
			//		make all "JMP <end>" sequences above into "JMP <target>"

			rtn.boolCanR = true
			patch := patchList([]int{})
			lr, lru := expr(ee.Left, state, reg, false).To(true)
			if lru {
				state.addInst(createABC(opTest, reg, 0, 0), ee.Left.Line()) // jump on false
				patch = append(patch, len(state.f.code))
				state.addInst(createAsBx(opJump, 0, 0), ee.Left.Line())
			} else {
				state.addInst(createABC(opTestSet, reg, lr, 0), ee.Left.Line()) // jump on false
				patch = append(patch, len(state.f.code))
				state.addInst(createAsBx(opJump, 0, 0), ee.Left.Line())
			}
			rr, rru := expr(ee.Right, state, reg, false).To(true)
			if rru {
				state.addInst(createABC(opTest, reg, 0, 0), ee.Right.Line()) // jump on false
				patch = append(patch, len(state.f.code))
				state.addInst(createAsBx(opJump, 0, 0), ee.Right.Line())
			} else {
				state.addInst(createABC(opTestSet, reg, rr, 0), ee.Right.Line()) // jump on false
				patch = append(patch, len(state.f.code))
				state.addInst(createAsBx(opJump, 0, 0), ee.Right.Line())
			}
			if boolRev {
				rtn.boolean = patchList([]int{len(state.f.code)})
				state.addInst(createAsBx(opJump, 0, 0), ee.Line())
				patch.patch(state.f, len(state.f.code))
				return rtn
			}
			rtn.boolean = patch
			return rtn
		case ast.OpOr:
			// TESTSET dest src sense ; if bool(src) == sense { sest = src } else { pc++ }
			// TEST val _ sense ; if bool(val) == sense { } else { pc++ }

			//	Get the left value into a register (any register)
			//	if left == reg
			//		TEST reg _ 1
			//		JMP <end> // on true
			//	else
			//		TESTSET reg left 1
			//		JMP <end> // on true
			//	Get the right value into any register
			//	if right == reg
			//		TEST reg _ 1
			//		JMP <end> // on true
			//	else
			//		TESTSET reg right 1
			//		JMP <end> // on true
			//	if jump on true
			//		make all "JMP <end>" sequences above into "JMP <target>"
			//	else
			//		JMP <target>
			//		<end>

			rtn.boolCanR = true
			patch := patchList([]int{})
			lr, lru := expr(ee.Left, state, reg, false).To(true)
			if lru {
				state.addInst(createABC(opTest, reg, 0, 1), ee.Left.Line()) // jump on true
				patch = append(patch, len(state.f.code))
				state.addInst(createAsBx(opJump, 0, 0), ee.Left.Line())
			} else {
				state.addInst(createABC(opTestSet, reg, lr, 1), ee.Left.Line()) // jump on true
				patch = append(patch, len(state.f.code))
				state.addInst(createAsBx(opJump, 0, 0), ee.Left.Line())
			}
			rr, rru := expr(ee.Right, state, reg, false).To(true)
			if rru {
				state.addInst(createABC(opTest, reg, 0, 1), ee.Right.Line()) // jump on true
				patch = append(patch, len(state.f.code))
				state.addInst(createAsBx(opJump, 0, 0), ee.Right.Line())
			} else {
				state.addInst(createABC(opTestSet, reg, rr, 1), ee.Right.Line()) // jump on true
				patch = append(patch, len(state.f.code))
				state.addInst(createAsBx(opJump, 0, 0), ee.Right.Line())
			}
			if boolRev {
				rtn.boolean = patch
				return rtn
			}
			rtn.boolean = patchList([]int{len(state.f.code)})
			state.addInst(createAsBx(opJump, 0, 0), ee.Line())
			patch.patch(state.f, len(state.f.code))
			return rtn
		}
	case *ast.FuncCall:
		compileCall(ee, state, reg, 1, false)
		rtn.mayMulti = true
		rtn.patchMulti = len(state.f.code) - 1
		rtn.register = true
	case *ast.FuncDecl:
		f := compile(ee, state)
		fi := len(state.f.prototypes)
		state.f.prototypes = append(state.f.prototypes, *f)
		sliceutil.Top(&state.blocks).(*blockStuff).hasUp = true // Possibly not, but better lazy than sorry
		state.addInst(createABx(opClosure, reg, fi), ee.Line())
		rtn.register = true
	case *ast.TableConstructor:
		keys := []ast.Expr{}
		keyed := []ast.Expr{}
		list := []ast.Expr{}
		for i, k := range ee.Keys {
			if k == nil {
				list = append(list, ee.Vals[i])
				continue
			}
			keys = append(keys, k)
			keyed = append(keyed, ee.Vals[i])
		}

		state.addInst(createABC(opNewTable, reg, int(float8FromInt(len(list))), int(float8FromInt(len(keys)))), ee.Line())

		ic := 0
		fc := 1
		for i, item := range list {
			if ic == 50 {
				ic = 0
				state.addInst(createABC(opSetList, reg, 50, fc), ee.Line())
				fc++
			}
			ex := expr(item, state, reg+ic+1, false)
			if i == len(list)-1 && ex.mayMulti {
				ex.setResults(-1)
				state.addInst(createABC(opSetList, reg, 0, fc), ee.Line())
				ic = -1
			}
			ex.To(false)
			ic++
		}
		if ic != 0 {
			state.addInst(createABC(opSetList, reg, ic, fc), ee.Line())
		}

		for i, item := range keyed {
			vrk, _ := expr(item, state, reg+1, false).RK()
			krk, _ := expr(keys[i], state, reg+2, false).RK()
			state.addInst(createABC(opSetTable, reg, krk, vrk), ee.Line())
		}
		rtn.register = true
	case *ast.TableAccessor:
		ident, _ := lowerIdent(e, state, reg)
		place, idx := ident.Get(reg, true)
		rtn.register = true
		if place {
			rtn.reg = idx
		}
	case *ast.Parens:
		switch eee := ee.Inner.(type) {
		case *ast.FuncCall:
			compileCall(eee, state, reg, 1, false)
			rtn.register = true
		case *ast.ConstVariadic:
			state.f.isVarArg = 1
			state.addInst(createABC(opVarArg, reg, 2, 0), eee.Line())
			rtn.register = true
		default:
			ex := expr(ee.Inner, state, reg, boolRev)
			return ex
		}
	case *ast.ConstInt:
		rtn.constant = state.constK(toInt(ee.Value))
	case *ast.ConstFloat:
		rtn.constant = state.constK(toFloat(ee.Value))
	case *ast.ConstString:
		rtn.constant = state.constK(ee.Value)
	case *ast.ConstIdent:
		ident, _ := lowerIdent(e, state, reg)
		place, idx := ident.Get(reg, true)
		rtn.register = true
		if place {
			rtn.reg = idx
		}
	case *ast.ConstBool:
		rtn.constant = state.constK(ee.Value == true)
	case *ast.ConstNil:
		rtn.constant = state.constK(nil)
	case *ast.ConstVariadic:
		state.f.isVarArg = 1
		rtn.mayMulti = true
		rtn.patchMulti = len(state.f.code)
		state.addInst(createABC(opVarArg, reg, 2, 0), ee.Line())
		rtn.register = true
	}
	return rtn
}

// Handle a list of expressions, always reads to registers.
// If "len(es) < minresults" then the last expression is expected to provide as many items as needed if
// possible, else the remainder are filled with nil values. This may result in registers above "firstreg+minresults"
// being filled! Normally this is not a problem, as the extra values are simply overwritten when these
// registers are needed (now you see what LOADNIL is for!).
func exprlist(es []ast.Expr, state *compState, firstreg, minresults int) {
	// This is really simple
	var last exprData
	for _, e := range es {
		last = expr(e, state, firstreg, false)
		firstreg++
		minresults--
		last.To(false)
	}
	if minresults > 0 {
		last.setResults(minresults + 1)
	}
}
