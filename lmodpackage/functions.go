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

package lmodpackage

import "github.com/milochristiansen/lua"

// Open loads the "package" module when executed with "lua.(*State).Call".
//
// It would also be possible to use this with "lua.(*State).Require" (which has some side effects that
// are inappropriate for a core library like this) or "lua.(*State).Preload" (which makes even less
// sense for a core library).
//
// The following standard Lua functions/fields are not provided:
//	package.config
//	package.cpath
//	package.loadlib
//	package.path
//	package.searchpath
//
// Additionally only one searcher is added to "package.searchers" by default, it tries to load modules
// from "package.preloaded".
func Open(l *lua.State) int {
	l.NewTable(0, 8)
	tidx := l.AbsIndex(-1)

	// Setup "package.loaded", it may already exist in the registry if someone called Require.
	l.Push("loaded")
	l.Push("_LOADED")
	if l.GetTableRaw(lua.RegistryIndex) != lua.TypTable {
		l.Pop(1)
		l.NewTable(0, 64)
		l.Push("_LOADED")
		l.PushIndex(-2)
		l.SetTableRaw(lua.RegistryIndex)
	}
	l.SetTableRaw(tidx)

	// Now setup "package.preload", it may already exist in the registry if Preload was called.
	l.Push("preload")
	l.Push("_PRELOAD")
	if l.GetTableRaw(lua.RegistryIndex) != lua.TypTable {
		l.Pop(1)
		l.NewTable(0, 16)
		l.Push("_PRELOAD")
		l.PushIndex(-2)
		l.SetTableRaw(lua.RegistryIndex)
	}
	l.SetTableRaw(tidx)

	// Next, "package.searchers".
	l.Push("searchers")
	l.NewTable(8, 0)
	// I only add the preload searcher, other searchers are up to the client.
	l.Push(int64(1))
	l.Push(func(l *lua.State) int {
		l.Push("_PRELOAD")
		l.GetTableRaw(lua.RegistryIndex)
		l.PushIndex(1)
		if l.GetTableRaw(-2) == lua.TypFunction {
			return 1
		}
		l.Push("\n\tpackage.preload['" + l.ToString(1) + "'] not a loader function.")
		return 1
	})
	l.SetTableRaw(-3)
	l.SetTableRaw(tidx)

	// Install the module in the global "package".
	l.Push("package")
	l.PushIndex(tidx)
	l.SetTableRaw(lua.GlobalsIndex)

	// Define "require". For some stupid reason the C Lua stores "package.searchers" as
	// an upval for require (what's wrong with the registry?). For now I do the same, if
	// only as a test of using upvalues from native functions...
	l.Push("searchers")
	l.GetTableRaw(tidx) // Grab a reference to package.searchers to use as an upvalue.
	l.Push("require")
	l.PushClosure(func(l *lua.State) int {
		l.PushIndex(lua.FirstUpVal - 1) // package.searchers
		searchers := l.AbsIndex(-1)

		msg := ""
		c := l.LengthRaw(searchers)
		for i := 1; i <= c; i++ {
			l.Push(int64(i))
			if l.GetTableRaw(searchers) != lua.TypFunction {
				continue // Really should be an error...
			}
			l.PushIndex(1)
			l.Call(1, 2)
			if typ := l.TypeOf(-2); typ != lua.TypFunction {
				if typ != lua.TypNil {
					msg += l.ToString(-2)
				}
				l.Pop(2)
				continue
			}
			l.PushIndex(1)
			l.Insert(-1) // This inserts the module name just below the top item (not counting the item it pops off to insert)
			l.Call(2, 1)

			l.Push("_LOADED")
			l.GetTableRaw(lua.RegistryIndex)
			l.PushIndex(1)
			l.GetTableRaw(-2)

			// If package.loaded[modname] == nil && <return value> == nil
			if l.IsNil(-1) && l.IsNil(-3) {
				// Set package.loaded[modname] to true and return true
				l.Pop(1)
				l.PushIndex(1)
				l.Push(true)
				l.SetTableRaw(-3)
				l.Push(true)
				return 1
			}

			// Else set package.loaded[modname] to <return value> and return <return value>
			l.Pop(1)
			l.PushIndex(1)
			l.PushIndex(-3)
			l.SetTableRaw(-3)
			l.Pop(1)
			return 1
		}

		l.Push("Could not load: " + l.ToString(1) + msg)
		l.Error()
		return 0
	}, -2)
	l.SetTableRaw(lua.GlobalsIndex)
	l.Pop(1) // Pop the reference to package.searchers that we got earlier.

	// Sanity check
	if l.AbsIndex(-1) != tidx {
		panic("FIXME!")
	}
	return 1
}
