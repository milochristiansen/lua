// +build ignore

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

package lua

import (
	"strings"

	"testing"

	glua "github.com/yuin/gopher-lua"
)

var source = `
function fib(n)
	if n == 0 then
		return 0
	elseif n == 1 then
		return 1
	end
	
	local n0, n1 = 0, 1
	
	for i = n, 2, -1 do
		local tmp = n0 + n1
		n0 = n1
		n1 = tmp
	end
	
	return n1
end
fib(50)
`

// A pair of almost identical benchmarks for comparing performance between this package and gopher-lua.
// Run with:
//	go test -bench=. -run=none github.com/milochristiansen/lua
// Don't forget to remove the build constraint first!

func BenchmarkA(b *testing.B) {
	l := NewState()

	err := l.LoadText(strings.NewReader(source), "fibtest.go", 0)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 1; i < b.N; i++ {
		l.PushIndex(-1)
		err := l.PCall(0, 0)
		if err != nil {
			b.Fatal(err)
			return
		}
	}
}

func BenchmarkB(b *testing.B) {
	l := glua.NewState()
	exe, err := l.Load(strings.NewReader(source), "fibtest.go")
	if err != nil {
		b.Fatal(err)
		return
	}

	b.ResetTimer()
	for i := 1; i < b.N; i++ {
		l.Push(exe)
		err := l.PCall(0, 0, nil)
		if err != nil {
			b.Fatal(err)
			return
		}
	}
}
