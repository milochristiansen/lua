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

import "io"
import "io/ioutil"
import "fmt"
import "os"
import "os/exec"
import "runtime"

import "github.com/milochristiansen/lua/luautil"

// Stack

// Push pushes the given value onto the stack.
// If the value is not one of nil, float32, float64, int, int32, int64, string, bool, or
// NativeFunction it is converted to a userdata value before being pushed.
func (l *State) Push(v interface{}) {
	switch v2 := v.(type) {
	case nil:
	case float32:
		v = float64(v2)
	case float64:
	case int:
		v = int64(v2)
	case int32:
		v = int64(v2)
	case int64:
	case string:
	case bool:
	case *table: // These three are needed for when the internal API uses these functions for some reason.
	case *function:
	case *userData:
	case func(l *State) int:
		v = &function{
			native: v2,
			uvDefs: []upValue{{name: "_ENV", index: -1}},
			uvClosed: []bool{true},
			upVals: []value{l.global},
			uvAbsIdxs: []int{-1},
		}
	case NativeFunction:
		v = &function{
			native: v2,
			uvDefs: []upValue{{name: "_ENV", index: -1}},
			uvClosed: []bool{true},
			upVals: []value{l.global},
			uvAbsIdxs: []int{-1},
		}
	default:
		v = &userData{
			data: v2,
		}
	}
	l.stack.Push(v)
}

// PushClosure pushes a native function as a closure.
// All native functions always have at least a single upval, _ENV, but this allows you to set more of them if you wish.
func (l *State) PushClosure(f NativeFunction, v ...int) {
	c := len(v)
	if c == 0 {
		l.Push(f)
		return
	}
	c++
	
	fn := &function{
		native: f,
		uvDefs: make([]upValue, c),
		uvClosed: make([]bool, c),
		upVals: make([]value, c),
		uvAbsIdxs: make([]int, c),
	}
	
	// ALL native functions ALWAYS have their first upvalue set to the global table.
	// This differs from standard Lua, but doesn't hurt anything.
	fn.uvDefs[0].name = "_ENV"
	fn.uvDefs[0].index = 0
	fn.uvClosed[0] = true
	fn.upVals[0] = l.global
	fn.uvAbsIdxs[0] = -1
	
	for i := 1; i < c; i++ {
		fn.uvDefs[i].name = "(native upvalue)"
		fn.uvDefs[i].index = -1
		fn.uvClosed[i] = true
		val := l.get(v[i-1])
		fn.upVals[i] = val
		fn.uvAbsIdxs[i] = -1
	}
	
	l.stack.Push(fn)
}

// PushIndex pushes a copy of the value at the given index onto the stack.
func (l *State) PushIndex(i int) {
	l.stack.Push(l.get(i))
}

// Insert takes the item from the TOS and inserts it at the given stack index.
// Existing items are shifted up as needed, this means that when called with a relative index the item
// does not end up at the given index, but just *under* that index.
func (l *State) Insert(i int) {
	if i >= 1 {
		i = i - 1
	}
	
	v := l.get(-1)
	l.Pop(1)
	
	l.stack.Insert(i, v)
}

// Set sets the value at index d to the value at index s (d = s).
// Trying to set the registry or an invalid index will do nothing.
// Setting an absolute index will never fail, the stack will be extended as needed. Be careful not
// to waste stack space or you could run out of memory!
// This function is mostly for setting up-values and things like that.
func (l *State) Set(d, s int) {
	v := l.get(s)
	switch {
	case d == RegistryIndex:
		// Do nothing.
	case d == GlobalsIndex:
		// Do nothing.
	case d <= FirstUpVal:
		l.stack.cFrame().setUp(d-FirstUpVal, v)
	case d >= 1:
		l.stack.Set(d - 1, v)
	case d < 0:
		l.stack.Set(d, v)
	default:
		// d == 0, do nothing.
	}
}

// Pop removes the top n items from the stack.
func (l *State) Pop(n int) {
	l.stack.Pop(n)
}

// AbsIndex converts the given index into an absolute index.
// Use -1 as the index to get the number of items currently on the stack.
func (l *State) AbsIndex(i int) int {
	if i >= 0 || i <= RegistryIndex {
		return i
	}
	
	// Need to add 2 so we get a 1 based index.
	return l.stack.TopIndex()+i+2
}

