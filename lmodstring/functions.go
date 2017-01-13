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

package lmodstring

import "github.com/milochristiansen/lua"

import "strings"
import "strconv"
import "fmt"

// Open loads the "string" module when executed with "lua.(*State).Call".
//
// It would also be possible to use this with "lua.(*State).Require" (which has some side effects that
// are inappropriate for a core library like this) or "lua.(*State).Preload" (which makes even less
// sense for a core library).
//
// The following standard Lua functions/fields are not provided:
//	string.gmatch
//	string.gsub
//	string.match
//	string.pack
//	string.packsize
//	string.unpack
//
// Additionally "string.find" does not allow pattern based searching.
//
// The following non-standard functions are provided:
//	string.count
//	string.hasprefix
//	string.hassuffix
//	string.join (like table.concat, but not exactly)
//	string.replace
//	string.split
//	string.splitafter
//	string.title
//	string.trim
//	string.trimprefix
//	string.trimspace
//	string.trimsuffix
//	string.unquote
// For more information about these extensions (including how to disable them) see the "README.md" file
// for this package (not the main "lua" package!).
func Open(l *lua.State) int {
	l.NewTable(0, 32) // 11 standard functions (+ 6 DNI) + 13 nonstandard
	tidx := l.AbsIndex(-1)

	l.SetTableFunctions(tidx, functions)

	l.Push("_NO_STRING_EXTS")
	l.GetTableRaw(lua.RegistryIndex)
	ok := l.ToBool(-1)
	l.Pop(1)
	if !ok {
		l.SetTableFunctions(tidx, extFunctions)
	}

	l.Push("string")
	l.PushIndex(tidx)
	l.SetTableRaw(lua.GlobalsIndex)

	l.Push("")
	l.NewTable(0, 1)
	l.Push("__index")
	l.PushIndex(tidx)
	l.SetTableRaw(-3)
	l.SetMetaTable(-2)
	l.Pop(1)

	// Sanity check
	if l.AbsIndex(-1) != tidx {
		panic("ItNoWork!")
	}
	return 1
}

var functions = map[string]lua.NativeFunction{
	"byte": func(l *lua.State) int {
		str := l.OptString(1, "")
		i := l.OptInt(2, 1)
		j := l.OptInt(3, i)

		if i < 0 {
			i = int64(len(str)) + (i + 1)
		}
		if j < 0 {
			j = int64(len(str)) + (j + 1)
		}

		if i < 1 {
			i = 1
		}
		if j > int64(len(str)) {
			j = int64(len(str))
		}
		if i > j {
			return 0
		}

		n := j - i + 1
		for k := int64(0); k < n; k++ {
			l.Push(int64(str[k+j-1]))
		}
		return int(n)
	},
	"char": func(l *lua.State) int {
		n := l.AbsIndex(-1)
		b := make([]byte, 0, n)
		for i := 1; i <= n; i++ {
			b = append(b, byte(l.ToInt(i)))
		}
		l.Push(string(b))
		return 1
	},
	"dump": func(l *lua.State) int {
		l.Push(l.DumpFunction(1, l.ToBool(2)))
		return 1
	},
	"find": func(l *lua.State) int { // No pattern matching
		str := l.OptString(1, "")
		sub := l.OptString(2, "")
		i := l.OptInt(3, 1)

		if i < 0 {
			i = int64(len(str)) + (i + 1)
		}
		if i < 1 {
			i = 1
		}
		i--

		idx := strings.Index(str[i:], sub)
		if idx == -1 {
			return 0
		}
		l.Push(int64(idx) + i + 1)
		l.Push(int64(idx) + i + int64(len(sub)))
		return 2
	},
	"format": func(l *lua.State) int { // Uses the same format codes as Go's fmt functions.
		n := l.AbsIndex(-1)
		args := make([]interface{}, 0, n)
		for i := 2; i <= n; i++ {
			args = append(args, l.GetRaw(i))
		}

		l.Push(fmt.Sprintf(l.OptString(1, ""), args...))
		return 1
	},
	// gmatch
	// gsub
	"len": func(l *lua.State) int {
		l.Push(int64(l.Length(1)))
		return 1
	},
	"lower": func(l *lua.State) int {
		str := l.OptString(1, "")
		l.Push(strings.ToLower(str))
		return 1
	},
	// match
	// pack
	// packsize
	"rep": func(l *lua.State) int {
		str := l.OptString(1, "")
		c := l.OptInt(2, 1)
		sep := l.OptString(3, "")

		b := make([]byte, 0, (int64(len(str))+int64(len(sep)))*c)

		for i := int64(0); i < c; i++ {
			b = append(b, str...)
			if i+1 < c {
				b = append(b, sep...)
			}
		}

		l.Push(string(b))
		return 1
	},
	"reverse": func(l *lua.State) int {
		str := l.OptString(1, "")

		b := make([]byte, 0, len(str))
		for i := len(str) - 1; i >= 0; i-- {
			b = append(b, str[i])
		}
		l.Push(string(b))
		return 1
	},
	"sub": func(l *lua.State) int {
		str := l.OptString(1, "")
		i := l.OptInt(2, 1)
		j := l.OptInt(3, -1)

		if i < 0 {
			i = int64(len(str)) + (i + 1)
		}
		if j < 0 {
			j = int64(len(str)) + (j + 1)
		}

		if i < 1 {
			i = 1
		}
		if j > int64(len(str)) {
			j = int64(len(str))
		}
		if i > j {
			return 0
		}

		i--
		j--

		l.Push(str[i : j+1])
		return 1
	},
	// unpack
	"upper": func(l *lua.State) int {
		str := l.OptString(1, "")
		l.Push(strings.ToUpper(str))
		return 1
	},
}

