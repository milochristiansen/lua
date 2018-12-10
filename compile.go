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

package lua

import "github.com/milochristiansen/lua/ast"
import "github.com/milochristiansen/lua/luautil"
import "fmt"

//import "runtime"

// TODO: Error messages are horrid and unhelpful. The AST is just sitting there, it should be possible to
// turn that information into an error message that is VERY helpful and has every detail you could ever want...

func mkoffset(from, to int) int {
	return to - from - 1 // -1 to correct for the automatic +1 to the PC after each instruction.
}

type compState struct {
	p *compState
	f *funcProto

	nextReg   int
	breaks    []patchList
	continues []patchList
	blocks    []*blockStuff
	locals    []int // local index -> register
}

type localPatchList []int

// Set the jump targets in the patchList to the given PC.
func (p localPatchList) patch(f *funcProto, soff int) {
	for _, l := range p {
		f.localVars[l].sPC = int32(len(f.code) + soff)
	}
}

// This is to help me remember to add line info for each instruction...
func (state *compState) addInst(inst instruction, line int) {
	state.f.lineInfo = append(state.f.lineInfo, line)
	state.f.code = append(state.f.code, inst)
}

func (state *compState) mklocal(name string, soff int) {
	state.locals = append(state.locals, state.nextReg)
	state.nextReg++
	state.f.localVars = append(state.f.localVars, localVar{
		name: name,
		sPC:  int32(len(state.f.code) + soff),
		ePC:  int32(state.blocks[len(state.blocks)-1].bpc),
	})
}

func (state *compState) mklocaladv(name string, p localPatchList) localPatchList {
	state.locals = append(state.locals, state.nextReg)
	state.nextReg++
	l := len(state.f.localVars)
	state.f.localVars = append(state.f.localVars, localVar{
		name: name,
		sPC:  -500, // magic, no special significance except it isn't any of the other magic values and is lower than any valid value
		ePC:  int32(state.blocks[len(state.blocks)-1].bpc),
	})
	return append(p, l)
}

// Returns a valid RK for the given constant.
// May add a new instruction in case of overflow. reg may be used as a temporary.
// val MUST be an int64, float64, bool, nil, or string!
func (state *compState) constRK(val value, reg, line int) (int, bool) {
	k := state.constK(val)
	if k > maxIndexRK {
		state.addInst(createABx(opLoadK, reg, k), line)
		return reg, true
	}
	return rkAsK(k), false
}

// Returns a valid index for the given constant.
// val MUST be an int64, float64, bool, nil, or string!
func (state *compState) constK(val value) int {
	for i, v := range state.f.constants {
		if val == v {
			return i
		}
	}
	at := len(state.f.constants)
	state.f.constants = append(state.f.constants, val)
	return at
}

type jumpDat struct {
	label string
	pc    int
	regs  int
	line  int
}

func (from jumpDat) patch(f *funcProto, to jumpDat) {
	if from.regs < to.regs {
		luautil.Raise(fmt.Sprintf("Unconditional jump on line %v (to line %v) into the scope of one or more local variables", from.line, to.line), luautil.ErrTypGenSyntax) // TODO: Better errors
	}

	f.code[from.pc].setSBx(mkoffset(from.pc, to.pc))
}

type blockStuff struct {
	bpc int

	labels []jumpDat
	gotos  map[string][]jumpDat

	hasUp bool // One or more locals in this block are used as upvalues
}

type patchList []int

// Set the jump targets in the patchList to the given PC.
func (p patchList) patch(f *funcProto, pc int) {
	for _, ipc := range p {
		f.code[ipc].setSBx(mkoffset(ipc, pc)) // -1 to correct for the automatic +1 to the PC after each instruction.
	}
}

// Patch the A field of the JMPs to close values at "reg" and above in addition to the normal PC patching.
func (p patchList) loop(f *funcProto, pc, reg int) {
	for _, ipc := range p {
		f.code[ipc].setA(reg)
		f.code[ipc].setSBx(mkoffset(ipc, pc))
	}
}

func compSource(source, name string, line int) (f *funcProto, err error) {
	// Quick-and-dirty error trapping.
	defer func() {
		if x := recover(); x != nil {
			//fmt.Println("Stack Trace:")
			//buf := make([]byte, 4096)
			//buf = buf[:runtime.Stack(buf, true)]
			//fmt.Printf("%s\n", buf)

			switch e := x.(type) {
			case luautil.Error:
				err = e
			case error:
				err = &luautil.Error{Err: e, Type: luautil.ErrTypWrapped}
			default:
				err = &luautil.Error{Msg: fmt.Sprint(x), Type: luautil.ErrTypEvil}
			}
		}
	}()
	//_ = fmt.Print

	block, err := ast.Parse(source, line)
	if err != nil {
		return nil, err
	}
	return compile(&ast.FuncDecl{Source: name, IsVariadic: true, Block: block}, nil), nil
}