// Helper
func (l *State) get(i int) value {
	switch {
	case i == RegistryIndex:
		return l.registry
	case i == GlobalsIndex:
		return l.global
	case i <= FirstUpVal:
		return l.stack.cFrame().getUp(FirstUpVal-i)
	case i > 0:
		return l.stack.Get(i - 1)
	case i < 0:
		return l.stack.Get(i)
	default:
		return nil
	}
}

// TypeOf returns the type of the value at the given index.
// Negative indexes are relative to TOS, positive indexes are absolute.
func (l *State) TypeOf(i int) TypeID {
	return typeOf(l.get(i))
}

// SubTypeOf returns the sub-type of the value at the given index.
// Negative indexes are relative to TOS, positive indexes are absolute.
func (l *State) SubTypeOf(i int) STypeID {
	return sTypeOf(l.get(i))
}

// TryFloat attempts to read the value at the given index as a floating point number.
// Negative indexes are relative to TOS, positive indexes are absolute.
func (l *State) TryFloat(i int) (float64, bool) {
	return tryFloat(l.get(i))
}

// ToFloat reads a floating point value from the stack at the given index.
// Negative indexes are relative to TOS, positive indexes are absolute.
// If the value is not an float and cannot be converted to one this may panic.
func (l *State) ToFloat(i int) float64 {
	return toFloat(l.get(i))
}

// OptFloat is the same as ToFloat, except the given default is returned if the value is nil or non-existent.
func (l *State) OptFloat(i int, d float64) float64 {
	v := l.get(i)
	if v == nil {
		return d
	}
	return toFloat(v)
}

// TryInt attempts to read the value at the given index as a integer number.
// Negative indexes are relative to TOS, positive indexes are absolute.
func (l *State) TryInt(i int) (int64, bool) {
	return tryInt(l.get(i))
}

// ToInt reads an integer value from the stack at the given index.
// Negative indexes are relative to TOS, positive indexes are absolute.
// If the value is not an integer and cannot be converted to one this may panic.
func (l *State) ToInt(i int) int64 {
	return toInt(l.get(i))
}

// OptInt is the same as ToInt, except the given default is returned if the value is nil or non-existent.
func (l *State) OptInt(i int, d int64) int64 {
	v := l.get(i)
	if v == nil {
		return d
	}
	return toInt(v)
}

// ToString reads a value from the stack at the given index and formats it as a string.
// Negative indexes are relative to TOS, positive indexes are absolute.
// This will call a __tostring meta method if provided.
func (l *State) ToString(i int) string {
	v := l.get(i)
	
	meth := l.hasMetaMethod(v, "__tostring")
	if meth != nil {
		l.Push(meth)
		l.Push(v)
		l.Call(1, 1)
		rtn := l.stack.Get(-1)
		l.Pop(1)
		return toString(rtn)
	}
	return toString(v)
}

// OptString is the same as ToString, except the given default is returned if the value is nil or non-existent.
func (l *State) OptString(i int, d string) string {
	if l.IsNil(i) {
		return d
	}
	return l.ToString(i)
}

// ToBool reads a value from the stack at the given index and interprets it as a boolean.
// Negative indexes are relative to TOS, positive indexes are absolute.
func (l *State) ToBool(i int) bool {
	return toBool(l.get(i))
}

// IsNil check if the value at the given index is nil. Nonexistent values are always nil.
// Negative indexes are relative to TOS, positive indexes are absolute.
func (l *State) IsNil(i int) bool {
	return l.get(i) == nil
}

// ToUser reads an userdata value from the stack at the given index.
// Negative indexes are relative to TOS, positive indexes are absolute.
// If the value is not an userdata value this may panic.
func (l *State) ToUser(i int) interface{} {
	v, ok := l.get(i).(*userData)
	if !ok {
		luautil.Raise("Invalid conversion to userdata: Value is not a user value.", luautil.ErrTypGenRuntime)
	}
	return v.data
}

// GetRaw gets the raw data for a Lua value.
// Lua types use the following mapping:
//	nil -> nil
//	number -> int64 or float64
//	string -> string
//	bool -> bool
//	table -> string: "table <pointer as hexadecimal>"
//	function -> string: "function <pointer as hexadecimal>"
//	userdata -> The raw user data value
func (l *State) GetRaw(i int) interface{} {
	v := l.get(i)
	switch v2 := v.(type) {
	case nil:
	case float64:
	case int64:
	case string:
	case bool:
	case *userData:
		return v2.data
	default:
		return toString(v)
	}
	return v
}