var extFunctions = map[string]lua.NativeFunction{
	"count": func(l *lua.State) int {
		str := l.OptString(1, "")
		sep := l.OptString(2, "")
		l.Push(int64(strings.Count(str, sep)))
		return 1
	},

	"hasprefix": func(l *lua.State) int {
		str := l.OptString(1, "")
		a := l.OptString(2, "")
		l.Push(strings.HasPrefix(str, a))
		return 1
	},

	"hassuffix": func(l *lua.State) int {
		str := l.OptString(1, "")
		a := l.OptString(2, "")
		l.Push(strings.HasSuffix(str, a))
		return 1
	},

	"join": func(l *lua.State) int {
		ln := l.Length(1)

		set := make([]string, 0, ln)
		for i := 1; i <= ln; i++ {
			l.Push(int64(i))
			l.GetTable(1)
			set = append(set, l.ToString(-1))
			l.Pop(1)
		}

		sep := l.OptString(2, ", ")

		l.Push(strings.Join(set, sep))
		return 1
	},

	"replace": func(l *lua.State) int {
		str := l.OptString(1, "")
		o := l.OptString(2, "")
		n := l.OptString(3, "")
		i := int(l.OptInt(4, -1))
		l.Push(strings.Replace(str, o, n, i))
		return 1
	},

	"split": func(l *lua.State) int {
		str := l.OptString(1, "")
		sep := l.OptString(2, "")
		n := int(l.OptInt(3, -1))

		result := strings.SplitN(str, sep, n)
		l.NewTable(len(result), 0)
		for i := range result {
			l.Push(int64(i + 1))
			l.Push(result[i])
			l.SetTableRaw(-3)
		}
		return 1
	},

	"splitafter": func(l *lua.State) int {
		str := l.OptString(1, "")
		sep := l.OptString(2, "")
		n := int(l.OptInt(3, -1))

		result := strings.SplitAfterN(str, sep, n)
		l.NewTable(len(result), 0)
		for i := range result {
			l.Push(int64(i + 1))
			l.Push(result[i])
			l.SetTableRaw(-3)
		}
		return 1
	},

	"title": func(l *lua.State) int {
		str := l.OptString(1, "")
		l.Push(strings.Title(str))
		return 1
	},

	"trim": func(l *lua.State) int {
		str := l.OptString(1, "")
		l.Push(strings.TrimSpace(str))
		return 1
	},

	"trimprefix": func(l *lua.State) int {
		str := l.OptString(1, "")
		a := l.OptString(2, "")
		l.Push(strings.TrimPrefix(str, a))
		return 1
	},

	"trimspace": func(l *lua.State) int {
		str := l.OptString(1, "")
		l.Push(strings.TrimSpace(str))
		return 1
	},

	"trimsuffix": func(l *lua.State) int {
		str := l.OptString(1, "")
		a := l.OptString(2, "")
		l.Push(strings.TrimSuffix(str, a))
		return 1
	},

	"unquote": func(l *lua.State) int {
		str := l.OptString(1, "")

		rtn, err := strconv.Unquote(str)
		if err != nil {
			l.Push(str)
			return 1
		}
		l.Push(rtn)
		return 1
	},
}
