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

package supermeta

import "github.com/milochristiansen/lua"

import "reflect"

func sliceTable(l *lua.State, obj reflect.Value) {
	l.Push(obj)
	l.NewTable(0, 4)

	vt := obj.Type().Elem()
	vk := vt.Kind()

	l.Push("__index")
	l.Push(func(l *lua.State) int {
		o := l.ToUser(1).(reflect.Value)

		k, ok := l.TryInt(2)
		if !ok {
			return 0
		}
		k--

		if k >= int64(o.Len()) || k < 0 {
			return 0
		}

		to(vk)(l, o.Index(int(k)))
		return 1
	})
	l.SetTableRaw(-3)

	l.Push("__newindex")
	l.Push(func(l *lua.State) int {
		o := l.ToUser(1).(reflect.Value)

		k, ok := l.TryInt(2)
		if !ok {
			return 0
		}
		k--

		ln := int64(o.Len())
		if k > ln || k < 0 {
			return 0
		}
		if k == ln {
			if o.Kind() != reflect.Slice {
				return 0
			}
			o.Set(reflect.Append(o, reflect.Zero(vt)))
		}

		err := from(vk)(l, o.Index(int(k)), 3)
		if err != nil {
			l.Push(err.Error())
			l.Error()
		}
		return 0
	})
	l.SetTableRaw(-3)

	l.Push("__len")
	l.Push(func(l *lua.State) int {
		o := l.ToUser(1).(reflect.Value)

		l.Push(o.Len())
		return 1
	})
	l.SetTableRaw(-3)

	l.Push("__pairs")
	l.Push(func(l *lua.State) int {
		l.Push(func(l *lua.State) int {
			o := l.ToUser(1).(reflect.Value)

			i := int(l.ToInt(2))
			i++
			if i <= o.Len() {
				l.Push(int64(i))
				to(vk)(l, o.Index(i-1))
				return 2
			}
			l.Push(nil)
			return 1
		})
		l.PushIndex(1)
		l.Push(int64(0))
		return 3
	})
	l.SetTableRaw(-3)

	l.SetMetaTable(-2)
}

type mapIter struct {
	i    int
	keys []reflect.Value
	self reflect.Value
}

func mapTable(l *lua.State, obj reflect.Value) {
	l.Push(obj)
	l.NewTable(0, 3)

	kk := obj.Type().Key().Kind()
	kt := obj.Type().Key()
	vk := obj.Type().Elem().Kind()
	vt := obj.Type().Elem()

	l.Push("__index")
	l.Push(func(l *lua.State) int {
		o := l.ToUser(1).(reflect.Value)

		k := reflect.New(kt).Elem()
		err := from(kk)(l, k, 2)
		if err != nil {
			l.Push(err.Error())
			l.Error()
		}
		// k is now a valid key for o

		to(vk)(l, o.MapIndex(k))
		return 1
	})
	l.SetTableRaw(-3)

	l.Push("__newindex")
	l.Push(func(l *lua.State) int {
		o := l.ToUser(1).(reflect.Value)

		k := reflect.New(kt).Elem()
		err := from(kk)(l, k, 2)
		if err != nil {
			l.Push(err.Error())
			l.Error()
		}

		v := reflect.New(vt).Elem()
		err = from(vk)(l, v, 3)
		if err != nil {
			l.Push(err.Error())
			l.Error()
		}

		o.SetMapIndex(k, v)
		return 0
	})
	l.SetTableRaw(-3)

	l.Push("__pairs")
	l.Push(func(l *lua.State) int {
		l.Push(func(l *lua.State) int {
			iter := l.ToUser(1).(*mapIter)

			iter.i++
			if iter.i < len(iter.keys) {
				to(kk)(l, iter.keys[iter.i])
				to(vk)(l, iter.self.MapIndex(iter.keys[iter.i]))
				return 2
			}
			l.Push(nil)
			return 1
		})
		l.Push(&mapIter{
			i:    -1,
			keys: obj.MapKeys(),
			self: obj,
		})
		l.Push(nil)
		return 3
	})
	l.SetTableRaw(-3)

	l.SetMetaTable(-2)
}

func structTable(l *lua.State, obj reflect.Value) {
	l.Push(obj)
	l.NewTable(0, 2)

	l.Push("__index")
	l.Push(func(l *lua.State) int {
		o := l.ToUser(1).(reflect.Value)

		k := l.ToString(2)

		f, ok := o.Type().FieldByName(k)
		if !ok {
			return 0
		}

		to(f.Type.Kind())(l, o.FieldByName(k))
		return 1
	})
	l.SetTableRaw(-3)

	l.Push("__newindex")
	l.Push(func(l *lua.State) int {
		o := l.ToUser(1).(reflect.Value)

		k := l.ToString(2)

		fv := o.FieldByName(k)
		if !fv.IsValid() || !fv.CanSet() {
			return 0
		}

		err := from(fv.Type().Kind())(l, fv, 3)
		if err != nil {
			l.Push(err.Error())
			l.Error()
		}
		return 0
	})
	l.SetTableRaw(-3)

	l.SetMetaTable(-2)
}