// Operators

// Arith performs the specified the arithmetic operator with the top two items on the stack (or just
// the top item for OpUMinus and OpBinNot). The result is pushed onto the stack. See "lua_arith" in
// the Lua 5.3 Reference Manual.
func (l *State) Arith(op opCode) {
	a := l.stack.Get(-2)
	b := a
	if op != OpUMinus && op != OpBinNot {
		b = l.stack.Get(-1)
	}

	l.stack.Pop(2)
	l.stack.Push(l.arith(op, a, b))
}

// Compare performs the specified the comparison operator with the items at the given stack indexes.
// See "lua_compare" in the Lua 5.3 Reference Manual.
func (l *State) Compare(i1, i2 int, op opCode) bool {
	a := l.get(i1)
	b := l.get(i2)
	
	return l.compare(op, a, b, false)
}

// CompareRaw is exactly like Compare, but without meta-methods.
func (l *State) CompareRaw(i1, i2 int, op opCode) bool {
	a := l.get(i1)
	b := l.get(i2)

	return l.compare(op, a, b, true)
}

// Table Access

// NewTable creates a new table with "as" preallocated array elements and "hs" preallocated hash elements.
func (l *State) NewTable(as, hs int) {
	l.stack.Push(newTable(l, as, hs))
}

// GetTable reads from the table at the given index, popping the key from the stack and pushing the result.
// The type of the pushed object is returned.
// This may raise an error if the value is not a table or is lacking the __index meta method.
func (l *State) GetTable(i int) TypeID {
	v := l.getTable(l.get(i), l.stack.Get(-1))
	l.Pop(1)
	l.Push(v)
	return typeOf(v)
}

// GetTableRaw is like GetTable except it ignores meta methods.
// This may raise an error if the value is not a table.
func (l *State) GetTableRaw(i int) TypeID {
	t := l.get(i)
	k := l.stack.Get(-1)
	l.Pop(1)
	
	tbl, ok := t.(*table)
	if !ok {
		luautil.Raise("Value is not a table.", luautil.ErrTypGenRuntime)
	}
	
	v := tbl.GetRaw(k)
	l.Push(v)
	return typeOf(v)
}

// SetTable writes to the table at the given index, popping the key and value from the stack.
// This may raise an error if the value is not a table or is lacking the __newindex meta method.
// The value must be on TOS, the key TOS-1.
func (l *State) SetTable(i int) {
	l.setTable(l.get(i), l.stack.Get(-2), l.stack.Get(-1))
	l.Pop(2)
}

// SetTableRaw is like SetTable except it ignores meta methods.
// This may raise an error if the value is not a table.
func (l *State) SetTableRaw(i int) {
	t := l.get(i)
	k := l.stack.Get(-2)
	v := l.stack.Get(-1)
	l.Pop(2)
	
	tbl, ok := t.(*table)
	if !ok {
		luautil.Raise("Value is not a table.", luautil.ErrTypGenRuntime)
	}
	
	tbl.SetRaw(k, v)
}

// SetTableFunctions does a raw set for each function in the provided map, using it's map key as the table key.
func (l *State) SetTableFunctions(i int, funcs map[string]NativeFunction) {
	i = l.AbsIndex(i)
	
	for k, v := range funcs {
		l.Push(k)
		l.Push(v) // This automatically wraps the native function
		l.SetTableRaw(i) // Me being lazy...
	}
}

// Next is a basic table iterator.
// Pass in the index of a table, Next will pop a key from the stack and push the next key and it's value.
// This function is not reentrant! Iteration order changes with each iteration, so trying to
// do two separate iterations of a single table at the same time will result in all kinds of weirdness.
// If you use this iterator in production code you need your head examined, it is here strictly to power
// the standard library function `next` (which you also should not use).
// See GetIter.
func (l *State) Next(i int) {
	t := l.get(i)
	k := l.stack.Get(-1)
	l.Pop(1)
	
	tbl, ok := t.(*table)
	if !ok {
		luautil.Raise("Value is not a table.", luautil.ErrTypGenRuntime)
	}
	
	nk, nv := tbl.Next(k)
	l.Push(nk)
	l.Push(nv)
}

