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

import "math"
import "fmt"

import "github.com/milochristiansen/lua/luautil"

type TypeID int
type STypeID int

const (
	TypNil    TypeID = iota
	TypNumber        // Both int and float
	TypString
	TypBool
	TypTable
	TypFunction
	TypUserData

	typeCount int = iota
)

const (
	STypNone STypeID = iota
	STypInt
	STypFloat
)

var typeNames = [...]string{"nil", "number", "string", "boolean", "table", "function", "userdata"}

func (typ TypeID) String() string {
	return typeNames[typ]
}

var stypeNames = [...]string{"none", "int", "float"}

func (typ STypeID) String() string {
	return stypeNames[typ]
}

type value interface{}

// See table.go for Table

// See function.go for Function

type userData struct {
	meta *table
	data interface{}
}

// Utility functions

func typeOf(v value) TypeID {
	switch v.(type) {
	case nil:
		return TypNil
	case float64:
		return TypNumber
	case int64:
		return TypNumber
	case string:
		return TypString
	case bool:
		return TypBool
	case *table:
		return TypTable
	case *function:
		return TypFunction
	case *userData:
		return TypUserData
	default:
		return TypUserData // Should be an error?
	}
}

func sTypeOf(v value) STypeID {
	switch v.(type) {
	case nil:
		return STypNone
	case float64:
		return STypFloat
	case int64:
		return STypInt
	case string:
		return STypNone
	case bool:
		return STypNone
	case *table:
		return STypNone
	case *function:
		return STypNone
	case *userData:
		return STypNone
	default:
		return STypNone // Should be an error?
	}
}

func (l *State) getMetaTable(v value) *table {
	switch v2 := v.(type) {
	case nil:
		return l.metaTbls[TypNil]
	case float64:
		return l.metaTbls[TypNumber]
	case int64:
		return l.metaTbls[TypNumber]
	case string:
		return l.metaTbls[TypString]
	case bool:
		return l.metaTbls[TypBool]
	case *table:
		return v2.meta
	case *function:
		return l.metaTbls[TypFunction]
	case *userData:
		return v2.meta
	default:
		luautil.Raise("Invalid type passed to getMetaTable.", luautil.ErrTypMajorInternal)
		panic("UNREACHABLE")
	}
}

func (l *State) hasMetaMethod(v value, name string) value {
	tbl := l.getMetaTable(v)
	if tbl == nil {
		return nil
	}
	return tbl.GetRaw(name)
}

func toStringConcat(v value) string {
	switch v2 := v.(type) {
	case nil:
		luautil.Raise("Attempt to concatenate a nil value.", luautil.ErrTypGenRuntime)
		panic("UNREACHABLE")
	case float64:
		return fmt.Sprintf("%g", v2)
	case int64:
		return fmt.Sprintf("%d", v2)
	case string:
		return v2
	case bool:
		luautil.Raise("Attempt to concatenate a bool value.", luautil.ErrTypGenRuntime)
		panic("UNREACHABLE")
	case *table:
		luautil.Raise("Attempt to concatenate a table value.", luautil.ErrTypGenRuntime)
		panic("UNREACHABLE")
	case *function:
		luautil.Raise("Attempt to concatenate a function value.", luautil.ErrTypGenRuntime)
		panic("UNREACHABLE")
	case *userData:
		luautil.Raise("Attempt to concatenate a userdata value.", luautil.ErrTypGenRuntime)
		panic("UNREACHABLE")
	default:
		luautil.Raise("Invalid type passed to toStringConcat.", luautil.ErrTypMajorInternal)
		panic("UNREACHABLE")
	}
}

func toString(v value) string {
	switch v2 := v.(type) {
	case nil:
		return "nil"
	case float64:
		return fmt.Sprintf("%g", v2)
	case int64:
		return fmt.Sprintf("%d", v2)
	case string:
		return v2
	case bool:
		if v2 {
			return "true"
		}
		return "false"
	case *table:
		return fmt.Sprintf("table %p", v2)
	case *function:
		return fmt.Sprintf("function %p", v2)
	case *userData:
		return fmt.Sprintf("userdata %p", v2)
	default:
		return fmt.Sprintf("unknown %p", v2)
	}
}

