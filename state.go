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

// DCLua - A light weight Go Lua VM designed for easy embedding.
//
// The compiler generates correct code in every case I have tested, but the code quality is sometimes poor. If
// you want better code quality it is possible to compile scripts with luac and load the binaries...
//
// Currently the compiler does not support constant folding, and some special instructions are not used at all
// (instead preferring simpler sequences of other instructions). For example TESTSET is never generated, TEST
// is used in all cases (largely because It would greatly complicate the compiler if I tried to use TESTSET
// where possible). Expressions use a simple "recursive" code generation style, meaning that it wastes registers
// like crazy in some (rare) cases.
//
// Most (if not all) of the API functions may cause a panic, but only if things go REALLY wrong. If a function
// does not state that it can panic or "raise an error" it will only do so if a critical internal assumption
// proves to be wrong (AKA there is a bug in the code somewhere). These errors will have a special prefix
// prepended onto the error message stating that this error indicates an internal VM bug. If you ever see
// such an error I want to know about it ASAP.
//
// That said, if an API function *can* "raise an error" it can and will panic if something goes wrong. This is
// not a problem inside a native function (as the VM is prepared for this), but if you need to call these functions
// outside of code to be run by the VM you may want to use Protect or Recover to properly catch these errors.
//
// The VM itself does not provide any Lua functions, the standard library is provided entirely by external packages.
// This means that the standard library never does anything that your own code cannot do (there is no "private API"
// that is used by the standard library).
//
// Anything to do with the OS or file IO is not provided. Such things do not belong in the core libraries of an
// embedded scripting language (do you really want scripts to be able to read and write random files without
// restriction?).
package lua

import "fmt"
import "os"
import "io"

const (
	// If you have more than 1000000 items in a single stack frame you probably should think about refactoring...
	RegistryIndex = -1000000 - iota
	GlobalsIndex
	FirstUpVal // To get a specific upvalue use "FirstUpVal-<upvalue index>"
)

// State is the central arbitrator of all Lua operations.
type State struct {
	// Output should be set to whatever writer you want to use for logging.
	// This is where the standard script functions like print will write their output.
	// If nil defaults to os.Stdout.
	Output io.Writer

	// Add a native stack trace to errors that have attached stack traces.
	NativeTrace bool

	registry *table
	global   *table // _G
	metaTbls [typeCount]*table

	stack *stack
}

// NewState creates a new State, ready to use.
func NewState() *State {
	l := &State{
		stack: newStack(),
	}

	l.global = newTable(l, 0, 64)
	l.global.SetRaw("_G", l.global)

	l.registry = newTable(l, 0, 32)
	l.registry.SetRaw("LUA_RIDX_GLOBALS", l.global)

	return l
}

// Output

// Printf writes to the designated output writer (see fmt.Printf).
func (l *State) Printf(format string, msg ...interface{}) {
	if l.Output != nil {
		fmt.Fprintf(l.Output, format, msg...)
		return
	}
	fmt.Fprintf(os.Stdout, format, msg...)
}

// Println writes to the designated output writer (see fmt.Println).
func (l *State) Println(msg ...interface{}) {
	if l.Output != nil {
		fmt.Fprintln(l.Output, msg...)
		return
	}
	fmt.Fprintln(os.Stdout, msg...)
}

// Print writes to the designated output writer (see fmt.Print).
func (l *State) Print(msg ...interface{}) {
	if l.Output != nil {
		fmt.Fprint(l.Output, msg...)
		return
	}
	fmt.Fprint(os.Stdout, msg...)
}

// See api.go for more.
