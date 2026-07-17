// Regression tests for the ast lexer.
package ast

import "testing"

// TestDecimalEscapeDoesNotOvershoot exercises the `\ddd` string escape
// handling. The previous implementation advanced one character past the last
// digit inside its inner loop, and then the outer escape-block trailer
// advanced again — silently consuming the character that followed the
// escape. When that character happened to be the closing quote (as in
// `"\128" .. "x"` or the protobuf byte-blob strings produced by many pb Lua
// loaders), the lexer would run off past the end of the string and later
// report either
//
//	"Unexpected EOF while reading a string"
//
// or
//
//	"Illegal character '\\' while lexing source"
//
// depending on what followed in the source. All of the snippets below are
// valid Lua and should parse without error.
func TestDecimalEscapeDoesNotOvershoot(t *testing.T) {
	cases := []string{
		`local s = "\128"`,
		`local s = "\9\10\18"`,
		`local s = "\128" .. "x"`,
		`local s = "abc\128" .. "y"`,
		`local s = "\9" .. "y"`,
		`local x = "\128"; local y = "z"`,
		// A protobuf-descriptor-like byte string (harmless bytes only).
		`local s = "a\24\2 \1(\9" .. "b\9"`,
	}
	for _, src := range cases {
		if _, err := Parse(src, 1); err != nil {
			t.Errorf("Parse(%q) unexpected error: %v", src, err)
		}
	}
}