func compile(f *ast.FuncDecl, parent *compState) *funcProto {
	name := f.Source
	if name == "" && parent != nil {
		name = parent.f.source
	}
	state := &compState{
		f: &funcProto{
			source: name,
			upVals: []upDef{
				{
					index: 0,
					name:  "_ENV",
				},
			},
			lineDefined:    f.Line(),
			parameterCount: len(f.Params),
		},
		p: parent,
	}

	if f.IsVariadic {
		state.f.isVarArg = 2 // Set to 1 if the function actually uses "..."
	}

	for _, param := range f.Params {
		state.locals = append(state.locals, state.nextReg)
		state.nextReg++
		state.f.localVars = append(state.f.localVars, localVar{
			name: param,
			sPC:  0,
			ePC:  -10, // Filled in with pc+2 later
		})
	}

	block(f.Block, state)

	state.addInst(createABC(opReturn, 0, 1, 0), -1)

	for i := range state.f.localVars {
		if state.f.localVars[i].ePC == -10 {
			state.f.localVars[i].ePC = int32(len(state.f.code))
		}
	}
	return state.f
}

func block(block []ast.Stmt, state *compState) {
	prepBlock(state)
	preppedBlock(block, state, 0)
}

func prepBlock(state *compState) {
	// Any code that creates a local in this block should use stuff.epc as the ePC value,
	// that way the block close code can find the required locals and set the proper
	// end value.
	// Locals that are in scope are guaranteed to have a sPC that is greater than
	// their ePC.
	state.blocks = append(state.blocks, &blockStuff{bpc: len(state.f.code) - 1, gotos: map[string][]jumpDat{}})
}

func preppedBlock(block []ast.Stmt, state *compState, epilogue int) {
	for _, n := range block {
		statement(n, state)
	}

	closeBlock(block, state, epilogue, 0)
}

func closeBlock(block []ast.Stmt, state *compState, epilogue, off int) {
	// Patch this block's local ePC values
	stuff := state.blocks[len(state.blocks)-1]
	state.blocks = state.blocks[:len(state.blocks)-1]
	locals := 0
	for i, l := range state.f.localVars {
		if l.sPC > l.ePC && l.ePC == int32(stuff.bpc) {
			state.f.localVars[i].ePC = int32(len(state.f.code) + epilogue)
			locals++
		}
	}

	// Adjust for locals going out of scope.
	state.nextReg -= locals

	// Issue a dummy JMP to close any upvalues if needed.
	// What ever happened to the CLOSE instruction? Using JMP seems weird.
	line := 0
	if len(block) != 0 {
		line = block[len(block)-1].Line()
	}
	if len(state.blocks) != 0 && stuff.hasUp {
		state.addInst(createAsBx(opJump, state.nextReg+1, off), line)
	} else if off != 0 {
		state.addInst(createAsBx(opJump, 0, off), line)
	}

	// Resolve this block's labels
	for _, l := range stuff.labels {
		if ts, ok := stuff.gotos[l.label]; ok {
			for _, t := range ts {
				t.patch(state.f, l) // Checks for jumping into a local's scope
			}
			delete(stuff.gotos, l.label)
		}
	}

	// If this is a top-level block make sure there are no unresolved gotos
	if len(state.blocks) == 0 {
		if len(stuff.gotos) == 1 {
			for label := range stuff.gotos {
				// Report only the first problem goto.
				issue := stuff.gotos[label][len(stuff.gotos[label])-1]
				luautil.Raise(fmt.Sprintf("Goto on line %v with undefined label %q", issue.line, issue.label), luautil.ErrTypGenSyntax)
			}
		}
		if len(stuff.gotos) > 1 {
			luautil.Raise(fmt.Sprintf("Multiple gotos with undefined labels."), luautil.ErrTypGenSyntax)
		}

		return
	}

	// Promote any unresolved gotos in this block to the next block up
	pstuff := state.blocks[len(state.blocks)-1]
	for t, ts := range stuff.gotos {
		pstuff.gotos[t] = append(pstuff.gotos[t], ts...)
	}
}

