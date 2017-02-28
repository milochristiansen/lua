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

package lmodbase

import "strings"
import "github.com/milochristiansen/lua"
import "github.com/milochristiansen/lua/luautil"

// Open loads the "base" module when executed with "lua.(*State).Call".
//
// It would also be possible to use this with "lua.(*State).Require" (which has some side effects that
// are inappropriate for a core library like this) or "lua.(*State).Preload" (which makes even less
// sense for a core library).
//
// The following standard Lua functions/fields are not provided:
//	collectgarbage
//	dofile
//	loadfile
//	xpcall
func Open(l *lua.State) int {
	l.NewTable(0, 32) // 19 standard functions (+4 DNI)
	tidx := l.AbsIndex(-1)

	l.SetTableFunctions(tidx, functions)

	l.PushIndex(lua.GlobalsIndex)
	for k, v := range functions {
		l.Push(k)
		l.Push(v)
		l.SetTableRaw(-3)
	}
	l.Pop(1)

	// Sanity check
	if l.AbsIndex(-1) != tidx {
		panic("iBroke!") // What happens when you throw an iPhone...
	}
	return 1
}

var functions = map[string]lua.NativeFunction{
	"assert": func(l *lua.State) int {
		if l.ToBool(1) {
			return l.AbsIndex(-1)
		}

		msg := l.OptString(2, "Assertion Failed!")
		l.Push(msg)
		l.Error()
		return 0
	},
	// collectgarbage, DNI: I use the Go collector, so many of the uses for this function make no sense.
	// dofile, DNI: This VM will not provide file IO.
	"error": func(l *lua.State) int {
		// TODO: Support level (optional second arg) once proper stack traces are working
		l.PushIndex(1)
		l.Error()
		return 0
	},
	"getmetatable": func(l *lua.State) int {
		typ := l.GetMetaField(1, "__metatable")
		if typ != lua.TypNil {
			return 1
		}

		ok := l.GetMetaTable(1)
		if !ok {
			return 0
		}
		return 1
	},
	"ipairs": func(l *lua.State) int {
		l.Push(func(l *lua.State) int {
			i := l.ToInt(2) + 1
			l.Push(i)
			l.Push(i)
			typ := l.GetTable(1)
			if typ == lua.TypNil {
				return 1
			}
			return 2
		})
		l.PushIndex(1)
		l.Push(int64(0))
		return 3
	},
	"load": func(l *lua.State) int {
		chunk := ""
		name := ""
		if l.TypeOf(1) != lua.TypString {
			l.PushIndex(1)
			l.Call(0, 1)
			v := l.OptString(-1, "")
			l.Pop(1)
			for v != "" {
				chunk += v
				l.PushIndex(1)
				l.Call(0, 1)
				v = l.OptString(-1, "")
				l.Pop(1)
			}

			name = l.OptString(2, "=(load)")
		} else {
			chunk = l.ToString(1)
			name = l.OptString(2, chunk)
		}

		env := 0
		if l.TypeOf(4) != lua.TypNil {
			env = 4
		}

		modeB := false
		modeT := false
		switch l.OptString(3, "bt") {
		case "b":
			modeB = true
		case "t":
			modeT = true
		default:
			modeB = true
			modeT = true
		}

		if modeB {
			err := l.LoadBinary(strings.NewReader(chunk), name, env)
			if err != nil && !modeT {
				l.Push(nil)
				l.Push(err.Error())
				return 2
			} else if err == nil {
				return 1
			}
		}

		if modeT {
			err := l.LoadText(strings.NewReader(chunk), name, env)
			if err != nil {
				l.Push(nil)
				l.Push(err.Error())
				return 2
			}
		}
		return 1
	},
	// loadfile, DNI: This VM will not provide file IO.
	"next": func(l *lua.State) int {
		// WARNING: next is not reentrant! Use getiter for most cases.
		l.PushIndex(2)
		l.Next(1)
		if l.TypeOf(-1) == lua.TypNil {
			return 1
		}
		return 2
	},
	"getiter": func(l *lua.State) int {
		// Get an iterator for the given table. Call the returned value to get key/value pairs.
		// The iterator is self contained and does not need access to the original table.
		// Like next modifying the table during iteration may produce weirdness.
		l.GetIter(1)
		return 1
	},
	"pairs": func(l *lua.State) int {
		typ := l.GetMetaField(1, "__pairs")
		if typ != lua.TypNil {
			l.PushIndex(1)
			l.Call(1, 3)
			return 3
		}

		// pairs does NOT use the default next! Instead I use a custom iterator to get around the fact
		// that next is not reentrant.
		l.GetIter(1)
		l.PushIndex(1) // The iterator does not actually use the table and first key.
		l.Push(nil)
		return 3
	},
	"pcall": func(l *lua.State) int {
		l.Push(true)
		l.Insert(1)

		err := l.PCall(l.AbsIndex(-1)-2, -1)
		if err != nil {
			l.Push(false)
			l.Push(err.Error())
			return 2
		}
		return l.AbsIndex(-1) - 1
	},
	"print": func(l *lua.State) int {
		top := l.AbsIndex(-1)
		for i := 1; i <= top; i++ {
			l.Print(l.ToString(i))
			if i < top {
				l.Print("\t")
			}
		}
		l.Println()
		return 0
	},
	"rawequal": func(l *lua.State) int {
		l.Push(l.CompareRaw(1, 2, lua.OpEqual))
		return 1
	},
	"rawget": func(l *lua.State) int {
		l.PushIndex(2)
		l.GetTableRaw(1)
		return 1
	},
	"rawlen": func(l *lua.State) int {
		l.Push(int64(l.LengthRaw(1)))
		return 1
	},
	"rawset": func(l *lua.State) int {
		l.PushIndex(2)
		l.PushIndex(3)
		l.SetTableRaw(1)
		return 0
	},
	"select": func(l *lua.State) int {
		n := l.AbsIndex(-1)

		if l.TypeOf(1) == lua.TypString && l.ToString(1) == "#" {
			l.Push(int64(l.AbsIndex(-1) - 1))
			return 1
		}

		i := int(l.ToInt(1))
		l.AbsIndex(i)

		if i > n {
			return 0
		}
		return l.AbsIndex(-1) - i
	},
	"setmetatable": func(l *lua.State) int {
		typ := l.GetMetaField(1, "__metatable")
		if typ != lua.TypNil {
			luautil.Raise("Cannot overwrite a meta table that has a __metatable field.", luautil.ErrTypGenRuntime)
		}

		l.PushIndex(2)
		l.SetMetaTable(1)
		l.PushIndex(1)
		return 1
	},
	"tonumber": func(l *lua.State) int {
		l.ConvertNumber(1)
		return 1
	},
	"tostring": func(l *lua.State) int {
		l.ConvertString(1)
		return 1
	},
	"type": func(l *lua.State) int {
		l.Push(l.TypeOf(1).String())
		return 1
	},
	// xpcall, DNI: Meaningless, as this VM has no concept of a message handler.
}
