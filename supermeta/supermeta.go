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

/*
Generic meta-table types and helpers.

This API mostly ignores exact types, relying on "kinds" (reflect.Kind) instead. This means that
custom types will work fine.

The following kinds are not supported:

	Complex64
	Complex128
	Chan
	Func
	UnsafePointer

These are not implemented because they have no corresponding Lua types, some of these may be
supported later...

Slices are special in that they allow you to write to the key one passed the end of the slice.
This allows you to append now data to an existing slice. If you want anything more complicated
than that you should write a custom metatable.

You should be able to assign a table to a value containing a "complex" type (slice, array, map,
struct). This does not create a new object (unless the existing object is a nil pointer), instead
the data from the table is used to fill as many keys in the object as possible. Slices will be
lengthened as needed, arrays will simply ignore extra items. If any key or value in the table
cannot be converted to the required type conversion will halt with an error.

Slices an arrays are indexed from 1. Basically I just subtract one from all incoming indexes.
This is done to better fit with the rest of Lua, not because I like 1-based indexing (I actually
think it is a really stupid idea, and Lua's biggest problem).

I make a fairly good effort at auto-vivification, any assignment to a nil pointer should result
in a new object being created. Sadly this cannot be done for nil interfaces (for obvious reasons).

Note that this API is somewhat fragile. Things should "just work", but malformed input
is likely to result in lots of errors. Nothing should panic or otherwise crash, but
returned errors are entirely possible. Generally I assume you are feeding in good input
(from both sides). Rather than trying to detect and handle cases where there is a
mix of good and bad, I simply give up at the first issue.

When working with untrusted scripts be very careful what you expose with this API! Some actions
have the potential to trash the exposed data! Always use a set of dedicated metatables where
possible!
*/
package supermeta

import "github.com/milochristiansen/lua"

import "reflect"
import "errors"

// New pushes the given object onto the Lua stack with a generic meta-table applied or
// otherwise converted to something Lua can use. In the case of simple values the
// conversion is by-value, slices, arrays, maps, structs, etc are by-reference. For
// simple values you should probably just use State.Push.
//
// 99% of the time you will want to pass the address of the item you want to work with,
// even if it is a type you would normally pass by value. Unless you pass a pointer
// the reflection library probably won't be able to modify the item.
//
// In the case of an unconvertible value you will either get a Lua error or a nil value.
// Invalid conversions from Lua to Go result in script errors, invalid conversions from
// Go to Lua result in (Lua) nils.
//
// Depending on the exposed value you may or may not be able to set it. Structs in
// particular may raise script errors when trying to set certain fields. Nil pointers
// and interfaces may also be a problem. I make an effort to auto-vivify nil pointers,
// but this may not work for some types.
func New(l *lua.State, obj interface{}) {
	rval := reflect.ValueOf(obj)
	to(rval.Kind())(l, rval)
}

// RValueToLValue pushes the given reflect.Value onto the stack.
//
// This is basically the same as New, but for use when you have an existing reflect.Value.
func RValueToLValue(l *lua.State, src reflect.Value) {
	to(src.Kind())(l, src)
}

// LValueToRValue stores a given Lua value in the given reflect.Value.
//
// If the given reflect.Value cannot hold the requested Lua value you will get an error.
// Note that the reflect.Value may have been modified! Not all errors happen immediately!
// For example if you are assigning a table to a map some of the key/value pairs may have
// been added before the "bad" key was found.
//
// Most of the time you should use the normal API to get values out of the VM!
func LValueToRValue(l *lua.State, dest reflect.Value, src int) error {
	return from(dest.Kind())(l, dest, src)
}

// Possible errors.
var ErrCantSet = errors.New("Cannot set given value.")
var ErrCantConv = errors.New("Conversion to given type not implemented.")
var ErrBadConv = errors.New("Conversion to required type not possible for this value.")

// The "to" functions push a Lua value for the given reflect.Value, the "from" functions grab a Lua
// value and store it in the given reflect.Value if possible (returning an error if not possible).
//
// The reflect.Value passed to these functions must hold (or be able to hold) a value of the correct kind!

