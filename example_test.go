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

package lua_test

import (
	"fmt"
	"strings"

	"github.com/milochristiansen/lua"
	"github.com/milochristiansen/lua/lmodbase"
	"github.com/milochristiansen/lua/lmodmath"
	"github.com/milochristiansen/lua/lmodpackage"
	"github.com/milochristiansen/lua/lmodstring"
	"github.com/milochristiansen/lua/lmodtable"
)

func Example() {
	l := lua.NewState()

	// This is the easiest way to load a core module.
	// For other modules you probably want to use Preload or Require

	// This sequence is wrapped in a call to Protect to show how that is done, not because you need to.
	// These particular functions *should* be 100% safe to call unprotected. Protect is generally used
	// when you need to do something more complicated and panicking is unacceptable. Code inside native
	// functions does not need to worry about protection, for them it is up to the caller to worry about
	// it. It is very rare to need to explicitly call Recover or Protect.
	err := l.Protect(func() {
		l.Push(lmodbase.Open)
		l.Call(0, 0)

		l.Push(lmodpackage.Open)
		l.Call(0, 0)

		l.Push(lmodstring.Open)
		l.Call(0, 0)

		l.Push(lmodtable.Open)
		l.Call(0, 0)

		l.Push(lmodmath.Open)
		l.Call(0, 0)

		// The following standard modules are not provided for one reason or another:
		//	coroutine: No coroutine support, and if I add support later it will not follow the same rules as standard Lua.
		//	utf8: Haven't gotten around to it yet...
		//	io: IO support is not something you want in an extension language.
		//	os: Even worse than IO, untrusted scripts should not have access to this stuff.
		//	debug: Also not good to expose to untrusted scripts (although there is some stuff here that should be part of the base functions).

		// l.Require("example", loader.Function, false)
		// l.Pop(1)
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	err = l.LoadText(strings.NewReader(`
	print("Hello from github.com/milochristiansen/lua!")
	`), "", 0)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = l.PCall(0, 0)
	if err != nil {
		fmt.Println(err)
	}

	// Output: Hello from github.com/milochristiansen/lua!
}
