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

package supermeta_test

import "testing"

import "github.com/milochristiansen/lua"
import "github.com/milochristiansen/lua/supermeta"
import "github.com/milochristiansen/lua/testhelp"

// This tests the very basics of the supported types. I should probably write more tests sometime...

// Note that I am not testing a single value. That would not produce the desired effect, since
// `x := 5; supermeta.New(l, x)` is equivalent to `l.Push(5)`. The "root" item needs to be a
// complex type (slice, array, map, struct) for this API to be useful.

type basics struct {
	A string
	b int
}

func TestStructBasics(t *testing.T) {
	l := testhelp.MkState()

	l.Push("x")
	x := &basics{A: "test", b: -5}
	supermeta.New(l, x)
	l.SetTableRaw(lua.GlobalsIndex)

	testhelp.AssertBlock(t, l, `
assert(x.A == "test")

assert(x.b == -5)

x.A = "Lua!"

local ok = pcall(function()
	x.b = 6
end)
assert(not ok)

return x.A
	`, "Lua!")

	testhelp.Assert(t, x.A == "Lua!", "Value did not persist")
}

func TestArrayBasics(t *testing.T) {
	l := testhelp.MkState()

	l.Push("x")
	x := &[5]string{"a", "B", "c", "D", "e"}
	supermeta.New(l, x)
	l.SetTableRaw(lua.GlobalsIndex)

	testhelp.AssertBlock(t, l, `
assert(x[1] == "a")

assert(#x == 5)

assert(x[6] == nil)

x[2] = "New"

return x[2]
	`, "New")

	testhelp.Assert(t, x[1] == "New", "Value did not persist")
}

func TestSliceBasics(t *testing.T) {
	l := testhelp.MkState()

	l.Push("x")
	x := []string{"a", "B", "c", "D", "e"}
	supermeta.New(l, &x)
	l.SetTableRaw(lua.GlobalsIndex)

	testhelp.AssertBlock(t, l, `
assert(x[1] == "a")

assert(#x == 5)

assert(x[6] == nil)

x[2] = "New"

assert(x[2] == "New")

x[#x+1] = "Appended"

return x[2]
	`, "New")

	if len(x) != 6 {
		t.Error("Append failed")
	} else {
		testhelp.Assert(t, x[5] == "Appended", "Value did not persist")
	}
}

// Slices and arrays use the exact same system for iteration
func TestSiceIteration(t *testing.T) {
	l := testhelp.MkState()

	l.Push("x")
	x := []string{"a", "B", "c", "D", "e"}
	supermeta.New(l, &x)
	l.SetTableRaw(lua.GlobalsIndex)

	testhelp.AssertBlock(t, l, `
local rtn = ""

-- Can't use ipairs, as it assumes indexing from 1
for k, v in pairs(x) do
	rtn = rtn..k..v
end

return rtn
	`, "1a2B3c4D5e")
}

func TestSiceTableSet(t *testing.T) {
	l := testhelp.MkState()

	l.Push("x")
	// Won't work. The struct copy of the slice will change, but this copy won't unless the struc contains a pointer (of course)
	//x := []string{}
	//supermeta.New(l, &struct{ Y []string }{x})
	x := &struct{ Y []string }{}
	supermeta.New(l, x)
	l.SetTableRaw(lua.GlobalsIndex)

	testhelp.AssertBlock(t, l, `
x.Y = {"a", "b", "c"}
	`, nil)

	testhelp.Assert(t, len(x.Y) == 3 && x.Y[0] == "a" && x.Y[1] == "b" && x.Y[2] == "c", "Set failed")
}

func TestMapBasics(t *testing.T) {
	l := testhelp.MkState()

	// This is the simplest possible case, simple types for both key and value
	l.Push("x")
	x := map[string]string{"a": "A", "b": "B", "c": "C"}
	supermeta.New(l, &x)
	l.SetTableRaw(lua.GlobalsIndex)

	// But we need something a little more complicated as well, so lets do one with struct keys.
	// (ignore the "b" key since we can't set it)
	l.Push("y")
	y := map[basics]int{basics{A: "A"}: 1, basics{A: "B"}: 2, basics{A: "C"}: 3}
	supermeta.New(l, &y)
	l.SetTableRaw(lua.GlobalsIndex)

	testhelp.AssertBlock(t, l, `
assert(x.a == "A")
assert(x.b == "B")
assert(x.c == "C")

x.b = "X"
assert(x.b == "X")

assert(y[{A = "B"}] == 2)
	`, nil)

	testhelp.Assert(t, x["b"] == "X", "Value did not persist")
}

func TestMapIteration(t *testing.T) {
	l := testhelp.MkState()

	l.Push("x")
	x := map[string]string{"a": "A", "b": "B", "c": "C"}
	supermeta.New(l, &x)
	l.SetTableRaw(lua.GlobalsIndex)

	testhelp.AssertBlock(t, l, `
local r, c = {}, 0
for k, v in pairs(x) do
	r[k] = true
	c = c + 1
end
assert(r.a and r.b and r.c and c == 3)
	`, nil)
}

func TestMapTableSet(t *testing.T) {
	l := testhelp.MkState()
	l.NativeTrace = true

	l.Push("x")
	x := map[basics]int{}
	// We need to wrap the map in a structure so we can trigger the __newindex metamethod instead of simply
	// replacing the existing value.
	supermeta.New(l, &struct{ Y map[basics]int }{x})
	l.SetTableRaw(lua.GlobalsIndex)

	testhelp.AssertBlock(t, l, `
x.Y = {[{A = "A"}] = 1, [{A = "B"}] = 2, [{A = "C"}] = 3}
	`, nil)

	testhelp.Assert(t, x[basics{A: "A"}] == 1, "Set failed")
	testhelp.Assert(t, x[basics{A: "B"}] == 2, "Set failed")
	testhelp.Assert(t, x[basics{A: "C"}] == 3, "Set failed")
}