// GetIter pushes a table iterator onto the stack.
// This value is type "userdata" and has a "__call" meta method. Calling the iterator will
// push the next key/value pair onto the stack. The key is not required for the next
// iteration, so unlike Next you must pop both values.
// The end of iteration is signaled by returning a single nil value.
func (l *State) GetIter(i int) {
	t := l.get(i)
	
	tbl, ok := t.(*table)
	if !ok {
		luautil.Raise("Value is not a table.", luautil.ErrTypGenRuntime)
	}
	
	l.Push(newTableIter(tbl))
	
	l.NewTable(0, 1)
	l.Push("__call")
	l.Push(func(l *State) int {
		i := l.ToUser(1).(*tableIter)
		k, v := i.Next()
		if k == nil {
			l.Push(k)
			return 1
		}
		l.Push(k)
		l.Push(v)
		return 2
	})
	l.SetTableRaw(-3)
	l.SetMetaTable(-2)
	
	// Alternate function version
	//l.Push(func(l *State) int {
	//	i := l.ToUser(FirstUpVal).(*tableIter)
	//	k, v := i.Next()
	//	if k == nil {
	//		l.Push(k)
	//		return 1
	//	}
	//	l.Push(k)
	//	l.Push(v)
	//	return 2
	//})
	//f := l.stack.Get(-1).(*function)
	//f.upVals[0] = &userData{
	//	data: newTableIter(tbl),
	//}
}

// ForEachInTable is a simple wrapper around GetIter and is provided as a convenience.
// 
// The given function is called once for every item in the table at t. For each call of the
// function the value is at -1 and the key at -2. You MUST keep the stack balanced inside
// the function! Do not pop the key and value off the stack before returning!
// Little to no error checking is done, as this is a simple convenience wrapper around
// a common sequence of public API functions.
func (l *State) ForEachInTable(t int, f func()) {
	// I never guessed that FORTH style stack comments would be useful in Go...
	l.GetIter(t) // -- iter
	l.PushIndex(-1) // iter -- iter iter
	l.Call(0, 2) // iter -- key value
	for !l.IsNil(-2) {
		f()
		
		l.Pop(2) // key value --
		l.PushIndex(-1) // iter -- iter iter
		l.Call(0, 2) // iter -- key value
	}
	l.Pop(3) // key value iter --
}

// Other

// SetUpVal sets upvalue "i" in the function at "f" to the value at "v".
// If the upvalue index is out of range or "f" is not a function false is returned
// and nothing is done, else returns true and sets the upval.
func (l *State) SetUpVal(f, i, v int) bool {
	fn, ok := l.get(f).(*function)
	if !ok || i >= len(fn.uvDefs) {
		return false
	}
	
	fn.uvClosed[i] = true
	fn.upVals[i] = l.get(v)
	return true
}

// ConvertNumber gets the value at the given index and converts it to a number
// (preferring int over float) and pushes the result. If this is impossible then
// it pushes nil instead.
func (l *State) ConvertNumber(i int) {
	v := l.get(i)
	if typeOf(v) == TypNumber {
		return
	}
	
	if n, ok := tryInt(v); ok {
		l.Push(n)
		return
	}
	
	if n, ok := tryFloat(v); ok {
		l.Push(n)
		return
	}
	l.Push(nil)
}

// ConvertString gets the value at the given index and converts it to a string
// then pushes the result.
func (l *State) ConvertString(i int) {
	s := l.ToString(i)
	l.Push(s)
}

// DumpFunction converts the Lua function at the given index to a binary chunk. The returned value may
// be used with LoadBinary to get a function equivalent to the dumped function (but without the original
// function's up values).
// 
// Currently the "strip" argument does nothing.
// 
// This (obviously) only works with Lua functions, trying to dump a native function or a non-function
// value will raise an error.
func (l *State) DumpFunction(i int, strip bool) []byte {
	f, ok := l.get(i).(*function)
	if !ok {
		luautil.Raise("Value is not a function.", luautil.ErrTypGenRuntime)
	}
	
	if f.native != nil {
		luautil.Raise("Function cannot be dumped, is native.", luautil.ErrTypGenRuntime)
	}
	
	return dumpBin(&f.proto)
}