func toBool(v value) bool {
	switch v2 := v.(type) {
	case nil:
		return false
	case bool:
		return v2
	default:
		return true
	}
}

func toFloat(v value) float64 {
	msg := ""
	switch v2 := v.(type) {
	case string:
		valid, _, _, f := luautil.ConvNumber(v2, false, true)
		if valid {
			return f
		}
		msg = ": String not numeric"
	case int64:
		return float64(v2)
	case float64:
		return v2
	default:
		msg = ": Not string or number"
	}
	luautil.Raise("Invalid conversion to float"+msg, luautil.ErrTypGenRuntime)
	panic("UNREACHABLE")
}

func tryFloat(v value) (float64, bool) {
	switch v2 := v.(type) {
	case string:
		valid, _, _, f := luautil.ConvNumber(v2, false, true)
		if valid {
			return f, true
		}
		return 0, false
	case int64:
		return float64(v2), true
	case float64:
		return v2, true
	default:
		return 0, false
	}
}

func toInt(v value) int64 {
	msg := ""
	switch v2 := v.(type) {
	case string:
		valid, _, i, _ := luautil.ConvNumber(v2, true, false)
		if valid {
			return i
		}
		msg = ": String not numeric"
	case int64:
		return v2
	case float64:
		if i := int64(v2); float64(i) == v2 {
			return i
		}
		msg = ": Non-integral float"
	default:
		msg = ": Not string or number"
	}
	luautil.Raise("Invalid conversion to integer"+msg, luautil.ErrTypGenRuntime)
	panic("UNREACHABLE")
}

func tryInt(v value) (int64, bool) {
	switch v2 := v.(type) {
	case string:
		valid, _, i, _ := luautil.ConvNumber(v2, true, false)
		if valid {
			return i, true
		}
		return 0, false
	case int64:
		return v2, true
	case float64:
		if i := int64(v2); float64(i) == v2 {
			return i, true
		}
		return 0, false
	default:
		return 0, false
	}
}

func (l *State) getTable(t, k value) value {
	tbl, ok := t.(*table)
	if ok && tbl.Exists(k) {
		return tbl.GetRaw(k)
	}

	meth := l.hasMetaMethod(t, "__index")
	if meth != nil {
		if tbl, ok := meth.(*table); ok {
			return l.getTable(tbl, k)
		}

		f, ok := meth.(*function)
		if !ok {
			luautil.Raise("Meta method __index is not a table or function.", luautil.ErrTypGenRuntime)
		}

		l.Push(f)
		l.Push(t)
		l.Push(k)
		l.Call(2, 1)
		rtn := l.stack.Get(-1)
		l.Pop(1)
		return rtn
	}

	if ok {
		return tbl.GetRaw(k)
	}
	luautil.Raise("Value is not a table and has no __index meta method.", luautil.ErrTypGenRuntime)
	panic("UNREACHABLE")
}

func (l *State) setTable(t, k, v value) {
	tbl, ok := t.(*table)
	if ok {
		tbl.SetRaw(k, v)
		return
	}

	meth := l.hasMetaMethod(t, "__newindex")
	if meth != nil {
		if t, ok := meth.(*table); ok {
			l.setTable(t, k, v)
			return
		}

		f, ok := meth.(*function)
		if !ok {
			luautil.Raise("Meta method __newindex is not a table or function.", luautil.ErrTypGenRuntime)
		}

		l.Push(f)
		l.Push(t)
		l.Push(k)
		l.Push(v)
		l.Call(3, 0)
		return
	}
	luautil.Raise("Value is not a table and has no __newindex meta method.", luautil.ErrTypGenRuntime)
	panic("UNREACHABLE")
}

var mathMeta = [...]string{
	"__add",
	"__sub",
	"__mul",
	"__mod",
	"__pow",
	"__div",
	"__idiv",
	"__band",
	"__bor",
	"__bxor",
	"__shl",
	"__shr",
	"__unm",
	"__bnot",
}

