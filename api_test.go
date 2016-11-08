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

import "testing"
import "testing/quick"

// The tests in this file do NOT run any Lua code, they just exercise the native API.
// In the same vein no attempt is made to test the compiler here.
//
// These are just quick tests to make sure nothing is obviously broken in the VM native
// API, more extensive script tests based on the official 5.3 test suit will cover
// everything else.
// 
// Note that these tests only cover the most important/common API functions.

// If "ok" is false then fail the test and log the message.
func assert(t *testing.T, ok bool, msg ...interface{}) {
	if !ok {
		t.Error(msg...)
	}
}

// If "ok" is false then fail the test and log the message.
func assertf(t *testing.T, ok bool, format string, msg ...interface{}) {
	if !ok {
		t.Errorf(format, msg...)
	}
}

// Check that the value at the given index is a certain exact type.
func assertTyp(t *testing.T, l *State, typ TypeID, styp STypeID, idx int) {
	assertf(t, l.TypeOf(idx) == typ && l.SubTypeOf(idx) == styp, "Incorrect value: %v/%v vs %v/%v (%v)\n", typ, styp, l.TypeOf(idx), l.SubTypeOf(idx), l.GetRaw(idx))
}

func TestStack(t *testing.T) {
	l := NewState()
	
	/////////////////////////////////////////
	// Test the basics of pushing and popping values.
	
	l.Push("a")
	l.Push(1)
	l.Push(1.0)
	
	assert(t, l.TypeOf(1) == TypString, "Wrong type on stack string vs", l.TypeOf(1))
	assert(t, l.TypeOf(2) == TypNumber, "Wrong type on stack number vs", l.TypeOf(2))
	assert(t, l.SubTypeOf(2) == STypInt, "Wrong subtype on stack int vs", l.SubTypeOf(2))
	assert(t, l.TypeOf(3) == TypNumber, "Wrong type on stack number vs", l.TypeOf(3))
	assert(t, l.SubTypeOf(3) == STypFloat, "Wrong subtype on stack float vs", l.SubTypeOf(3))
	assert(t, l.TypeOf(4) == TypNil, "Wrong type on stack nil vs", l.TypeOf(4))
	
	assertTyp(t, l, TypNumber, STypFloat, -1)
	l.Pop(1)
	assertTyp(t, l, TypNumber, STypInt, -1)
	l.Pop(1)
	assertTyp(t, l, TypString, STypNone, -1)
	l.Pop(1)
	assertTyp(t, l, TypNil, STypNone, -1)
	l.Pop(1)
	assertTyp(t, l, TypNil, STypNone, -1)
	
	// Ensure the stack is clear so the next part of this test doesn't have to worry about it.
	assert(t, l.AbsIndex(-1) == 0, "Items remain on stack after all values popped.")
	
	
	/////////////////////////////////////////
	// Test popping over frame boundaries (or rather test to make sure you can't do it)
	
	l.Push("xyz")
	
	// We need to use the internal API to add a new frame without calling a function.
	l.stack.AddFrame(&function{}, 1, 0, 0)
	
	l.Push(200)
	
	assert(t, l.AbsIndex(-1) == 1, "Can see non-frame values.")
	l.Pop(2)
	assertTyp(t, l, TypNil, STypNone, -1)
	
	l.stack.DropFrame()
	
	assertTyp(t, l, TypString, STypNone, -1)
	l.Pop(1)
	
	assert(t, l.AbsIndex(-1) == 0, "Items remain on stack after all values popped.")
	
	
	/////////////////////////////////////////
	// Test setting and inserting at arbitrary stack indexes.
	
	l.Push(1)
	l.Push(nil)
	l.Push("a")
	l.Push(true)
	
	assertTyp(t, l, TypNil, STypNone, 2)
	l.Push(1.1); l.Set(2, -1); l.Pop(1)
	assertTyp(t, l, TypNumber, STypFloat, 2)
	
	assertTyp(t, l, TypString, STypNone, 3)
	l.Push(nil); l.Insert(3)
	assertTyp(t, l, TypNumber, STypFloat, 2)
	assertTyp(t, l, TypNil, STypNone, 3)
	assertTyp(t, l, TypString, STypNone, 4)
	
	l.Pop(4)
	assertTyp(t, l, TypNumber, STypInt, -1)
	l.Push("a"); l.Insert(-1)
	assertTyp(t, l, TypNumber, STypInt, -1)
	assertTyp(t, l, TypString, STypNone, -2)
	
	l.Pop(2)
	assert(t, l.AbsIndex(-1) == 0, "Items remain on stack after all values popped.")
}

