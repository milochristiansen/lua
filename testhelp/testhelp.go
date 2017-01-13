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

// Helper functions for running scripts snippets in tests.
package testhelp

import "testing"
import "strings"

import "github.com/milochristiansen/lua"
import "github.com/milochristiansen/lua/lmodbase"
import "github.com/milochristiansen/lua/lmodpackage"
import "github.com/milochristiansen/lua/lmodstring"
import "github.com/milochristiansen/lua/lmodtable"
import "github.com/milochristiansen/lua/lmodmath"

// MkState creates a basic script state and populates it with most of the Lua standard library.
// The custom "string" module extensions are not installed.
func MkState() *lua.State {
	l := lua.NewState()

	// Don't include the string extensions.
	l.Push("_NO_STRING_EXTS")
	l.Push(true)
	l.SetTableRaw(lua.RegistryIndex)

	// Require the standard libraries
	l.Push(lmodbase.Open)
	l.Call(0, 0)
	l.Push(lmodpackage.Open)
	l.Call(0, 0)
	l.Push(lmodstring.Open)
	l.Call(0, 0)
	l.Push(lmodtable.Open)
	l.Call(0, 0)
	l.Push(lmodmath.Open)
	l.Call(0, 0)

	return l
}

// AssertBlock runs a block of Lua code. The test fails if there is an error or if "v"
// does not match the snippet's return value.
func AssertBlock(t *testing.T, l *lua.State, blk string, v interface{}) {
	err := l.LoadText(strings.NewReader(blk), "error", 0)
	if err != nil {
		t.Error(err)
		return
	}

	err = l.PCall(0, 1)
	if err != nil {
		t.Error(err)
		return
	}

	l.Push(v)
	Assertf(t, l.Compare(-1, -2, lua.OpEqual), "Did not return expected value. Returned: %v Expected: %v", l.ToString(-2), l.ToString(-1))
}

// AssertTyp checks that the value at the given index is a certain exact type.
func AssertTyp(t *testing.T, l *lua.State, typ lua.TypeID, styp lua.STypeID, idx int) {
	Assertf(t, l.TypeOf(idx) == typ && l.SubTypeOf(idx) == styp, "Incorrect value: %v/%v vs %v/%v (%v)\n", typ, styp, l.TypeOf(idx), l.SubTypeOf(idx), l.GetRaw(idx))
}

// Assert fails the test and logs the message if "ok" is false.
//
// This is purely a lazy convenience.
func Assert(t *testing.T, ok bool, msg ...interface{}) {
	if !ok {
		t.Error(msg...)
	}
}

// Assertf fails the test and logs the message if "ok" is false.
//
// This is purely a lazy convenience.
func Assertf(t *testing.T, ok bool, format string, msg ...interface{}) {
	if !ok {
		t.Errorf(format, msg...)
	}
}
