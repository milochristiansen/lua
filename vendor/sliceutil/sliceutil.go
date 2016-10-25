/*
Copyright 2016 by Milo Christiansen

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

// Generic slice utility functions, for when you just want to manipulate a slice with minimum fuss and bother.
// 
// The functions in this package are intended for use when you want something quick and easy. They are not
// designed for speed or safety, but for simplicity and ease of use. If you really want to do things properly
// then make a dedicated type, if you want something quick use these functions.
// 
// Most of the functions in this package do something simple, but with a minor twist designed to avoid
// repetitiveness. For example the following:
// 
//	sliceutil.Push(&test, 5)
// 
// Is equivalent to:
// 
//	test = append(test, 5)
// 
// For this case its not very useful (Push is mostly there to make the API orthogonal), but take this example:
// 
//	return sliceutil.Pop(&test).(int)
// 
// Would need to be expanded to:
// 
//	rtn := test[len(test)-1]
//	test = test[:len(test)-1]
//	return rtn
// 
// Sure, it only saves two lines, but the code is much less repetitive (and so there is less chance of a typo
// causing problems).
// 
// These functions have minimal error handling, unless you like panics make sure you only feed in good input.
package sliceutil

import "reflect"

// Push an item onto the end of a slice.
// Like all function in this package Push takes a slice *pointer* as it's first argument.
// The second argument must be assignable to the slice's fields.
func Push(slice, item interface{}) {
	sv := reflect.Indirect(reflect.ValueOf(slice))
	ns := reflect.Append(sv, reflect.ValueOf(item))
	sv.Set(ns)
}

// Pop an item off the end of a slice and returns it.
// Like all function in this package Pop takes a slice *pointer* as it's first argument.
func Pop(slice interface{}) interface{} {
	sv := reflect.Indirect(reflect.ValueOf(slice))
	l := sv.Len()
	rtn := sv.Index(l - 1)
	sv.Set(sv.Slice(0, l - 1))
	return rtn.Interface()
}

// Top retrieves the last item of a slice and returns it.
// Like all function in this package Top takes a slice *pointer* as it's first argument.
func Top(slice interface{}) interface{} {
	sv := reflect.Indirect(reflect.ValueOf(slice))
	l := sv.Len()
	return sv.Index(l - 1).Interface()
}

// Insert an item at the given index in the slice, shifting the existing items up to make room.
// Like all function in this package Insert takes a slice *pointer* as it's first argument.
func Insert(slice interface{}, at int, item interface{}) {
	sv := reflect.Indirect(reflect.ValueOf(slice))
	l := sv.Len()
	
	if at < 0 || at >= l {
		panic("insertion index out of range")
	}
	
	sv.Set(reflect.Append(sv, reflect.Zero(reflect.TypeOf(item))))
	
	reflect.Copy(sv.Slice(at+1, l+1), sv.Slice(at, l))
	sv.Index(at).Set(reflect.ValueOf(item))
}

// Remove the item at the given index in the slice, shifting the existing items down as needed.
// Like all function in this package Remove takes a slice *pointer* as it's first argument.
func Remove(slice interface{}, at int) {
	sv := reflect.Indirect(reflect.ValueOf(slice))
	l := sv.Len()
	
	if at < 0 || at >= l {
		panic("removal index out of range")
	}
	
	if at == l - 1 {
		sv.Set(sv.Slice(0, l-1))
		return
	}
	
	sv.Set(reflect.AppendSlice(sv.Slice(0, at), sv.Slice(at+1, l)))
}