func (l *State) tryMathMeta(op opCode, a, b value) value {
	if op < OpAdd || op > OpBinNot {
		luautil.Raise("Operator passed to tryMathMeta out of range.", luautil.ErrTypMajorInternal)
	}
	name := mathMeta[op-OpAdd]

	meta := l.hasMetaMethod(a, name)
	if meta == nil {
		meta = l.hasMetaMethod(b, name)
		if meta == nil {
			luautil.Raise("Neither operand has a "+name+" meta method.", luautil.ErrTypGenRuntime)
		}
	}

	f, ok := meta.(*function)
	if !ok {
		luautil.Raise("Meta method "+name+" is not a function.", luautil.ErrTypGenRuntime)
	}

	l.Push(f)
	l.Push(a)
	l.Push(b)
	l.Call(2, 1)
	rtn := l.stack.Get(-1)
	l.Pop(1)
	return rtn
}

func (l *State) arith(op opCode, a, b value) value {
	switch op {
	case OpAdd:
		ia, oka := a.(int64)
		ib, okb := b.(int64)
		if oka && okb {
			return ia + ib
		}

		fa, oka := tryFloat(a)
		fb, okb := tryFloat(b)
		if oka && okb {
			return fa + fb
		}

		return l.tryMathMeta(op, a, b)
	case OpSub:
		ia, oka := a.(int64)
		ib, okb := b.(int64)
		if oka && okb {
			return ia - ib
		}

		fa, oka := tryFloat(a)
		fb, okb := tryFloat(b)
		if oka && okb {
			return fa - fb
		}

		return l.tryMathMeta(op, a, b)
	case OpMul:
		ia, oka := a.(int64)
		ib, okb := b.(int64)
		if oka && okb {
			return ia * ib
		}

		fa, oka := tryFloat(a)
		fb, okb := tryFloat(b)
		if oka && okb {
			return fa * fb
		}

		return l.tryMathMeta(op, a, b)
	case OpMod:
		ia, oka := a.(int64)
		ib, okb := b.(int64)
		if oka && okb {
			return ia % ib
		}

		fa, oka := tryFloat(a)
		fb, okb := tryFloat(b)
		if oka && okb {
			return math.Mod(fa, fb)
		}

		return l.tryMathMeta(op, a, b)
	case OpPow:
		fa, oka := tryFloat(a)
		fb, okb := tryFloat(b)
		if oka && okb {
			return math.Pow(fa, fb)
		}

		return l.tryMathMeta(op, a, b)
	case OpDiv:
		fa, oka := tryFloat(a)
		fb, okb := tryFloat(b)
		if oka && okb {
			return fa / fb
		}

		return l.tryMathMeta(op, a, b)
	case OpIDiv:
		ia, oka := tryInt(a)
		ib, okb := tryInt(b)
		if oka && okb {
			return ia / ib
		}

		return l.tryMathMeta(op, a, b)
	case OpBinAND:
		ia, oka := tryInt(a)
		ib, okb := tryInt(b)
		if oka && okb {
			return ia & ib
		}

		return l.tryMathMeta(op, a, b)
	case OpBinOR:
		ia, oka := tryInt(a)
		ib, okb := tryInt(b)
		if oka && okb {
			return ia | ib
		}

		return l.tryMathMeta(op, a, b)
	case OpBinXOR:
		ia, oka := tryInt(a)
		ib, okb := tryInt(b)
		if oka && okb {
			return ia ^ ib
		}

		return l.tryMathMeta(op, a, b)
	case OpBinShiftL:
		ia, oka := tryInt(a)
		ib, okb := tryInt(b)
		if oka && okb {
			if ib < 0 {
				return int64(uint64(ia) >> uint64(-ib))
			} else {
				return int64(uint64(ia) << uint64(ib))
			}
		}

		return l.tryMathMeta(op, a, b)
	case OpBinShiftR:
		ia, oka := tryInt(a)
		ib, okb := tryInt(b)
		if oka && okb {
			if ib < 0 {
				return int64(uint64(ia) << uint64(-ib))
			} else {
				return int64(uint64(ia) >> uint64(ib))
			}
		}

		return l.tryMathMeta(op, a, b)
	case OpUMinus:
		ia, oka := a.(int64)
		if oka {
			return -ia
		}

		fa, oka := tryFloat(a)
		if oka {
			return -fa
		}

		return l.tryMathMeta(op, a, b)
	case OpBinNot:
		ia, oka := tryInt(a)
		if oka {
			return ^ia
		}

		return l.tryMathMeta(op, a, b)
	default:
		luautil.Raise("Invalid opCode passed to arith", luautil.ErrTypMajorInternal)
		panic("UNREACHABLE")
	}
}

