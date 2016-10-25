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

package lmodmath

import "github.com/milochristiansen/lua"

import "math"
import "math/rand"

// Open loads the "math" module when executed with "lua.(*State).Call".
// 
// It would also be possible to use this with "lua.(*State).Require" (which has some side effects that
// are inappropriate for a core library like this) or "lua.(*State).Preload" (which makes even less
// sense for a core library).
func Open(l *lua.State) int {
	l.NewTable(0, 32) // 27 standard functions
	tidx := l.AbsIndex(-1)
	
	l.SetTableFunctions(tidx, functions)
	
	l.Push("huge")
	l.Push(math.MaxFloat64)
	l.SetTableRaw(tidx)
	
	l.Push("maxinteger")
	l.Push(int64(math.MaxInt64))
	l.SetTableRaw(tidx)
	
	l.Push("mininteger")
	l.Push(int64(math.MinInt64))
	l.SetTableRaw(tidx)
	
	l.Push("pi")
	l.Push(math.Pi)
	l.SetTableRaw(tidx)
	
	l.Push("math")
	l.PushIndex(tidx)
	l.SetTableRaw(lua.GlobalsIndex)
	
	// Sanity check
	if l.AbsIndex(-1) != tidx {
		panic("Oops!")
	}
	return 1
}

var functions = map[string]lua.NativeFunction{
	"abs": func(l *lua.State) int {
		if l.SubTypeOf(1) != lua.STypFloat {
			i, ok := l.TryInt(1)
			if ok {
				if i < 0 {
					i = -i
				}
				l.Push(i)
				return 1
			}
		}
		
		f := l.ToFloat(1)
		f = math.Abs(f)
		l.Push(f)
		return 1
	},
	"acos": func(l *lua.State) int {
		l.Push(math.Acos(l.ToFloat(1)))
		return 1
	},
	"asin": func(l *lua.State) int {
		l.Push(math.Asin(l.ToFloat(1)))
		return 1
	},
	"atan": func(l *lua.State) int {
		y := l.ToFloat(1)
		x := l.OptFloat(2, 1)
		
		l.Push(math.Atan2(y, x))
		return 1
	},
	"ceil": func(l *lua.State) int {
		l.Push(math.Ceil(l.ToFloat(1)))
		return 1
	},
	"cos": func(l *lua.State) int {
		l.Push(math.Cos(l.ToFloat(1)))
		return 1
	},
	"deg": func(l *lua.State) int {
		l.Push(l.ToFloat(1) * (180 / math.Pi)) // This will probably yield a more precise value than standard Lua due to Go's constant math.
		return 1
	},
	"exp": func(l *lua.State) int {
		l.Push(math.Exp(l.ToFloat(1)))
		return 1
	},
	"floor": func(l *lua.State) int {
		l.Push(math.Floor(l.ToFloat(1)))
		return 1
	},
	"fmod": func(l *lua.State) int {
		if l.SubTypeOf(1) != lua.STypFloat && l.SubTypeOf(2) != lua.STypFloat {
			i1, ok1 := l.TryInt(1)
			i2, ok2 := l.TryInt(2)
			if ok1 && ok2 {
				l.Push(i1 % i2)
				return 1
			}
		}
		
		l.Push(math.Mod(l.ToFloat(1), l.ToFloat(2)))
		return 1
	},
	"log": func(l *lua.State) int { // ??? I hate math like this...
		x := l.ToFloat(1)
		base := l.OptFloat(2, math.E)
		l.Push(math.Log(x)/math.Log(base))
		return 1
	},
	"max": func(l *lua.State) int {
		top := l.AbsIndex(-1)
		max := 1
		for i := 1; i <= top; i ++ {
			if l.Compare(max, i, lua.OpLessThan) {
				max = i
			}
		}
		l.PushIndex(max)
		return 1
	},
	"min": func(l *lua.State) int {
		top := l.AbsIndex(-1)
		min := 1
		for i := 1; i <= top; i ++ {
			if l.Compare(i, min, lua.OpLessThan) {
				min = i
			}
		}
		l.PushIndex(min)
		return 1
	},
	"modf": func(l *lua.State) int {
		a, b := math.Modf(l.ToFloat(1))
		l.Push(a)
		if i, ok := l.TryInt(-1); ok {
			l.Pop(1)
			l.Push(i)
		}
		l.Push(b)
		return 2
	},
	"rad": func(l *lua.State) int {
		l.Push(l.ToFloat(1) * (math.Pi / 180))
		return 1
	},
	"random": func(l *lua.State) int {
		switch l.AbsIndex(-1) {
		case 0:
			l.Push(rand.Float64())
		case 1:
			// The range for Int63n is NOT inclusive!
			l.Push(rand.Int63n(l.ToInt(1)) + 1)
		default:
			m := l.ToInt(1)
			n := l.ToInt(2)
			l.Push(rand.Int63n(n - m + 1) + m)
		}
		return 1
	},
	"randomseed": func(l *lua.State) int {
		rand.Seed(l.ToInt(1))
		return 0
	},
	"sin": func(l *lua.State) int {
		l.Push(math.Sin(l.ToFloat(1)))
		return 1
	},
	"sqrt": func(l *lua.State) int {
		l.Push(math.Sqrt(l.ToFloat(1)))
		return 1
	},
	"tan": func(l *lua.State) int {
		l.Push(math.Tan(l.ToFloat(1)))
		return 1
	},
	"tointeger": func(l *lua.State) int {
		if i, ok := l.TryInt(1); ok {
			l.Push(i)
			return 1
		}
		l.Push(nil)
		return 1
	},
	"type": func(l *lua.State) int {
		switch l.SubTypeOf(1) {
		case lua.STypInt:
			l.Push("integer")
		case lua.STypFloat:
			l.Push("float")
		default:
			l.Push(nil)
		}
		return 1
	},
	"ult": func(l *lua.State) int {
		i1 := uint64(l.ToInt(1))
		i2 := uint64(l.ToInt(2))
		
		l.Push(i1 < i2)
		return 1
	},
}
