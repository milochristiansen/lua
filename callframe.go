/*
Copyright 2015-2016 by Milo Christiansen

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

import "github.com/milochristiansen/lua/luautil"

type callFrame struct {
	fn  *function
	stk *stack

	pc int32

	base int // The index of the last item from the previous frame (-1 if this is the first frame)

	// If true then the function is all of the following:
	//	1. A Lua function
	//	2. A variadic function
	//	3. Contains at least one use of ...
	// In this case all stack operation must be offset by nArgs to prevent the arguments from being clobbered.
	holdArgs bool
	
	nArgs   int
	nRet    int // The number of items expected
	retC    int // The actual number of items returned
	retBase int // First value to return
	retTo   int // Index (in previous frame) to place the first return value into.
	
	unclosed *ucUpVal
}

type ucUpVal struct {
	next *ucUpVal
	prev *ucUpVal
	
	fn  *function
	idx int
	
	absidx int // Absolute index into the stack for local upvalues.
}

func (uc *ucUpVal) unlink() (*ucUpVal, bool) {
	if uc.prev != nil && uc.next != nil {
		uc.prev.next, uc.next.prev = uc.next, uc.prev
		return nil, false
	}
	if uc.prev != nil {
		uc.prev.next = nil
		return nil, false
	}
	if uc.next != nil {
		uc.next.prev = nil
		return uc.next, true
	}
	return nil, true
}

// nxtOp gets the next opCode from a Lua function's code.
func (cf *callFrame) nxtOp() (instruction, bool) {
	if int(cf.pc) >= len(cf.fn.proto.code) || cf.pc < 0 {
		return 0, false
	}

	i := cf.fn.proto.code[cf.pc]
	cf.pc++
	return i, true
}

// tryNxtOp gets the next opCode from a Lua function's code and ensures it is of a specific type.
// If the next opCode is not of the required type this returns "0, false".
func (cf *callFrame) tryNxtOp(op opCode) (instruction, bool) {
	if int(cf.pc) >= len(cf.fn.proto.code) {
		return 0, false
	}

	i := cf.fn.proto.code[cf.pc]
	if i.getOpCode() != op {
		return 0, false
	}
	cf.pc++
	return i, true
}

// reqNxtOp gets the next opCode from a Lua function's code and ensures it is of a specific type.
func (cf *callFrame) reqNxtOp(op opCode) instruction {
	if int(cf.pc) >= len(cf.fn.proto.code) {
		luautil.Raise("VM did not find required opcode!", luautil.ErrTypMajorInternal)
	}

	i := cf.fn.proto.code[cf.pc]
	cf.pc++
	if i.getOpCode() != op {
		luautil.Raise("VM did not find required opcode!", luautil.ErrTypMajorInternal)
	}
	return i
}

func (cf *callFrame) getUp(i int) value {
	if i < 0 || i >= len(cf.fn.uvDefs) {
		return nil
	}
	if cf.fn.uvClosed[i] {
		return cf.fn.upVals[i]
	}
	if cf.fn.uvDefs[i].isLocal || cf.fn.uvAbsIdxs[i] > -1 {
		return cf.stk.GetAbs(cf.fn.uvAbsIdxs[i])
	}
	return cf.getUpLvl(cf.fn.uvDefs[i].index, 2)
}

// Internal helper, don't use
func (cf *callFrame) getUpLvl(i, lvl int) value {
	pf := cf.stk.frame(lvl)
	if pf == nil {
		luautil.Raise("Unclosed parent upvalue and no parent frame available!", luautil.ErrTypMajorInternal)
	}
	
	if i < 0 || i >= len(pf.fn.uvDefs) {
		return nil
	}
	if pf.fn.uvClosed[i] {
		return pf.fn.upVals[i]
	}
	if pf.fn.uvDefs[i].isLocal || pf.fn.uvAbsIdxs[i] > -1 {
		return pf.stk.GetAbs(pf.fn.uvAbsIdxs[i])
	}
	return cf.getUpLvl(pf.fn.uvDefs[i].index, lvl+1)
}

func (cf *callFrame) setUp(i int, v value) {
	if i < 0 || i >= len(cf.fn.uvDefs) {
		return
	}
	if cf.fn.uvClosed[i] {
		cf.fn.upVals[i] = v
		return
	}
	if cf.fn.uvDefs[i].isLocal || cf.fn.uvAbsIdxs[i] > -1 {
		cf.stk.SetAbs(cf.fn.uvAbsIdxs[i], v)
		return
	}
	cf.setUpLvl(cf.fn.uvDefs[i].index, 2, v)
}

// Internal helper, don't use
func (cf *callFrame) setUpLvl(i, lvl int, v value) {
	pf := cf.stk.frame(lvl)
	if pf == nil {
		luautil.Raise("Unclosed parent upvalue and no parent frame available!", luautil.ErrTypMajorInternal)
	}
	
	if i < 0 || i >= len(pf.fn.uvDefs) {
		return
	}
	if pf.fn.uvClosed[i] {
		pf.fn.upVals[i] = v
		return
	}
	if pf.fn.uvDefs[i].isLocal || pf.fn.uvAbsIdxs[i] > -1 {
		pf.stk.SetAbs(pf.fn.uvAbsIdxs[i], v)
		return
	}
	cf.setUpLvl(pf.fn.uvDefs[i].index, lvl+1, v)
}

func (cf *callFrame) closeUp(a int) {
	for uc := cf.unclosed; uc != nil; uc = uc.next {
		def := uc.fn.uvDefs[uc.idx]
		if uc.fn.uvClosed[uc.idx] { panic("IMPOSSIBLE") }
		if def.index < a || !def.isLocal {
			continue
		}
		r, nr := uc.unlink()
		if nr {
			cf.unclosed = r
		}
		uc.fn.uvClosed[uc.idx] = true
		uc.fn.upVals[uc.idx] = cf.stk.GetAbs(uc.fn.uvAbsIdxs[uc.idx])
	}
}

func (cf *callFrame) closeUpAll() {
	for uc := cf.unclosed; uc != nil; uc = uc.next {
		def := uc.fn.uvDefs[uc.idx]
		if uc.fn.uvClosed[uc.idx] { panic("IMPOSSIBLE") }
		r, nr := uc.unlink()
		if nr {
			cf.unclosed = r
		}
		uc.fn.uvClosed[uc.idx] = true
		if def.isLocal {
			uc.fn.upVals[uc.idx] = cf.stk.GetAbs(uc.fn.uvAbsIdxs[uc.idx])
			continue
		}
		uc.fn.upVals[uc.idx] = cf.getUpLvl(def.index, 1)
	}
	if cf.unclosed != nil { panic("IMPOSSIBLE") }
}