var cmpMeta = [...]string{
	"__eq",
	"__lt",
	"__le", // if this does not exist then try !lt(b, a)
}

func (l *State) tryCmpMeta(op opCode, a, b value) bool {
	if op < OpEqual || op > OpLessOrEqual {
		luautil.Raise("Operator passed to tryCmpMeta out of range.", luautil.ErrTypMajorInternal)
	}
	name := cmpMeta[op-OpEqual]

	var meta value
	tryLEHack := false
try:
	meta = l.hasMetaMethod(a, name)
	if meta == nil {
		meta = l.hasMetaMethod(b, name)
		if meta == nil {
			if name == "__le" {
				tryLEHack = true
				name = "__lt"
				goto try
			}
			if name == "__eq" {
				return a == b // Fall back to raw equality.
			}

			return false
		}
	}

	l.Push(meta)
	if tryLEHack {
		l.Push(b)
		l.Push(a)
	} else {
		l.Push(a)
		l.Push(b)
	}
	l.Call(2, 1)
	rtn := toBool(l.stack.Get(-1))
	l.Pop(1)
	if tryLEHack {
		return !rtn
	}
	return rtn
}

func (l *State) compare(op opCode, a, b value, raw bool) bool {
	tm := true
	t := typeOf(a)
	if t != typeOf(b) {
		tm = false
	}

	switch op {
	case OpEqual:
		if tm {
			switch t {
			case TypNil:
				return true // Obviously.
			case TypNumber:
				ia, oka := a.(int64)
				ib, okb := b.(int64)
				if oka && okb {
					return ia == ib
				}

				fa, oka := a.(float64)
				fb, okb := b.(float64)
				if oka && okb {
					return fa == fb
				}

				// Weird, but this is what the reference implementation does.
				return toInt(a) == toInt(b)

			case TypString:
				return a.(string) == b.(string)
			case TypBool:
				return a.(bool) == b.(bool)
			}
		}

		if raw {
			return a == b
		}
		return l.tryCmpMeta(op, a, b)

	case OpLessThan:
		if tm {
			switch t {
			case TypNumber:
				ia, oka := a.(int64)
				ib, okb := b.(int64)
				if oka && okb {
					return ia < ib
				}

				fa, oka := a.(float64)
				fb, okb := b.(float64)
				if oka && okb {
					return fa < fb
				}

				// Weird, but this is what the reference implementation does.
				return toInt(a) < toInt(b)

			case TypString:
				return a.(string) < b.(string) // Fix me, should be locale sensitive, not lexical
			}
		}

		if raw {
			return false
		}
		return l.tryCmpMeta(op, a, b)

	case OpLessOrEqual:
		if tm {
			switch t {
			case TypNumber:
				ia, oka := a.(int64)
				ib, okb := b.(int64)
				if oka && okb {
					return ia <= ib
				}

				fa, oka := a.(float64)
				fb, okb := b.(float64)
				if oka && okb {
					return fa <= fb
				}

				// Weird, but this is what the reference implementation does.
				return toInt(a) <= toInt(b)

			case TypString:
				return a.(string) <= b.(string) // Fix me, should be locale sensitive, not lexical
			}
		}

		if raw {
			return false
		}
		return l.tryCmpMeta(op, a, b)
	default:
		luautil.Raise("Invalid comparison operator.", luautil.ErrTypGenRuntime)
		panic("UNREACHABLE")
	}
}
