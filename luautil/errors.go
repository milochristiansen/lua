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

package luautil

type ErrType int

// Error types.
const (
	ErrTypUndefined     ErrType = iota // Anything that does not fit a category.
	ErrTypMajorInternal                // Errors that should not happen, ever.

	ErrTypGenLexer // Generic syntax errors caught by the lexer.

	ErrTypGenSyntax  // Generic syntax errors.
	ErrTypGenRuntime // Generic run time errors.

	ErrTypBinLoader // An error encountered while loading a binary chunk.
	ErrTypBinDumper

	ErrTypWrapped // An error from some other library or native API code wrapped into a standard Error.
	ErrTypEvil    // If some idiot panics with a non-error value, it will be wrapped with this type.
)

// Error is used for any and every error that is produced by the VM and its peripherals.
type Error struct {
	Err  error
	Msg  string
	Type ErrType

	Trace string
}

// Error formats an Error like so:
//	<Msg>: <Err.Error()>
//	  Stack Trace: <Trace>
// If any of the three parts are missing they are elided, in the extreme case of an empty error the message will be:
//	Unspecified error
func (err Error) Error() string {
	at := ""
	if err.Trace != "" {
		at = "\n  Stack Trace:" + err.Trace
	}

	msg := "Unspecified error"
	if err.Msg != "" {
		msg = err.Msg
	}
	if err.Type == ErrTypMajorInternal {
		msg = "Major internal error, this indicates an internal VM bug! " + msg
	}

	errmsg := ""
	if err.Err != nil {
		errmsg = ": " + err.Err.Error()
	}

	return msg + errmsg + at
}

// Raise converts a string to a Error and then panics with it.
func Raise(msg string, typ ErrType) {
	panic(Error{Msg: msg, Type: typ})
}

// RaiseExisting converts an error to a Error then panics with it.
func RaiseExisting(err error, msg string) {
	panic(Error{Msg: msg, Type: ErrTypWrapped, Err: err})
}