func TestTable(t *testing.T) {
	l := NewState()
	
	/////////////////////////////////////////
	// Test basic raw set/get.
	
	l.NewTable(0, 0)
	tidx := l.AbsIndex(-1)
	
	l.Push("key")
	l.Push(1)
	l.SetTableRaw(tidx)
	
	l.Push(false)
	
	l.Push("key")
	l.GetTableRaw(tidx)
	assertTyp(t, l, TypNumber, STypInt, -1)
	assertTyp(t, l, TypBool, STypNone, -2)
	
	l.Pop(3)
	assert(t, l.AbsIndex(-1) == 0, "Items remain on stack after all values popped.")
	
	// Do some random key tests for all the basic key types.
	// These just make sure the same value gets the same key.
	f2 := func(v1, v2 interface{}) bool {
		l.NewTable(0, 0)
		
		l.Push(v1)
		l.Push(1)
		l.SetTableRaw(-3)
		
		l.Push(false)
		
		l.Push(v2)
		l.GetTableRaw(-3)
		
		ok := l.TypeOf(-1) == TypNumber && l.SubTypeOf(-1) == STypInt &&
			l.TypeOf(-2) == TypBool && l.SubTypeOf(-2) == STypNone
		
		l.Pop(3)
		return ok
	}
	f := func(v interface{}) bool {
		return f2(v, v)
	}
	
	if err := quick.Check(func(v float32) bool { return f(v) }, nil); err != nil { t.Error(err) }
	if err := quick.Check(func(v float64) bool { return f(v) }, nil); err != nil { t.Error(err) }
	if err := quick.Check(func(v int) bool { return f(v) }, nil); err != nil { t.Error(err) }
	if err := quick.Check(func(v int32) bool { return f(v) }, nil); err != nil { t.Error(err) }
	if err := quick.Check(func(v int64) bool { return f(v) }, nil); err != nil { t.Error(err) }
	if err := quick.Check(func(v string) bool { return f(v) }, nil); err != nil { t.Error(err) }
	if err := quick.Check(func(v bool) bool { return f(v) }, nil); err != nil { t.Error(err) }
	
	assert(t, l.AbsIndex(-1) == 0, "Items remain on stack after all values popped.")
	
	// Make sure nil is not a valid key
	assert(t, !f(nil), "Nil acting as valid key.")
	
	// And finally do some equivalent key tests
	assert(t, f2(1, 1.0), "Equivalent keys failing. (1, 1.0)")
	assert(t, f2(5, 5.0), "Equivalent keys failing. (5, 5.0)")
	assert(t, f2(0, 0.0), "Equivalent keys failing. (0, 0.0)")
	assert(t, f2(10000030, 10000030.0), "Equivalent keys failing. (10000030, 10000030.0)")
	assert(t, !f2(1, 1.1), "Non-Equivalent keys succeeding. (1, 1.1)")
	
	// TODO: I should test using functions, tables, and userdata values as keys.
	
	assert(t, l.AbsIndex(-1) == 0, "Items remain on stack after all values popped.")
	
	
	/////////////////////////////////////////
	// Advanced table manipulation.
	
	// Basic table iteration
	l.NewTable(0, 0)
	
	l.Push(0); l.Push(true)
	l.SetTableRaw(-3)
	
	l.Push(1); l.Push(true)
	l.SetTableRaw(-3)
	
	l.Push(2); l.Push(true)
	l.SetTableRaw(-3)
	
	results := [3]bool{}
	l.ForEachInTable(-1, func(){
		results[l.ToInt(-2)] = l.ToBool(-1)
	})
	
	for _, v := range results {
		assert(t, v, "Table iteration produced unexpected results.")
	}
	
	l.Pop(1)
	
	assert(t, l.AbsIndex(-1) == 0, "Items remain on stack after all values popped.")
}