// Error pops a value off the top of the stack, converts it to a string, and raises it as an error.
func (l *State) Error() {
	msg := l.ToString(-1)
	l.stack.Pop(1)
	luautil.Raise(msg, luautil.ErrTypGenRuntime)
}

// GetMetaField pushes the meta method with the given name for the item at the given index onto the stack, then
// returns the type of the pushed item.
// If the item does not have a meta table or does not have the specified method this does nothing and returns TypNil
func (l *State) GetMetaField(i int, name string) TypeID {
	meth := l.hasMetaMethod(l.get(i), name)
	if meth != nil {
		l.Push(meth)
	}
	return typeOf(meth)
}

// GetMetaTable gets the meta table for the value at the given index and pushes it onto the stack.
// If the value does not have a meta table then this returns false and pushes nothing.
func (l *State) GetMetaTable(i int) bool {
	meta := l.getMetaTable(l.get(i))
	if meta != nil {
		l.Push(meta)
		return true
	}
	return false
}

// SetMetaTable pops a table from the stack and sets it as the meta table of the value at the given index.
// If the value is not a userdata or table then the meta table is set for ALL values of that type!
func (l *State) SetMetaTable(i int) {
	v := l.get(i)
	tbl, ok := l.stack.Get(-1).(*table)
	l.stack.Pop(1)
	if !ok {
		luautil.Raise("Value is not a table.", luautil.ErrTypGenRuntime)
	}
	
	switch v2 := v.(type) {
	case nil:
		l.metaTbls[TypNil] = tbl
	case float64:
		l.metaTbls[TypNumber] = tbl
	case int64:
		l.metaTbls[TypNumber] = tbl
	case string:
		l.metaTbls[TypString] = tbl
	case bool:
		l.metaTbls[TypBool] = tbl
	case *table:
		v2.meta = tbl
	case *function:
		l.metaTbls[TypFunction] = tbl
	case *userData:
		v2.meta = tbl
	default:
		luautil.Raise("Invalid type passed to SetMetaTable.", luautil.ErrTypMajorInternal)
	}
}

// Returns the "length" of the item at the given index, exactly like the "#" operator would.
// If this calls a meta method it may raise an error if the length is not an integer.
func (l *State) Length(i int) int {
	v := l.get(i)
	
	if s, ok := v.(string); ok {
		return len(s)
	}
	
	meth := l.hasMetaMethod(v, "__len")
	if meth != nil {
		f, ok := meth.(*function)
		if !ok {
			luautil.Raise("Meta method __len is not a function.", luautil.ErrTypGenRuntime)
		}
		
		l.Push(f)
		l.Push(v)
		l.Call(1, 1)
		rtn := l.stack.Get(-1)
		l.Pop(1)
		return int(toInt(rtn))
	}
	
	tbl, ok := v.(*table)
	if !ok {
		luautil.Raise("Value is not a string or table and has no __len meta method.", luautil.ErrTypGenRuntime)
	}
	return tbl.Length()
}

// Returns the length of the table or string at the given index. This does not call meta methods.
// If the value is not a table or string this will raise an error.
func (l *State) LengthRaw(i int) int {
	v := l.get(i)
	
	if s, ok := v.(string); ok {
		return len(s)
	}
	
	tbl, ok := v.(*table)
	if !ok {
		luautil.Raise("Value is not a string or table.", luautil.ErrTypGenRuntime)
	}
	return tbl.Length()
}

// SetGlobal pops a value from the stack and sets it as the new value of global name.
func (l *State) SetGlobal(name string) {
	v := l.stack.Get(-1)
	l.stack.Pop(1)
	l.global.SetRaw(name, v)
}

// Require calls the given loader (with name as an argument) if there is no entry for "name" in package.loaded.
// The result from the call is stored in package.loaded, and if global is true, in a global variable named "name".
// In any case the module value is pushed onto the stack.
func (l *State) Require(name string, loader NativeFunction, global bool) {
	// This is the index C Lua uses. Do not assume it is properly set yet.
	loaded, ok := l.registry.GetRaw("_LOADED").(*table)
	if ok {
		if mod := loaded.GetRaw(name); mod != nil {
			l.stack.Push(mod)
			return
		}
	} else {
		// The first time this function is called it needs to initialize what will become "package.loaded".
		loaded = newTable(l, 0, 64)
		l.registry.SetRaw("_LOADED", loaded)
	}
	
	l.Push(loader)
	l.Push(name)
	l.Call(1, 1)
	if global {
		l.global.SetRaw(name, l.stack.Get(-1))
	}
	loaded.SetRaw(name, l.stack.Get(-1))
}

