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
	if i < 0 || i >= len(cf.fn.up) {
		luautil.Raise("Attempt to get out of range upvalue!", luautil.ErrTypMajorInternal)
	}
	
	def := cf.fn.up[i]
	if def.isLocal && !def.closed {
		return cf.stk.GetAbs(def.absIdx)
	}
	if !def.closed { panic("IMPOSSIBLE") }
	return def.val
}

func (cf *callFrame) setUp(i int, v value) {
	if i < 0 || i >= len(cf.fn.up) {
		luautil.Raise("Attempt to set out of range upvalue!", luautil.ErrTypMajorInternal)
		return
	}
	def := cf.fn.up[i]
	if def.isLocal && !def.closed {
		cf.stk.SetAbs(def.absIdx, v)
		return
	}
	if !def.closed { panic("IMPOSSIBLE") }
	def.val = v
}

// Note that the closing functions close upvalues in the TOP frame(s) NOT the frame it was called on (unless called on the top frame).

func (cf *callFrame) closeUpAbs(a int) {
	//x := cf.stk.unclosed
	//for x != nil {
	//	println("< ", x.absIdx, ":", toString(cf.stk.GetAbs(x.absIdx)), ":", x.name)
	//	x = x.next
	//}
	//println("> close:", a)
	
	nxt := cf.stk.unclosed
	for nxt != nil {
		if !nxt.isLocal || nxt.absIdx < 0 { panic("IMPOSSIBLE") } // All upvalues in actual use are in some way on the stack or already closed
		if nxt.absIdx < a {
			break // all values beyond this point are lower in the stack
		}
		//println(">   closing", nxt.absIdx)
		
		nxt.val = cf.stk.GetAbs(nxt.absIdx)
		nxt.closed = true
		nxt = nxt.next
	}
	cf.stk.unclosed = nxt
}

// Lazy convenience
func (cf *callFrame) closeUp(a int) {
	cf.closeUpAbs(cf.stk.absIndex(a))
}

// Lazy convenience
func (cf *callFrame) closeUpAll() {
	cf.closeUpAbs(cf.stk.absIndex(0))
}

// Find or create an unclosed *local* upvalue that matches the definition
func (cf *callFrame) findUp(def upDef) *upValue {
	idx := cf.stk.absIndex(def.index)
	
	node := cf.stk.unclosed
	var pnode *upValue
	for {
		// Case order is very important!
		switch {
		case node == nil:
			// No list exists yet, add this item as the head.
			// This can only happen on the very first iteration, so check it last.
			up := def.makeUp()
			up.absIdx = idx
			
			cf.stk.unclosed = up
			return up
		case node.absIdx == idx:
			// Found a matching item, return it directly
			return node
		case node.absIdx < idx:
			// New item should be inserted just before this item
			up := def.makeUp()
			up.absIdx = idx
			
			if pnode == nil {
				up.next = node
				cf.stk.unclosed = up
			} else {
				up.next = node
				pnode.next = up
			}
			return up
		case node.next == nil:
			// If item should be added to the end of the list
			up := def.makeUp()
			up.absIdx = idx
			
			node.next = up
			return up
		}
		pnode = node
		node = node.next
	}
}
