/*
Copyright 2019 by ofunc

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

package lmodutf8

import (
	"strconv"
	"unicode/utf8"

	"github.com/milochristiansen/lua"
)

// Open loads the "utf8" module when executed with "lua.(*State).Call".
//
// It would also be possible to use this with "lua.(*State).Require" (which has some side effects that
// are inappropriate for a core library like this) or "lua.(*State).Preload" (which makes even less
// sense for a core library).
func Open(l *lua.State) int {
	l.NewTable(0, 8) // 6 standard functions
	tidx := l.AbsIndex(-1)

	l.SetTableFunctions(tidx, functions)

	l.Push("charpattern")
	l.Push("[\x00-\x7F\xC2-\xF4][\x80-\xBF]*")
	l.SetTableRaw(tidx)

	l.Push("utf8")
	l.PushIndex(tidx)
	l.SetTableRaw(lua.GlobalsIndex)

	// Sanity check
	if l.AbsIndex(-1) != tidx {
		panic("Oops!")
	}
	return 1
}

var functions = map[string]lua.NativeFunction{
	"char": func(l *lua.State) int {
		n := l.AbsIndex(-1)
		xs := make([]rune, 0, n)
		for i := 1; i <= n; i++ {
			xs = append(xs, rune(l.ToInt(i)))
		}
		l.Push(string(xs))
		return 1
	},
	"codes": func(l *lua.State) int {
		next := liter(l, l.ToString(1), 1, -1)
		l.Push(func(l *lua.State) int {
			if r, n, k := next(); n <= 0 {
				return 0
			} else if r == utf8.RuneError {
				panic("invalid utf8 code at " + strconv.Itoa(k))
				return 0
			} else {
				l.Push(k)
				l.Push(r)
				return 2
			}
		})
		return 1
	},
	"codepoint": func(l *lua.State) int {
		i := int(l.OptInt(2, 1))
		j := int(l.OptInt(3, int64(i)))
		next := liter(l, l.ToString(1), i, j)
		m := 0
		for {
			if r, n, k := next(); n <= 0 {
				break
			} else if r == utf8.RuneError {
				panic("invalid utf8 code at " + strconv.Itoa(k))
			} else {
				l.Push(r)
				m += 1
			}
		}
		return m
	},
	"len": func(l *lua.State) int {
		i := int(l.OptInt(2, 1))
		j := int(l.OptInt(3, -1))
		next := liter(l, l.ToString(1), i, j)
		m := 0
		for {
			if r, n, k := next(); n <= 0 {
				break
			} else if r == utf8.RuneError {
				l.Push(nil)
				l.Push(k)
				return 2
			} else {
				m += 1
			}
		}
		l.Push(m)
		return 1
	},
	"offset": func(l *lua.State) int {
		s := l.ToString(1)
		n := int(l.ToInt(2))
		d := 0
		if n >= 0 {
			d = 1
		} else {
			d = len(s) + 1
		}
		i := int(l.OptInt(3, int64(d)))
		if i < 0 {
			i = len(s) + i + 1
		}

		var next func() (rune, int, int)
		var ok func() bool
		r, m, k := rune(0), 0, i+1
		if n > 0 {
			next = liter(l, s, i, -1)
			ok = func() bool {
				n -= 1
				return n >= 0
			}
		} else if n < 0 {
			next = riter(l, s, 1, i-1)
			ok = func() bool {
				n += 1
				return n <= 0
			}
		} else {
			next = riter(l, s, 1, -1)
			ok = func() bool {
				return k > i
			}
		}
		for ok() {
			if r, m, k = next(); m <= 0 {
				return 0
			} else if r == utf8.RuneError {
				panic("invalid utf8 code at " + strconv.Itoa(k))
			}
		}
		l.Push(k)
		return 1
	},
}

func index(l *lua.State, n, i, j int) (int, int) {
	if i < 0 {
		i = n + i + 1
	}
	if j < 0 {
		j = n + j + 1
	}
	if i <= 0 || j > n {
		panic("index out of range")
	}
	return i, j
}

func liter(l *lua.State, s string, i, j int) func() (rune, int, int) {
	xs := []byte(s)
	i, j = index(l, len(xs), i, j)
	return func() (rune, int, int) {
		if i > j {
			return utf8.RuneError, 0, 0
		}
		r, n := utf8.DecodeRune(xs[i-1:])
		i += n
		return r, n, i - n
	}
}

func riter(l *lua.State, s string, i, j int) func() (rune, int, int) {
	xs := []byte(s)
	i, j = index(l, len(xs), i, j)
	return func() (rune, int, int) {
		if i > j {
			return utf8.RuneError, 0, 0
		}
		r, n := utf8.DecodeLastRune(xs[:j])
		j -= n
		return r, n, j + 1
	}
}