// Preload adds the given loader function to "package.preload" for use with "require".
func (l *State) Preload(name string, loader NativeFunction) {
	// This is the index C Lua uses. Do not assume it is properly set yet.
	loaded, ok := l.registry.GetRaw("_PRELOAD").(*table)
	if !ok {
		// The first time this function is called it needs to initialize what will become "package.preload".
		loaded = newTable(l, 0, 16)
		l.registry.SetRaw("_PRELOAD", loaded)
	}
	
	// Lazy, lazy...
	l.Push(loader)
	fn := l.get(-1)
	l.Pop(1)
	
	loaded.SetRaw(name, fn)
}

// Test prints some stack information for sanity checking during test runs.
func (l *State) Test() {
	l.Println("+++++")
	l.Println("D:", len(l.stack.data))
	l.Println("F:", len(l.stack.frames))
	l.Println("TOS:", toString(l.stack.Get(-1)))
	l.Println("-----")
}

// DebugValue prints internal information about a script value.
func (l *State) DebugValue(i int) {
	l.Println("+++++")
	l.Println("I:", i)
	l.Printf("V: %#v\n", l.get(i))
	l.Println("-----")
}

// ListFunc prints an assembly listing of the given function's code.
func (l *State) ListFunc(i int) {
	f, ok := l.get(i).(*function)
	if !ok {
		luautil.Raise("Value is not a function.", luautil.ErrTypGenRuntime)
	}
	
	if f.native != nil {
		luautil.Raise("Function cannot be listed, is native.", luautil.ErrTypGenRuntime)
	}
	
	l.Println(f.proto.String())
}

// Execution

func (l *State) asFunc(proto *funcProto, env *table) *function {
	f := &function{
		proto: *proto,
		uvDefs: proto.upVals,
		uvClosed: make([]bool, len(proto.upVals)),
		upVals: make([]value, len(proto.upVals)),
		uvAbsIdxs: make([]int, len(proto.upVals)),
	}
	for i := range f.uvAbsIdxs {
		f.uvAbsIdxs[i] = -1
	}
	
	f.uvClosed[0] = true
	f.upVals[0] = env
	
	return f
}

// LoadBinary loads a binary chunk into memory and pushes the result onto the stack.
// If there is an error it is returned and nothing is pushed.
// Set env to 0 to use the default environment.
func (l *State) LoadBinary(in io.Reader, name string, env int) error {
	proto, err := loadBin(in, name)
	if err != nil {
		return err
	}
	
	envv := l.global
	if env != 0 {
		ok := false
		envv, ok = l.get(env).(*table)
		if !ok {
			return luautil.Error{Msg: "Value used as environment is not a table.", Type: luautil.ErrTypGenRuntime}
		}
	}
	
	l.stack.Push(l.asFunc(proto, envv))
	return nil
}

// LoadText loads a text chunk into memory and pushes the result onto the stack.
// If there is an error it is returned and nothing is pushed.
// Set env to 0 to use the default environment.
//
// This version uses my own compiler. This compiler does not produce code identical to the standard Lua
// compiler for all syntax constructs, sometimes it is a little worse, rarely a little better.
func (l *State) LoadText(in io.Reader, name string, env int) error {
	source, err := ioutil.ReadAll(in)
	if err != nil {
		return err
	}
	proto, err := compSource(string(source), name, 1)
	if err != nil {
		return err
	}
	
	envv := l.global
	if env != 0 {
		ok := false
		envv, ok = l.get(env).(*table)
		if !ok {
			return luautil.Error{Msg: "Value used as environment is not a table.", Type: luautil.ErrTypGenRuntime}
		}
	}
	
	l.stack.Push(l.asFunc(proto, envv))
	return nil
}