// init64
func toInt(l *lua.State, src reflect.Value) {
	l.Push(src.Int())
}
func fromInt(l *lua.State, dest reflect.Value, src int) error {
	if !dest.CanSet() {
		return ErrCantSet
	}

	v, ok := l.TryInt(src)
	if !ok {
		return ErrBadConv
	}

	dest.SetInt(v)
	return nil
}

// uint64
func toUInt(l *lua.State, src reflect.Value) {
	l.Push(int64(src.Uint()))
}
func fromUInt(l *lua.State, dest reflect.Value, src int) error {
	if !dest.CanSet() {
		return ErrCantSet
	}

	v, ok := l.TryInt(src)
	if !ok {
		return ErrBadConv
	}

	dest.SetUint(uint64(v))
	return nil
}

// string
func toString(l *lua.State, src reflect.Value) {
	l.Push(src.String())
}
func fromString(l *lua.State, dest reflect.Value, src int) error {
	if !dest.CanSet() {
		return ErrCantSet
	}

	dest.SetString(l.ToString(src))
	return nil
}

// float64
func toFloat(l *lua.State, src reflect.Value) {
	l.Push(src.Float())
}
func fromFloat(l *lua.State, dest reflect.Value, src int) error {
	if !dest.CanSet() {
		return ErrCantSet
	}

	v, ok := l.TryFloat(src)
	if !ok {
		return ErrBadConv
	}

	dest.SetFloat(v)
	return nil
}

// bool
func toBool(l *lua.State, src reflect.Value) {
	l.Push(src.Bool())
}
func fromBool(l *lua.State, dest reflect.Value, src int) error {
	if !dest.CanSet() {
		return ErrCantSet
	}

	dest.SetBool(l.ToBool(src))
	return nil
}

// []type
func toSlice(l *lua.State, src reflect.Value) {
	sliceTable(l, src)
}
func fromSlice(l *lua.State, dest reflect.Value, src int) (err error) {
	defer l.Recover(0, false)(&err) // So stuff in here can raise errors!

	if !dest.CanSet() {
		return ErrCantSet
	}
	src = l.AbsIndex(src) // Make sure we can safely push more items

	// Check if src is a userdata item of the needed type.
	if l.TypeOf(src) == lua.TypUserData {
		v := l.ToUser(src)
		if rv, ok := v.(reflect.Value); ok && rv.Type() == dest.Type() {
			dest.Set(rv)
			return nil
		}

		if reflect.TypeOf(v) == dest.Type() {
			dest.Set(reflect.ValueOf(v))
			return nil
		}
	}

	i := l.Length(src)

	// Based on code from "encoding/json"
	if dest.Kind() == reflect.Slice {
		// Grow slice if necessary
		if i > dest.Cap() {
			newcap := dest.Cap() + dest.Cap()/2
			if newcap < 4 {
				newcap = 4
			}
			newv := reflect.MakeSlice(dest.Type(), dest.Len(), newcap)
			reflect.Copy(newv, dest)
			dest.Set(newv)
		}
		if i > dest.Len() {
			dest.SetLen(i)
		}
	}

	// Adjust i
	if i > dest.Len() {
		i = dest.Len()
	}

	for j := 1; j <= i; j++ {
		d := dest.Index(j - 1)

		l.Push(j)
		l.GetTable(src)
		err := from(d.Kind())(l, d, -1)
		l.Pop(1)
		if err != nil {
			return err
		}
	}
	return nil
}

// map[type]type
func toMap(l *lua.State, src reflect.Value) {
	mapTable(l, src)
}
func fromMap(l *lua.State, dest reflect.Value, src int) (err error) {
	defer l.Recover(0, false)(&err) // So stuff in here can raise errors!

	if !dest.CanSet() {
		return ErrCantSet
	}
	src = l.AbsIndex(src) // Make sure we can safely push more items

	// Check if src is a userdata item of the needed type.
	if l.TypeOf(src) == lua.TypUserData {
		v := l.ToUser(src)
		if rv, ok := v.(reflect.Value); ok && rv.Type() == dest.Type() {
			dest.Set(rv)
			return nil
		}

		if reflect.TypeOf(v) == dest.Type() {
			dest.Set(reflect.ValueOf(v))
			return nil
		}
	}

	dt := dest.Type()
	kt := dt.Key()
	kc := from(kt.Kind())
	vt := dt.Elem()
	vc := from(vt.Kind())
	l.ForEach(src, func() bool {
		k := reflect.New(kt).Elem()
		err = kc(l, k, -2)
		if err != nil {
			return false
		}

		v := reflect.New(vt).Elem()
		err = vc(l, v, -1)
		if err != nil {
			return false
		}
		dest.SetMapIndex(k, v)
		return true
	})
	return err
}

