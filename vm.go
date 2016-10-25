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

import "bytes"

import "github.com/milochristiansen/lua/luautil"

// call handles all function calls. "fi" *must* be a valid stack index!
func (l *State) call(fi, args, rtns int, tail bool) {
	if fi < 0 {
		fi = l.stack.TopIndex() + fi + 1
	}
	
	v := l.stack.Get(fi)
	f, ok := v.(*function)
	if !ok {
		meth := l.hasMetaMethod(v, "__call")
		if meth != nil {
			f, ok := meth.(*function)
			if !ok {
				luautil.Raise("Meta method __call is not a function.", luautil.ErrTypGenRuntime)
			}
			
			l.stack.Insert(fi, f)
			
			if tail {
				l.stack.TailFrame(f, fi, args + 1)
				l.exec()
				return
			}
			l.stack.AddFrame(f, fi, args + 1, rtns)
			l.exec()
			l.stack.ReturnFrame()
			return
		}
		luautil.Raise("Value is not a function and has no __call meta method.", luautil.ErrTypGenRuntime)
	}
	
	if tail {
		l.stack.TailFrame(f, fi, args)
		l.exec()
		return
	}
	l.stack.AddFrame(f, fi, args, rtns)
	l.exec()
	l.stack.ReturnFrame()
	return
}

func (l *State) exec() {
	if l.stack.cFrame().fn.native != nil {
		fr := l.stack.cFrame()
		fr.retC = fr.fn.native(l)
		fr.retBase = l.stack.TopIndex() + 1 - fr.retC
	} else {
		i, ok := l.stack.cFrame().nxtOp()
		for ok {
			//l.Printf("[%v]\t%v\n", l.stack.cFrame().pc-1, i)
			_ = "breakpoint" // Next Instruction
			if instructionTable[i.getOpCode()](l, i) { // RETURN and TAILCALL return true
				return
			}
			//l.Printf("%#v\n", l.stack.data)
			i, ok = l.stack.cFrame().nxtOp()
		}
	}
	return
}

type float8 int

// Converts an integer to a "floating point byte", represented as
// (eeeeexxx), where the real value is (1xxx) * 2^(eeeee - 1) if
// eeeee != 0 and (xxx) otherwise.
func float8FromInt(x int) float8 {
	if x < 8 {
		return float8(x)
	}
	e := 0
	for ; x >= 0x10; e++ {
		x = (x + 1) >> 1
	}
	return float8(((e + 1) << 3) | (x - 8))
}

func intFromFloat8(x float8) int {
	e := x >> 3 & 0x1f
	if e == 0 {
		return int(x)
	}
	return int(x&7+8) << uint(e-1)
}

func rk(l *State, f int) value {
	if (f & bitRK) != 0 {
		return l.stack.cFrame().fn.proto.constants[f & ^bitRK]
	} else {
		return l.stack.Get(f)
	}
}