func statement(n ast.Stmt, state *compState) {
	switch nn := n.(type) {
	case *ast.Assign:
		if nn.LocalDecl {
			if len(nn.Values) == 0 {
				state.addInst(createABC(opLoadNil, state.nextReg, len(nn.Targets)-1, 0), nn.Line())
			} else {
				exprlist(nn.Values, state, state.nextReg, len(nn.Targets))
			}

			// Don't actually create the locals until they are all set, that way they are not available
			// inside their own initialization expressions.
			for _, e := range nn.Targets {
				// Each e must be a single name constant
				n, ok := e.(*ast.ConstIdent)
				if !ok {
					luautil.Raise(fmt.Sprintf("Invalid local declaration on line %v", e.Line()), luautil.ErrTypGenSyntax) // TODO: Better errors
				}

				// For some bizarre reason it is not an error to redeclare a local variable.
				// Since I already search the variable list in reverse order all I need to do
				// is blindly declare the "new" variable.
				state.mklocal(n.Value, 0)
			}
			return
		}
		if nn.LocalFunc {
			// nn.Targets is exactly one ConstIdent
			// nn.Values is exactly one FuncDecl
			// The local needs to be defined before the initializer is handled because this should
			// be equivalent to a local declaration followed by an assignment.
			n, ok := nn.Targets[0].(*ast.ConstIdent)
			if !ok {
				luautil.Raise(fmt.Sprintf("Invalid local function name on line %v", nn.Targets[0].Line()), luautil.ErrTypGenSyntax) // TODO: Better errors
			}
			reg := state.nextReg
			state.mklocal(n.Value, 0)
			expr(nn.Values[0], state, reg, false).To(false)
			return
		}

		// My solution to table clobbering (where a non-table set clobbers an "earlier" table set by overwriting the
		// register holding the table) if fairly inefficient, but it works. (note that this problem also effects upvalues
		// and register keys for tables)

		// Lower targets

		// No-clobber lists
		tblat := make(map[int][]int)
		upat := make(map[int][]int)
		keyat := make(map[int][]int)

		tdata := make([]identData, len(nn.Targets))
		nextTemp := state.nextReg
		for c, target := range nn.Targets {
			data, usedregs := lowerIdent(target, state, nextTemp)

			// Populate the non-clobber tables
			if data.isTable && !data.isUp {
				tblat[data.itemIdx] = append(tblat[data.itemIdx], c)
			} else if data.isUp {
				upat[data.itemIdx] = append(upat[data.itemIdx], c)
			}
			if data.isTable && !isK(data.keyRK) {
				keyat[data.keyRK] = append(keyat[data.keyRK], c)
			}

			nextTemp += usedregs // 0, 1, or 2 (2 will only come up if the last element is a table access with an expression key)
			tdata[c] = data
		}

		// Evaluate expressions
		exprlist(nn.Values, state, nextTemp, len(nn.Targets))

		// Get the top index so we have a place to shift tables if we need to.
		firstRes := nextTemp
		nextTemp += len(nn.Targets)

		// Assign values
		// Do the assignments in reverse order so that lower items do not clobber upper items
		// (during expression execution for example)
		for i := len(nn.Targets) - 1; i >= 0; i-- {
			data := tdata[i]
			// Test if we need to shift any tables/upvalues/keys.
			if !data.isTable {
				if data.isUp {
					// Don't clobber upvalues that we will need later...
					if ds, ok := upat[data.itemIdx]; ok {
						state.addInst(createABC(opGetUpValue, nextTemp, tdata[ds[0]].itemIdx, 0), data.line)
						for _, d := range ds {
							tdata[d].itemIdx = nextTemp
							tdata[d].isUp = false
						}
						nextTemp++
					}
				} else {
					// Or tables...
					if ds, ok := tblat[data.itemIdx]; ok {
						state.addInst(createABC(opMove, nextTemp, tdata[ds[0]].itemIdx, 0), data.line)
						for _, d := range ds {
							tdata[d].itemIdx = nextTemp
						}
						nextTemp++
					}

					// Or registers holding table keys.
					if ds, ok := keyat[data.itemIdx]; ok {
						state.addInst(createABC(opMove, nextTemp, tdata[ds[0]].keyRK, 0), data.line)
						for _, d := range ds {
							tdata[d].keyRK = nextTemp
						}
						nextTemp++
					}
				}
			}

			data.Set(firstRes + i)
		}

		// This version is *too* clever. `a, b = b, a` won't work due to mutual clobbering.
		// Evaluate expressions
		//results := []int{}
		//req := len(nn.Targets)
		//for i, e := range nn.Values {
		//	if i == len(nn.Values)-1 {
		//		ex := expr(e, state, nextTemp, false)
		//		if req > 1 {
		//			ex.To(false)
		//			ex.setResults(req)
		//			results = append(results, nextTemp)
		//			break
		//		}
		//		r, _ := ex.RK()
		//		results = append(results, r)
		//		break
		//	}
		//	r, u := expr(e, state, nextTemp, false).RK()
		//	results = append(results, r)
		//	if u {
		//		nextTemp++
		//	}
		//	req--
		//}
		//
		//// Assign values
		//// Do the assignments in reverse order so that lower items do not clobber upper items
		//// (during expression execution for example)
		//for i := len(nn.Targets)-1; i >= 0; i-- {
		//	if i >= len(results) {
		//		lr := len(results)-1
		//		tdata[i].Set(results[lr] + (i - lr))
		//	} else {
		//		tdata[i].Set(results[i])
		//	}
		//}
	case *ast.DoBlock:
		block(nn.Block, state)
	case *ast.If:
		list, k := expr(nn.Cond, state, state.nextReg, false).Bool()
		if list == nil {
			if k {
				block(nn.Then, state)
			} else {
				block(nn.Else, state)
			}
			return
		}
		block(nn.Then, state)
		toend := patchList([]int{len(state.f.code)})
		state.addInst(createAsBx(opJump, 0, 0), nn.Line())
		thenend := len(state.f.code)
		block(nn.Else, state)
		if thenend == len(state.f.code) {
			// If there was no else block or the else block contained no code remove the unnecessary jump instruction
			list.patch(state.f, thenend-1)
			state.f.code = state.f.code[:len(state.f.code)-1]
			state.f.lineInfo = state.f.lineInfo[:len(state.f.lineInfo)-1]
		} else {
			// Else patch the jump instruction so it skips the else block
			list.patch(state.f, thenend)
			toend.patch(state.f, len(state.f.code))
		}
	case *ast.WhileLoop:
		begin := len(state.f.code)
		list, k := expr(nn.Cond, state, state.nextReg, false).Bool()
		if list == nil && !k {
			return
		}
		state.breaks = append(state.breaks, patchList([]int{}))
		state.continues = append(state.continues, patchList([]int{}))
		block(nn.Block, state)
		tmp := state.continues[len(state.continues)-1]
		state.continues = state.continues[:len(state.continues)-1]
		tmp.loop(state.f, len(state.f.code), state.nextReg+1)
		state.addInst(createAsBx(opJump, 0, mkoffset(len(state.f.code), begin)), nn.Line()) // Go back to the top
		if list != nil {
			list.patch(state.f, len(state.f.code)) // Set the false jump target to the next instruction (does not exist yet).
		}
		tmp = state.breaks[len(state.breaks)-1]
		state.breaks = state.breaks[:len(state.breaks)-1]
		tmp.loop(state.f, len(state.f.code), state.nextReg+1)
	case *ast.RepeatUntilLoop:
		begin := len(state.f.code)
		state.breaks = append(state.breaks, patchList([]int{}))
		state.continues = append(state.continues, patchList([]int{}))

		// I hate repeat-until.
		// I need to manually parse the block here, then jump through hoops to make sure the upvalues are not closed
		// before the expression is parsed. It's nasty.
		prepBlock(state)
		for _, n := range nn.Block {
			statement(n, state)
		}
		tmp := state.continues[len(state.continues)-1]
		state.continues = state.continues[:len(state.continues)-1]
		tmp.loop(state.f, len(state.f.code), state.nextReg+1)
		list, k := expr(nn.Cond, state, state.nextReg, false).Bool()
		if list == nil {
			if k {
				closeBlock(nn.Block, state, 0, 0)
			} else {
				closeBlock(nn.Block, state, 0, mkoffset(len(state.f.code), begin))
			}
		} else {
			closeBlock(nn.Block, state, 0, 0)
			list.loop(state.f, begin, state.nextReg+1) // Set the false jump target to the loop beginning.
		}
		tmp = state.breaks[len(state.breaks)-1]
		state.breaks = state.breaks[:len(state.breaks)-1]
		tmp.loop(state.f, len(state.f.code), state.nextReg+1)

	case *ast.ForLoopNumeric:
		prepBlock(state)
		initReg, nreg := state.nextReg, state.nextReg
		pl := state.mklocaladv("(for index)", nil)
		pl = state.mklocaladv("(for limit)", pl)
		pl = state.mklocaladv("(for step)", pl)
		pl = state.mklocaladv(nn.Counter, pl)
		expr(nn.Init, state, nreg, false).To(false)
		nreg++
		expr(nn.Limit, state, nreg, false).To(false)
		nreg++
		expr(nn.Step, state, nreg, false).To(false)
		pl.patch(state.f, 1)
		prep := patchList([]int{len(state.f.code)})
		state.addInst(createAsBx(opForPrep, initReg, 0), nn.Line())
		ltop := len(state.f.code)
		state.breaks = append(state.breaks, patchList([]int{}))
		state.continues = append(state.continues, patchList([]int{}))
		preppedBlock(nn.Block, state, 1)
		lbottom := len(state.f.code)
		tmp := state.continues[len(state.continues)-1]
		state.continues = state.continues[:len(state.continues)-1]
		tmp.loop(state.f, len(state.f.code), state.nextReg+1)
		state.addInst(createAsBx(opForLoop, initReg, mkoffset(lbottom, ltop)), nn.Line())
		tmp = state.breaks[len(state.breaks)-1]
		state.breaks = state.breaks[:len(state.breaks)-1]
		tmp.loop(state.f, len(state.f.code), state.nextReg+1)
		prep.patch(state.f, lbottom)
	case *ast.ForLoopGeneric:
		initReg := state.nextReg
		prepBlock(state)
		pl := state.mklocaladv("(for generator)", nil)
		pl = state.mklocaladv("(for state)", pl)
		pl = state.mklocaladv("(for control)", pl)
		exprlist(nn.Init, state, initReg, 3)
		pl.patch(state.f, 1)
		for _, name := range nn.Locals {
			state.mklocal(name, 1)
		}
		begin := patchList([]int{len(state.f.code)})
		state.addInst(createAsBx(opJump, 0, 0), nn.Line())
		ltop := len(state.f.code)
		state.breaks = append(state.breaks, patchList([]int{}))
		state.continues = append(state.continues, patchList([]int{}))
		preppedBlock(nn.Block, state, 2)
		state.addInst(createABC(opTForCall, initReg, 0, len(nn.Locals)), nn.Line())
		lbottom := len(state.f.code)
		tmp := state.continues[len(state.continues)-1]
		state.continues = state.continues[:len(state.continues)-1]
		tmp.loop(state.f, len(state.f.code), state.nextReg+1)
		state.addInst(createAsBx(opTForLoop, initReg+2, mkoffset(lbottom, ltop)), nn.Line())
		tmp = state.breaks[len(state.breaks)-1]
		state.breaks = state.breaks[:len(state.breaks)-1]
		tmp.loop(state.f, len(state.f.code), state.nextReg+1)
		begin.patch(state.f, lbottom-1)
	case *ast.Goto:
		if nn.IsBreak {
			if len(state.breaks) == 0 {
				luautil.Raise(fmt.Sprintf("Break or continue statement on line %v outside of loop", nn.Line()), luautil.ErrTypGenSyntax) // TODO: Better errors
			}
			if nn.Label == "break" {
				l := len(state.breaks) - 1
				state.breaks[l] = append(state.breaks[l], len(state.f.code))
				state.addInst(createAsBx(opJump, 0, 0), nn.Line())
				break
			}
			l := len(state.continues) - 1
			state.continues[l] = append(state.continues[l], len(state.f.code))
			state.addInst(createAsBx(opJump, 0, 0), nn.Line())
			break
		}

		stuff := state.blocks[len(state.blocks)-1]
		stuff.gotos[nn.Label] = append(stuff.gotos[nn.Label], jumpDat{
			label: nn.Label,
			pc:    len(state.f.code),
			regs:  state.nextReg,
			line:  nn.Line(),
		})
		state.addInst(createAsBx(opJump, 0, 0), nn.Line())
	case *ast.Label:
		stuff := state.blocks[len(state.blocks)-1]
		stuff.labels = append(stuff.labels, jumpDat{
			label: nn.Label,
			pc:    len(state.f.code),
			regs:  state.nextReg,
			line:  nn.Line(),
		})
	case *ast.Return:
		nreg := state.nextReg
		items := len(nn.Items) + 1

		if len(nn.Items) == 1 {
			if call, ok := nn.Items[0].(*ast.FuncCall); ok {
				compileCall(call, state, nreg, -1, true)
				// Don't bother generating an unnecessary RETURN instruction.
				return
			}
		}

		for i, e := range nn.Items {
			ex := expr(e, state, nreg, false)
			if i == len(nn.Items)-1 && ex.mayMulti {
				ex.setResults(-1)
				items = 0
			}
			ex.To(false)
			nreg++
		}

		state.addInst(createABC(opReturn, state.nextReg, items, 0), nn.Line())
	case *ast.FuncCall:
		compileCall(nn, state, state.nextReg, 0, false)
	}
}