// LoadTextExternal loads a text chunk into memory and pushes the result onto the stack.
// If there is an error it is returned and nothing is pushed.
// Set env to 0 to use the default environment.
// 
// This version looks for and runs "luac" to compile the chunk. Make sure luac is on
// your path.
// 
// This functions is not safe for concurrent use.
func (l *State) LoadTextExternal(in io.Reader, name string, env int) error {
	outFile := os.TempDir() + "/dctech.lua.bin" // Go seems to lack a function to get a temporary file name, so this is unsafe for concurrent use!
	cmd := exec.Command("luac", "-o", outFile, "-")
	cmd.Stdin = in

	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := string(out)
		if msg == "" {
			msg = "Error starting luac"
		}
		return luautil.Error{Msg: msg, Type: luautil.ErrTypWrapped, Err: err}
	}
	
	file, err := os.Open(outFile)
	if err != nil {
		return luautil.Error{Msg: "Error opening luac output file", Type: luautil.ErrTypWrapped, Err: err}
	}
	defer file.Close()
	
	envv := l.global
	if env != 0 {
		ok := false
		envv, ok = l.get(env).(*table)
		if !ok {
			return luautil.Error{Msg: "Value used as environment is not a table.", Type: luautil.ErrTypGenRuntime}
		}
	}
	
	proto, err := loadBin(file, name)
	if err != nil {
		return err
	}
	l.stack.Push(l.asFunc(proto, envv))
	return nil
}

// Call runs a function with the given number of arguments and results.
// The function must be on the stack just before the first argument.
// If this raises an error the stack is NOT unwound! Call this only from
// code that is below a call to PCall unless you want your State to be
// permanently trashed!
func (l *State) Call(args, rtns int) {
	if args < 0 {
		luautil.Raise("Cannot use Call if arg count is unknown.", luautil.ErrTypGenRuntime)
	}
	
	fi := -(args + 1) // Generate a relative index for the function
	l.call(fi, args, rtns, false)
}

// PCall is exactly like Call, except instead of panicking when it encounters an error the
// error is cleanly recovered and returned.
// On error the stack is reset to the way it was before the call minus the function and it's arguments,
// the State may then be reused.
func (l *State) PCall(args, rtns int) (err error) {
	frames := len(l.stack.frames)
	top := len(l.stack.data) - args - 1
	defer func() {
		e := recover()
		if e != nil {
			// Compile a stack trace.
			// TODO: The produced trace is terrible, do this properly.
			sources := []string{}
			lines := []int{}
			for i := len(l.stack.frames) - 1; i >= frames; i-- {
				frame := l.stack.frames[i]
				if frame.fn.native == nil {
					sources = append(sources, frame.fn.proto.source)
					if int(frame.pc) < len(frame.fn.proto.lineInfo) {
						lines = append(lines, frame.fn.proto.lineInfo[frame.pc])
					} else if len(frame.fn.proto.lineInfo) > 0 {
						lines = append(lines, frame.fn.proto.lineInfo[len(frame.fn.proto.lineInfo)-1])
					} else {
						lines = append(lines, -1)
					}
				} else {
					sources = append(sources, "(native code)")
					lines = append(lines, -1)
				}
			}
			
			trace := ""
			for i := range sources {
				if lines[i] == -1 {
					trace += fmt.Sprintf("\n    \"%v\"", sources[i])
					continue
				}
				trace += fmt.Sprintf("\n    \"%v\": <line: %v>", sources[i], lines[i])
			}
			
			if l.NativeTrace {
				buf := make([]byte, 4096)
				buf = buf[:runtime.Stack(buf, true)]
				trace = fmt.Sprintf("%v\n\nNative Trace:\n%s\n", trace, buf)
			}
			
			// Make sure the stack is back to the way we found it, minus the function and it's arguments.
			l.stack.frames = l.stack.frames[:frames]
			for i := len(l.stack.data) - 1; i >= top; i-- {
				l.stack.data[i] = nil
			}
			l.stack.data = l.stack.data[:top]
			
			// Attach the stack trace to the error
			switch e2 := e.(type) {
			case luautil.Error:
				e2.Trace = trace
				err = e2
			case error:
				err = luautil.Error{Type: luautil.ErrTypWrapped, Err: e2, Trace: trace}
			default:
				err = luautil.Error{Type: luautil.ErrTypEvil, Err: fmt.Errorf("%v", e), Trace: trace}
			}
		}
	}()
	
	l.Call(args, rtns)
	return nil
}