// struct
func toStruct(l *lua.State, src reflect.Value) {
	structTable(l, src)
}
func fromStruct(l *lua.State, dest reflect.Value, src int) (err error) {
	defer l.Recover(0, false)(&err) // So stuff in here can raise errors!

	if !dest.CanSet() {
		return ErrCantSet
	}
	src = l.AbsIndex(src) // Make sure we can safely push more items.

	// Check if src is a userdata item of the needed type.
	if l.TypeOf(src) == lua.TypUserData {
		v := l.ToUser(src)
		if rv, ok := v.(reflect.Value); ok && rv.Type() == dest.Type() {
			dest.Set(rv)
			return nil
		}

		if reflect.TypeOf(v) == dest.Type() {
			dest.Set(reflect.ValueOf(v))
			return nil
		}
	}

	// Iterate all structure fields looking for matching fields in the table.
	ln := dest.NumField()
	dt := dest.Type()
	for i := 0; i < ln; i++ {
		fi := dt.Field(i)
		l.Push(fi.Name)
		l.GetTable(src)

		if l.IsNil(-1) {
			l.Pop(1)
			continue
		}

		err := from(fi.Type.Kind())(l, dest.Field(i), -1)
		l.Pop(1)
		if err != nil {
			return err
		}
	}
	return nil
}

// *type
func toPtr(l *lua.State, src reflect.Value) {
	src = src.Elem()
	to(src.Kind())(l, src)
}
func fromPtr(l *lua.State, dest reflect.Value, src int) error {
	dest = dest.Elem()
	
	if !dest.IsValid() {
		return ErrCantConv
	}

	// If nil and a concrete type make new item.
	// This *should* be all that is needed for auto-vivification.
	if dest.IsNil() {
		switch dest.Kind() {
		case reflect.String,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Float32, reflect.Float64,
			reflect.Bool, reflect.Struct, reflect.Ptr, reflect.Array:
			dest.Set(reflect.New(dest.Type()).Elem())
		case reflect.Slice:
			dest.Set(reflect.MakeSlice(dest.Type(), 0, 0))
		case reflect.Map:
			dest.Set(reflect.MakeMap(dest.Type()))
		default:
			return ErrCantConv
		}
	}

	return from(dest.Kind())(l, dest, src)
}

// Default case
func toErr(l *lua.State, src reflect.Value) {
	l.Push(nil)
}
func fromErr(l *lua.State, dest reflect.Value, src int) error {
	return ErrCantConv
}

func to(k reflect.Kind) func(*lua.State, reflect.Value) {
	switch k {
	case reflect.String:
		return toString
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return toUInt
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return toInt
	case reflect.Float32, reflect.Float64:
		return toFloat
	case reflect.Bool:
		return toBool
	case reflect.Array, reflect.Slice:
		return toSlice
	case reflect.Map:
		return toMap
	case reflect.Struct:
		return toStruct
	case reflect.Ptr, reflect.Interface:
		return toPtr
	}
	return toErr
}

func from(k reflect.Kind) func(*lua.State, reflect.Value, int) error {
	switch k {
	case reflect.String:
		return fromString
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return fromUInt
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fromInt
	case reflect.Float32, reflect.Float64:
		return fromFloat
	case reflect.Bool:
		return fromBool
	case reflect.Array, reflect.Slice:
		return fromSlice
	case reflect.Map:
		return fromMap
	case reflect.Struct:
		return fromStruct
	case reflect.Ptr, reflect.Interface:
		return fromPtr
	}
	return fromErr
}
