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

package lmodtable

import "github.com/milochristiansen/lua"

import "strings"
import "sort"

// Open loads the "table" module when executed with "lua.(*State).Call".
//
// It would also be possible to use this with "lua.(*State).Require" (which has some side effects that
// are inappropriate for a core library like this) or "lua.(*State).Preload" (which makes even less
// sense for a core library).
func Open(l *lua.State) int {
	l.NewTable(0, 8) // 7 standard functions
	tidx := l.AbsIndex(-1)

	l.SetTableFunctions(tidx, functions)

	l.Push("table")
	l.PushIndex(tidx)
	l.SetTableRaw(lua.GlobalsIndex)

	// Sanity check
	if l.AbsIndex(-1) != tidx {
		panic("CrashAndBurn!")
	}
	return 1
}

var functions = map[string]lua.NativeFunction{
	"concat": func(l *lua.State) int {
		ln := l.Length(1)
		i := l.OptInt(3, 1)
		j := l.OptInt(4, int64(ln))

		set := make([]string, 0, ln)
		for k := i; k <= j; k++ {
			l.Push(k)
			l.GetTable(1)
			set = append(set, l.ToString(-1)) // Not exactly to spec, but I feel kinda lazy right now...
			l.Pop(1)
		}

		sep := l.OptString(2, "")

		l.Push(strings.Join(set, sep))
		return 1
	},
	"insert": func(l *lua.State) int {
		// tbl, [pos], value

		ln := l.Length(1)
		at := ln + 1
		v := 2
		if l.AbsIndex(-1) > 2 {
			at = int(l.ToInt(2))
			v = 3
		}

		// The easy case, insertion index is the item one passed the end.
		if at == ln+1 {
			l.Push(int64(at))
			l.PushIndex(v)
			l.SetTable(1)
			return 0
		}

		// Shift up
		for i := ln + 1; i > at; i-- {
			l.Push(int64(i))
			l.Push(int64(i - 1))
			l.GetTable(1)
			l.SetTable(1)
		}

		// Then set the required index.
		l.Push(int64(at))
		l.PushIndex(v)
		l.SetTable(1)
		return 0
	},
	"move": func(l *lua.State) int {
		f := l.ToInt(2)
		e := l.ToInt(3)
		t := l.ToInt(4)

		tt := 5
		if l.IsNil(5) {
			tt = 1
		}

		// If we have items to copy...
		if e >= f {
			n := e - f

			if t > e || t <= f || tt != 1 {
				for i := int64(0); i <= n; i++ {
					l.Push(t + i)
					l.Push(f + i)
					l.GetTable(1)
					l.SetTable(tt)
				}
			} else {
				for i := n; i >= 0; i-- {
					l.Push(t + i)
					l.Push(f + i)
					l.GetTable(1)
					l.SetTable(tt)
				}
			}
		}

		l.PushIndex(tt)
		return 1
	},
	"pack": func(l *lua.State) int {
		top := l.AbsIndex(-1)
		l.NewTable(top, 0)
		tidx := top + 1

		l.Push("n")
		l.Push(int64(top))
		l.SetTable(tidx)

		for i := 1; i < tidx; i++ {
			l.Push(int64(i))
			l.PushIndex(i)
			l.SetTable(tidx)
		}

		return 1
	},
	"remove": func(l *lua.State) int {
		ln := int64(l.Length(1))
		at := l.OptInt(2, ln)

		if at != ln && (at < 1 || at > ln) {
			l.Push("Index out of range in call to remove")
			l.Error()
		}

		l.Push(at)
		l.GetTable(1)

		if at != ln {
			// Shift down
			for i := at; i < ln; i++ {
				l.Push(i)
				l.Push(i + 1)
				l.GetTable(1)
				l.SetTable(1)
			}
		}
		l.Push(ln)
		l.Push(nil)
		l.SetTable(1)

		return 1
	},
	"sort": func(l *lua.State) int {
		if l.AbsIndex(-1) == 1 {
			l.Push(func(l *lua.State) int {
				l.Push(l.Compare(1, 2, lua.OpLessThan))
				return 1
			})
		}

		sort.Sort((*tableSorter)(l))
		return 0
	},
	"unpack": func(l *lua.State) int {
		ln := int64(l.Length(1))
		i := l.OptInt(2, 1)
		e := l.OptInt(3, ln)

		if i > e {
			return 0
		}

		n := e - i + 1
		for ; i <= e; i++ {
			l.Push(int64(i))
			l.GetTable(1)
		}

		return int(n)
	},
}

type tableSorter lua.State

func (s *tableSorter) Len() int {
	l := (*lua.State)(s)

	return l.Length(1)
}

func (s *tableSorter) Less(i, j int) bool {
	l := (*lua.State)(s)

	i++ // Lua is 1 based
	j++

	l.PushIndex(2)
	l.Push(int64(i))
	l.GetTable(1)
	l.Push(int64(j))
	l.GetTable(1)
	l.Call(2, 1)
	less := l.ToBool(-1)
	l.Pop(1)
	return less
}

func (s *tableSorter) Swap(i, j int) {
	l := (*lua.State)(s)

	i++ // Lua is 1 based
	j++

	l.Push(int64(j)) // Key for j = i   - j
	l.Push(int64(i)) // Value for j = i - j i
	l.GetTable(1)    // ^^^             - j iv
	l.Push(int64(i)) // Key for i = j   - j iv i
	l.Push(int64(j)) // Value for i = j - j iv i j
	l.GetTable(1)    // ^^^             - j iv i jv
	l.SetTable(1)    // i = j           - j iv
	l.SetTable(1)    // j = i           -
}