var instructionTable [opCodeCount]func(l *State, i instruction) bool
func init() {
	instructionTable = [opCodeCount]func(l *State, i instruction) bool{
		// MOVE
		func(l *State, i instruction) bool {
			l.stack.Set(i.a(), l.stack.Get(i.b()))
			return false
		},
		// LOADK
		func(l *State, i instruction) bool {
			l.stack.Set(i.a(), l.stack.cFrame().fn.proto.constants[i.bx()])
			return false
		},
		// LOADKX
		func(l *State, i instruction) bool {
			l.stack.Set(i.a(), l.stack.cFrame().fn.proto.constants[l.stack.cFrame().reqNxtOp(opExtraArg).ax()])
			return false
		},
		// LOADBOOL
		func(l *State, i instruction) bool {
			if i.c() != 0 {
				l.stack.cFrame().pc++
			}
	
			l.stack.Set(i.a(), i.b() != 0)
			return false
		},
		// LOADNIL
		func(l *State, i instruction) bool {
			a, b := i.a(), i.b()
			
			for k := a; k <= a + b; k++ {
				l.stack.Set(k, nil)
			}
			return false
		},
	
		// GETUPVAL
		func(l *State, i instruction) bool {
			l.stack.Set(i.a(), l.stack.cFrame().getUp(i.b()))
			return false
		},
		// GETTABUP
		func(l *State, i instruction) bool {
			tbl := l.stack.cFrame().getUp(i.b())
			l.stack.Set(i.a(), l.getTable(tbl, rk(l, i.c())))
			return false
		},
		// GETTABLE
		func(l *State, i instruction) bool {
			tbl := l.stack.Get(i.b())
			l.stack.Set(i.a(), l.getTable(tbl, rk(l, i.c())))
			return false
		},
	
		// SETTABUP
		func(l *State, i instruction) bool {
			tbl := l.stack.cFrame().getUp(i.a())
			l.setTable(tbl, rk(l, i.b()), rk(l, i.c()))
			return false
		},
		// SETUPVAL
		func(l *State, i instruction) bool {
			l.stack.cFrame().setUp(i.b(), l.stack.Get(i.a())) // Yes, this really is the reverse of everything else...
			return false
		},
		// SETTABLE
		func(l *State, i instruction) bool {
			tbl := l.stack.Get(i.a())
			l.setTable(tbl, rk(l, i.b()), rk(l, i.c()))
			return false
		},
	
		// NEWTABLE
		func(l *State, i instruction) bool {
			var tbl *table
			if b, c := float8(i.b()), float8(i.c()); b != 0 || c != 0 {
				tbl = newTable(l, intFromFloat8(b), intFromFloat8(c))
			} else {
				tbl = newTable(l, 0, 0)
			}
			l.stack.Set(i.a(), tbl)
			return false
		},
	
		// SELF
		func(l *State, i instruction) bool {
			a := i.a()
			tbl := l.stack.Get(i.b())
			l.stack.Set(a, l.getTable(tbl, rk(l, i.c())))
			l.stack.Set(a+1, tbl)
			return false
		},
	
		// ADD
		opMath,
		// SUB
		opMath,
		// MUL
		opMath,
		// MOD
		opMath,
		// POW
		opMath,
		// DIV
		opMath,
		// IDIV
		opMath,
		// BAND
		opMath,
		// BOR
		opMath,
		// BXOR
		opMath,
		// SHL
		opMath,
		// SHR
		opMath,
		// UNM
		func(l *State, i instruction) bool {
			b := rk(l, i.b())
			l.stack.Set(i.a(), l.arith(i.getOpCode(), b, b))
			return false
		},
		// BNOT
		func(l *State, i instruction) bool {
			b := rk(l, i.b())
			l.stack.Set(i.a(), l.arith(i.getOpCode(), b, b))
			return false
		},
		// NOT
		func(l *State, i instruction) bool {
			l.stack.Set(i.a(), !toBool(rk(l, i.b())))
			return false
		},
		// LEN
		func(l *State, i instruction) bool {
			v := l.stack.Get(i.b())
			
			if s, ok := v.(string); ok {
				l.stack.Set(i.a(), int64(len(s)))
				return false
			}
			
			meth := l.hasMetaMethod(v, "__len")
			if meth != nil {
				l.Push(meth)
				l.Push(v)
				l.Call(1, 1)
				rtn := l.stack.Get(-1)
				l.Pop(1)
				l.stack.Set(i.a(), rtn)
				return false
			}
			
			tbl, ok := v.(*table)
			if !ok {
				luautil.Raise("Value is not a string or table and has no __len meta method.", luautil.ErrTypGenRuntime)
			}
			l.stack.Set(i.a(), int64(tbl.Length()))
			return false
		},
	
		// CONCAT
		func(l *State, i instruction) bool {
			b, c := i.b(), i.c()
			
			var buff *bytes.Buffer
			concat := func(v1, v2 value) {
				meth := l.hasMetaMethod(v1, "__concat")
				if meth == nil {
					meth = l.hasMetaMethod(v2, "__concat")
					if meth == nil {
						toStringConcat(v1) // For the error message
						toStringConcat(v2)
						panic("UNREACHABLE")
					}
				}
				
				l.Push(meth)
				l.Push(v1)
				l.Push(v2)
				l.Call(2, 1)
				rtn := l.stack.Get(-1)
				l.Pop(1)
				buff = bytes.NewBufferString(toStringConcat(rtn))
			}
			
			k := b
			v := l.stack.Get(k)
			if t := typeOf(v); t == TypString || t == TypNumber {
				buff = bytes.NewBufferString(toStringConcat(v))
			} else {
				k++
				if k > c {
					luautil.Raise("CONCAT called with a range of less than 2 registers.", luautil.ErrTypMajorInternal)
				}
				v2 := l.stack.Get(k)
					
				concat(v, v2)
			}
			k++
			
			for ; k <= c; k++ {
				v := l.stack.Get(k)
				if t := typeOf(v); t == TypString || t == TypNumber {
					buff.WriteString(toStringConcat(v))
					continue
				}
				concat(buff.String(), v)
			}
			l.stack.Set(i.a(), buff.String())
			return false
		},
	
		// JMP
		func(l *State, i instruction) bool {
			a := i.a()
			if a != 0 {
				// Close all upvalues that refer to indexes at or above A-1
				l.stack.cFrame().closeUp(a-1)
			}
			
			l.stack.cFrame().pc += int32(i.sbx())
			return false
		},
		// EQ
		opCmp,
		// LT
		opCmp,
		// LE
		opCmp,
	
		// TEST
		func(l *State, i instruction) bool {
			if toBool(l.stack.Get(i.a())) == (i.c() == 0) {
				l.stack.cFrame().pc++
			}
			// I don't require a following JMP instruction.
			return false
		},
		// TESTSET
		func(l *State, i instruction) bool {
			b := l.stack.Get(i.b())
			if toBool(b) == (i.c() == 0) {
				l.stack.cFrame().pc++
			} else {
				l.stack.Set(i.a(), b)
				// I don't require a following JMP instruction.
			}
			return false
		},
	
		// CALL
		func(l *State, i instruction) bool {
			a, b, c := i.a(), i.b(), i.c()
			
			args := 0
			switch b {
			case 0:
				args = l.stack.TopIndex() - a
			default:
				args = b - 1
			}
			
			// Value of -1 means "set later" (either by RETURN or the return value of a native function).
			rtns := c - 1
			
			l.call(a, args, rtns, false)
			return false
		},
		// TAILCALL
		func(l *State, i instruction) bool {
			l.stack.cFrame().closeUpAll()
			
			a, b := i.a(), i.b()
			
			args := 0
			switch b {
			case 0:
				args = l.stack.TopIndex() - a
			default:
				args = b - 1
			}
			
			l.call(a, args, -1, true)
			return true
		},
		// RETURN
		func(l *State, i instruction) bool {
			l.stack.cFrame().closeUpAll()
			
			a, b := i.a(), i.b()
			b--
			
			l.stack.cFrame().retBase = a
			
			// Nothing to return
			if b == 0 {
				// Break
				l.stack.cFrame().retC = 0
				return true
			}
			
			// Return all items from a to TOS
			if b < 0 {
				l.stack.cFrame().retC = l.stack.TopIndex() + 1 - a
				return true
			}
			
			// Fixed number of results
			l.stack.cFrame().retC = b
			return true
		},
	
		// FORLOOP
		func(l *State, i instruction) bool {
			a := i.a()
			step := l.stack.Get(a + 2)
			av := l.arith(OpAdd, l.stack.Get(a), step) // Probably bad for performance...
			
			cmp := false
			if toFloat(step) < 0 {
				cmp = l.compare(OpLessOrEqual, l.stack.Get(a + 1), av, true) // I should probably do a raw check here too
			} else {
				cmp = l.compare(OpLessOrEqual, av, l.stack.Get(a + 1), true)
			}
			if !cmp {
				return false
			}
			
			l.stack.Set(a, av)
			l.stack.Set(a + 3, av)
			l.stack.cFrame().pc += int32(i.sbx())
			return false
		},
		// FORPREP
		func(l *State, i instruction) bool {
			a := i.a()
			init, limit, step := l.stack.Get(a), l.stack.Get(a + 1), l.stack.Get(a + 2)
			
			// Make sure all three values are numbers, preferably integers.
			iinit, oka := tryInt(init)
			ilimit, okb := tryInt(limit)
			istep, okc := tryInt(step)
			if oka && okb && okc {
				l.stack.Set(a, iinit - istep)
				l.stack.Set(a + 1, ilimit)
				l.stack.Set(a + 2, istep)
				l.stack.cFrame().pc += int32(i.sbx())
				return false
			}
	
			finit, oka := tryFloat(init)
			flimit, okb := tryFloat(limit)
			fstep, okc := tryFloat(step)
			if !(oka && okb && okc) {
				luautil.Raise("All three values passed to a numeric for loop must be numeric!", luautil.ErrTypGenRuntime)
			}
			
			l.stack.Set(a, finit - fstep)
			l.stack.Set(a + 1, flimit)
			l.stack.Set(a + 2, fstep)
			l.stack.cFrame().pc += int32(i.sbx())
			return false
		},
	
		// TFORCALL
		func(l *State, i instruction) bool {
			a, c := i.a(), i.c()
			
			// Clear the upper parts of the stack so the results of the iterator call will be at a known fixed index.
			segC, _ := l.stack.bounds(-1)
			l.stack.data = l.stack.data[:segC + a + 4]
			
			l.stack.Push(l.stack.Get(a))
			l.stack.Push(l.stack.Get(a + 1))
			l.stack.Push(l.stack.Get(a + 2))
			l.Call(2, c)
			
			// C Lua asserts that this is followed by a TFORLOOP, then jumps directly there, but why bother.
			// Not handling these two op codes explicitly in tandem adds one cycle to to loop processing, but
			// frankly there is not much gained by doing it the C way, particularly since I can't just "goto"
			// where I need to be.
			return false
		},
		// TFORLOOP
		func(l *State, i instruction) bool {
			a := i.a()
			if l.stack.Get(a + 1) == nil {
				return false
			}
			
			l.stack.Set(a, l.stack.Get(a + 1))
			l.stack.cFrame().pc += int32(i.sbx())
			return false
		},
	
		// SETLIST
		func(l *State, i instruction) bool {
			// Here I assume the array part of the table is already preallocated to the correct size (which it
			// will be in most cases). If not performance will suffer. I should probably fix this.
			
			a := i.a()
			t := l.stack.Get(a).(*table)
			b, c := i.b(), i.c()
			
			if b == 0 {
				b = l.stack.TopIndex() - a
			}
			
			if c == 0 {
				c = l.stack.cFrame().reqNxtOp(opExtraArg).ax()
			}
			
			first := (c - 1) * fieldsPerFlush + TableIndexOffset
			for i := 0; i < b; i++ {
				t.SetRaw(int64(first + i), l.stack.Get(a+1+i))
			}
			
			// Drop the values above "a"
			l.stack.SetTop(a)
			
			return false
		},
	
		// CLOSURE
		func(l *State, i instruction) bool {
			me := l.stack.cFrame()
			p := me.fn.proto.prototypes[i.bx()]
			
			f := &function{
				proto: p,  
				uvDefs: p.upVals,
				uvClosed: make([]bool, len(p.upVals)),
				upVals: make([]value, len(p.upVals)),
				uvAbsIdxs: make([]int, len(p.upVals)),
			}
			
			// luac seems to set isLocal for _ENV, but only for the top level function.
			// Since Call automatically closes the _ENV value for the top level function this
			// should not make any real difference, but it's weird...
			
			// Add all upvalues to the unclosed list:
			for i, def := range f.uvDefs {
				ouc := me.unclosed
				me.unclosed = &ucUpVal{
					prev: nil,
					next: me.unclosed,
					fn: f,
					idx: i,
				}
				if def.isLocal {
					f.uvAbsIdxs[i] = l.stack.absIndex(def.index)
				} else {
					f.uvAbsIdxs[i] = me.fn.uvAbsIdxs[def.index]
				}
				if ouc != nil {
					ouc.prev = me.unclosed
				}
			}
			
			l.stack.Set(i.a(), f)
			return false
		},
	
		// VARARG
		func(l *State, i instruction) bool {
			a, b := i.a(), i.b() - 1
			
			argc := l.stack.cFrame().nArgs
			if b == -1 {
				b = argc
			}
			
			for k := b - 1; k >= 0; k-- {
				if k >= argc {
					l.stack.Set(a + k, nil)
					continue
				}
				l.stack.Set(a + k, l.stack.GetArgs(k))
			}
			return false
		},
	
		// EXTRAARG
		func(l *State, i instruction) bool {
			luautil.Raise("Impossible instruction!", luautil.ErrTypMajorInternal)
			return false
		},
	}
}

func opMath(l *State, i instruction) bool {
	l.stack.Set(i.a(), l.arith(i.getOpCode(), rk(l, i.b()), rk(l, i.c())))
	return false
}

func opCmp(l *State, i instruction) bool {
	// Unlike the C Lua I do not always execute a JMP in the true case.
	// This adds flexibility at the (arguable) cost of performance.
	// Lua does not use this flexibility, but I want to use this VM
	// (with minor changes) for my own languages.
	if !l.compare(i.getOpCode(), rk(l, i.b()), rk(l, i.c()), false) == (i.a() != 0) {
		l.stack.cFrame().pc++
	}
	
	return false
}
