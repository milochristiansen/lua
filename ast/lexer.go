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

package ast

import "strings"
import "unicode"
import "github.com/milochristiansen/lua/luautil"

const (
	tknINVALID = iota - 1 // Invalid

	// Keywords
	tknAnd
	tknOr
	tknNot
	tknWhile
	tknFor
	tknRepeat
	tknUntil
	tknIn
	tknDo
	tknBreak
	tknEnd
	tknIf
	tknThen
	tknElse
	tknElseif
	tknFunction
	tknGoto
	tknLocal
	tknReturn
	tknTrue
	tknFalse
	tknNil
	tknContinue // Not used by the default parser.

	// Operators
	tknAdd         // +
	tknSub         // - (also unary minus)
	tknMul         // *
	tknDiv         // /
	tknIDiv        // //
	tknMod         // %
	tknPow         // ^
	tknLen         // #
	tknSet         // =
	tknEQ          // ==
	tknGT          // >
	tknGE          // >=
	tknLT          // <
	tknLE          // <=
	tknNE          // ~=
	tknShiftL      // <<
	tknShiftR      // >>
	tknBXOr        // ~ (also unary bitwise not)
	tknBOr         // |
	tknBAnd        // &
	tknColon       // :
	tknDblColon    // ::
	tknDot         // .
	tknConcat      // ..
	tknVariadic    // ...
	tknSeperator   // ,
	tknUnnecessary // ;) Syntactic sugar causes cancer of the semicolon
	tknOIndex      // [
	tknCIndex      // ]
	tknOBracket    // {
	tknCBracket    // }
	tknOParen      // (
	tknCParen      // )

	// Values
	tknInt
	tknFloat
	tknName
	tknString
)

var keywords = map[string]int{
	"and":    tknAnd,
	"or":     tknOr,
	"not":    tknNot,
	"while":  tknWhile,
	"for":    tknFor,
	"repeat": tknRepeat,
	"until":  tknUntil,
	"in":     tknIn,
	"do":     tknDo,
	"break":  tknBreak,
	//"continue": tknContinue, // Uncomment to enable the "continue" keyword.
	"end":      tknEnd,
	"if":       tknIf,
	"then":     tknThen,
	"else":     tknElse,
	"elseif":   tknElseif,
	"function": tknFunction,
	"goto":     tknGoto,
	"local":    tknLocal,
	"return":   tknReturn,
	"true":     tknTrue,
	"false":    tknFalse,
	"nil":      tknNil,
}

func keyword(s string) int {
	if t, ok := keywords[s]; ok {
		return t
	}
	return tknName
}

type lexer struct {
	exlook  *token
	look    *token
	current *token

	source *strings.Reader
	line   int
	char   rune
	nline  int // Keep some lookahead information around.
	nchar  rune
	eof    bool // true if there are no more chars to read
	neof   bool // true if nchar is invalid (next call to next will trigger EOF)

	lexeme []rune

	token     int
	tokenline int

	strdepth int
	objdepth int
}

// Returns a new Lua lexer.
func newLexer(source string, line int) *lexer {
	lex := new(lexer)

	lex.source = strings.NewReader(source)

	lex.line = line
	lex.nline = line

	lex.lexeme = make([]rune, 0, 20)

	lex.token = tknINVALID
	lex.tokenline = line

	lex.strdepth = 0
	lex.objdepth = 0

	// prime the pump
	lex.nextchar()
	lex.nextchar()
	lex.exlook = &token{"INVALID", tknINVALID, lex.tokenline}
	lex.look = &token{"INVALID", tknINVALID, lex.tokenline}
	lex.advance()
	lex.advance()

	return lex
}

// advance retrieves the next token from the stream.
// For most purposes use getcurrent instead.
func (lex *lexer) advance() {
	lex.current, lex.look = lex.look, lex.exlook
	if lex.eof {
		lex.exlook = &token{"EOF", tknINVALID, lex.tokenline}
		return
	}

	lex.eatWS()
	if lex.eof {
		lex.exlook = &token{"EOF", tknINVALID, lex.tokenline}
		return
	}

	// We are at the beginning of a token
	lex.tokenline = lex.line
	switch lex.char {
	case ';':
		lex.makeToken(tknUnnecessary)
	case '+':
		lex.makeToken(tknAdd)
	case '-':
		lex.makeToken(tknSub) // eatWS already eliminated any comments
	case '*':
		lex.makeToken(tknMul)
	case '/':
		if lex.nmatch("/") {
			lex.nextchar()
			lex.makeToken(tknIDiv)
			break
		}
		lex.makeToken(tknDiv)
	case '%':
		lex.makeToken(tknMod)
	case '^':
		lex.makeToken(tknPow)
	case '#':
		lex.makeToken(tknLen)
	case '=':
		if lex.nmatch("=") {
			lex.nextchar()
			lex.makeToken(tknEQ)
			break
		}
		lex.makeToken(tknSet)
	case '>':
		if lex.nmatch(">") {
			lex.nextchar()
			lex.makeToken(tknShiftR)
			break
		}
		if lex.nmatch("=") {
			lex.nextchar()
			lex.makeToken(tknGE)
			break
		}
		lex.makeToken(tknGT)
	case '<':
		if lex.nmatch("<") {
			lex.nextchar()
			lex.makeToken(tknShiftL)
			break
		}
		if lex.nmatch("=") {
			lex.nextchar()
			lex.makeToken(tknLE)
			break
		}
		lex.makeToken(tknLT)
	case '~':
		if lex.nmatch("=") {
			lex.nextchar()
			lex.makeToken(tknNE)
			break
		}
		lex.makeToken(tknBXOr)
	case '|':
		lex.makeToken(tknBOr)
	case '&':
		lex.makeToken(tknBAnd)
	case ':':
		if lex.nmatch(":") {
			lex.nextchar()
			lex.makeToken(tknDblColon)
			break
		}
		lex.makeToken(tknColon)
	case '.':
		if lex.nmatch(".") {
			lex.nextchar()
			if lex.nmatch(".") {
				lex.nextchar()
				lex.makeToken(tknVariadic)
				break
			}
			lex.makeToken(tknConcat)
			break
		}
		lex.makeToken(tknDot)
	case ',':
		lex.makeToken(tknSeperator)
	case '[':
		if !lex.nmatch("[=") {
			lex.makeToken(tknOIndex)
			break
		}
		lex.matchRawString()
	case ']':
		lex.makeToken(tknCIndex)
	case '{':
		lex.makeToken(tknOBracket)
	case '}':
		lex.makeToken(tknCBracket)
	case '(':
		lex.makeToken(tknOParen)
	case ')':
		lex.makeToken(tknCParen)
	case '\'':
		lex.matchString('\'')
	case '"':
		lex.matchString('"')
	default:
		if lex.matchAlpha() {
			// Identifier or keyword
			for !lex.eof && (lex.matchAlpha() || lex.matchNumeric()) {
				lex.addLexeme()
				lex.nextchar()
			}

			ident := string(lex.lexeme)
			lex.exlook = &token{ident, keyword(ident), lex.tokenline}
		} else if lex.matchNumeric() {
			lex.matchNumber()
		} else {
			luautil.Raise("Illegal character '"+string([]rune{lex.char})+"' while lexing source", luautil.ErrTypGenLexer)
		}
	}

	lex.lexeme = lex.lexeme[0:0]
}

// getCurrent gets the next token, and panics with an error if it's not of type tokenType.
// May cause a panic if the lexer encounters an error.
// Used as a type checked advance.
func (lex *lexer) getCurrent(tokenTypes ...int) {
	lex.advance()

	for _, val := range tokenTypes {
		if lex.current.Type == val {
			return
		}
	}

	exitOnTokenExpected(lex.current, tokenTypes...)
}

// checkLook checks to see if the look ahead is one of tokenTypes and if so returns true.
func (lex *lexer) checkLook(tokenTypes ...int) bool {
	for _, val := range tokenTypes {
		if lex.look.Type == val {
			return true
		}
	}
	return false
}

// return true if the current char matches one of the chars in the string.
func (lex *lexer) match(chars string) bool {
	if lex.eof {
		return false
	}

	for _, char := range chars {
		if lex.char == char {
			return true
		}
	}
	return false
}

// return true if the next char matches one of the chars in the string.
func (lex *lexer) nmatch(chars string) bool {
	if lex.neof {
		return false
	}

	for _, char := range chars {
		if lex.nchar == char {
			return true
		}
	}
	return false
}

func (lex *lexer) matchAlpha() bool {
	if lex.eof {
		return false
	}

	// The way standard Lua does it:
	//return lex.char >= `a` && lex.char <= `z` || lex.char >= `A` && lex.char <= `Z` || lex.char == '_'
	// But why waste my unicode lexer? If you want to give your variables names in chinese you should be able to.
	return lex.char == '_' || unicode.IsLetter(lex.char)
}

func (lex *lexer) matchNumeric() bool {
	if lex.eof {
		return false
	}
	// It would be WAY too complicated to support non-arabic numerals.
	return lex.char >= '0' && lex.char <= '9'
}

// Fetch the next char (actually a Unicode code point).
// I don't like the way EOF is handled, but there is really no better way that is flexible enough.
func (lex *lexer) nextchar() {
	if lex.eof {
		return
	}
	if lex.neof {
		lex.eof = true
		return
	}

	var err error
	prevNL := '\000'

	lex.char = lex.nchar
	lex.line = lex.nline

	// Read the next char. This does a lot of special stuff to handle all possible types
	// of line endings (as required by the stupid Lua spec). The only place special handling
	// of newlines is actually required is in strings, where it is defined that "\r", "\n",
	// "\r\n", and "\n\r" should all collapse to "\n".
again:
	lex.nchar, _, err = lex.source.ReadRune() // err should only ever be io.EOF
	if err != nil {
		if prevNL == '\n' || prevNL == '\r' {
			lex.nchar = '\n'
			lex.nline++
			return
		}
		lex.neof = true
		return
	}

	// Shortcut all the following nonsense for the common case
	if lex.nchar != '\n' && lex.nchar != '\r' && prevNL == '\000' {
		return
	}

	// If the last char we read before this one was a newline and this char is a different
	// kind of new line than that one, then collapse the two to one.
	if (prevNL == '\n' && lex.nchar == '\r') || (prevNL == '\r' && lex.nchar == '\n') {
		prevNL = '\000'
		lex.nchar = '\n'
		lex.nline++
		return
	}

	// If we find a newline then try to find it's companion (if it has one).
	if (lex.nchar == '\n' || lex.nchar == '\r') && prevNL == '\000' {
		prevNL = lex.nchar
		goto again
	}

	// If we found a newline before but the next char was not a newline then unread the next char and go on.
	if prevNL == '\n' || prevNL == '\r' {
		lex.nchar = '\n'
		lex.nline++
		lex.source.UnreadRune()
		return
	}

	panic("UNREACHABLE?")
}

// Add the current char to the lexeme buffer.
func (lex *lexer) addLexeme() {
	lex.lexeme = append(lex.lexeme, lex.char)
}

// Add the current char to the lexeme buffer.
func (lex *lexer) makeToken(tkn int) {
	lex.exlook = &token{"", tkn, lex.tokenline}
	lex.nextchar()
}

// Eat white space and comments.
func (lex *lexer) eatWS() {
	for {
		if lex.match("-") && lex.nmatch("-") {
			lex.nextchar()
			lex.nextchar()
			if lex.eof {
				return
			}

			// Is long comment?
			if lex.match("[") && lex.nmatch("[=") {
				i := 0
				lex.nextchar()
				if lex.eof {
					return
				}
				for lex.match("=") {
					i++
					lex.nextchar()
					if lex.eof {
						return
					}
				}
				lex.nextchar()
				if lex.eof {
					return
				}

			nextcchar:
				for {
					if lex.match("]") && lex.nmatch("=]") {
						// Make sure the closing long bracket is the same level as the opener
						lex.nextchar()
						if lex.eof {
							return
						}

						if i > 0 {
							for k := 0; k < i; k++ {
								if !lex.match("=") {
									continue nextcchar
								}
								lex.nextchar()
								if lex.eof {
									return
								}
							}
						}

						if !lex.match("]") {
							continue
						}
						lex.nextchar()
						if lex.eof {
							return
						}
						break
					}
					lex.nextchar()
					if lex.eof {
						return
					}
				}
				continue
			}

			for {
				if lex.match("\n") {
					lex.nextchar()
					if lex.eof {
						return
					}
					break
				}
				lex.nextchar()
				if lex.eof {
					return
				}
			}
		}
		if lex.match("\n\r \t") {
			lex.nextchar()
			if lex.eof {
				return
			}
			continue
		}
		if lex.match("-") && lex.nmatch("-") {
			continue
		}
		break
	}
}

func (lex *lexer) matchNumber() {
	expo := "Ee"
	hex := false
	if lex.match("0") && lex.nmatch("Xx") {
		expo = "Pp"
		hex = true
		lex.addLexeme()
		lex.nextchar()
		lex.addLexeme()
		lex.nextchar()

		// We need at least one digit.
		if lex.eof || !(lex.matchNumeric() || lex.match(".") || lex.match("abcdefABCDEF")) {
			luautil.Raise("Unexpected end of hexadecimal numeric literal", luautil.ErrTypGenLexer)
		}
	}

	for !lex.eof {
		if lex.match(".") {
			lex.addLexeme()
			lex.nextchar()
			continue
		}
		if lex.match(expo) {
			lex.addLexeme()
			lex.nextchar()
			if lex.match("+-") {
				lex.addLexeme()
				lex.nextchar()
			}
			continue
		}
		if lex.matchNumeric() || hex && lex.match("abcdefABCDEF") {
			lex.addLexeme()
			lex.nextchar()
			continue
		}

		break
	}

	n := string(lex.lexeme)
	valid, iok, _, _ := luautil.ConvNumber(n, true, true)
	if !valid {
		luautil.Raise("Invalid numeric literal", luautil.ErrTypGenLexer)
	}
	if iok {
		lex.exlook = &token{n, tknInt, lex.tokenline}
		return
	}
	lex.exlook = &token{n, tknFloat, lex.tokenline}
}

func hexval(r rune) rune {
	if r >= 'a' && r <= 'f' {
		return r - 'a' + 10
	} else if r >= 'A' && r <= 'F' {
		return r - 'A' + 10
	} else if r >= '0' && r <= '9' {
		return r - '0'
	}
	luautil.Raise("Invalid hexadecimal digit in escape", luautil.ErrTypGenLexer)
	panic("UNREACHABLE")
}

func (lex *lexer) matchString(delim rune) {
	lex.nextchar()
	if lex.eof {
		luautil.Raise("Unexpected EOF while reading a string", luautil.ErrTypGenLexer)
	}
	if lex.char == delim {
		lex.exlook = &token{"", tknString, lex.tokenline}
		lex.nextchar()
		return
	}

	for lex.char != delim {
		if lex.eof {
			luautil.Raise("Unexpected EOF while reading a string", luautil.ErrTypGenLexer)
		}

		// Handle escapes
		if lex.char == '\\' {
			lex.nextchar()
			if lex.eof {
				luautil.Raise("Unexpected EOF while reading a string", luautil.ErrTypGenLexer)
			}

			switch lex.char {
			case '\n':
				fallthrough
			case 'n':
				lex.lexeme = append(lex.lexeme, '\n')
			case 'r':
				lex.lexeme = append(lex.lexeme, '\r')
			case 't':
				lex.lexeme = append(lex.lexeme, '\t')
			case 'v':
				lex.lexeme = append(lex.lexeme, '\v')
			case 'a':
				lex.lexeme = append(lex.lexeme, '\a')
			case 'b':
				lex.lexeme = append(lex.lexeme, '\b')
			case 'f':
				lex.lexeme = append(lex.lexeme, '\f')
			case '"':
				lex.lexeme = append(lex.lexeme, '"')
			case '\'':
				lex.lexeme = append(lex.lexeme, '\'')
			case '\\':
				lex.lexeme = append(lex.lexeme, '\\')
			case '0':
				lex.lexeme = append(lex.lexeme, '\000')
			case 'z':
				for lex.match("\n\r \t") {
					lex.nextchar()
					if lex.eof {
						luautil.Raise("Unexpected EOF while reading a string", luautil.ErrTypGenLexer)
					}
				}
			case 'x':
				r := '\000'
				lex.nextchar()
				r = hexval(lex.char) << 4
				lex.nextchar()
				r = r + hexval(lex.char)
				if lex.eof {
					luautil.Raise("Unexpected EOF while reading a string", luautil.ErrTypGenLexer)
				}
				lex.lexeme = append(lex.lexeme, r)
			case 'u':
				lex.nextchar()
				if lex.eof {
					luautil.Raise("Unexpected EOF while reading a string", luautil.ErrTypGenLexer)
				}
				if lex.char != '{' {
					luautil.Raise("Missing open bracket in unicode escape", luautil.ErrTypGenLexer)
				}

				r := '\000'
				for i := 0; ; i++ {
					lex.nextchar()
					if lex.eof {
						luautil.Raise("Unexpected EOF while reading a string", luautil.ErrTypGenLexer)
					}
					if lex.char == '}' {
						break
					}

					r = (r << 4) + hexval(lex.char)
				}
				if r > 0x10FFFF {
					luautil.Raise("Unicode escape value is too large", luautil.ErrTypGenLexer)
				}
				lex.lexeme = append(lex.lexeme, r)
			default:
				if lex.matchNumeric() {
					r := '\000'
					for i := 0; i < 3 && lex.matchNumeric(); i++ {
						r = 10*r + lex.char - '0'

						lex.nextchar()
						if lex.eof {
							luautil.Raise("Unexpected EOF while reading a string", luautil.ErrTypGenLexer)
						}
					}
					if r > 0xFF {
						luautil.Raise("Decimal escape value is too large", luautil.ErrTypGenLexer)
					}
					lex.lexeme = append(lex.lexeme, r)
				}
				luautil.Raise("Invalid escape sequence while reading a string", luautil.ErrTypGenLexer)
			}

			lex.nextchar()
			continue
		}

		lex.addLexeme()
		lex.nextchar()
	}
	lex.nextchar()
	lex.exlook = &token{string(lex.lexeme), tknString, lex.tokenline}
	return
}

func (lex *lexer) matchRawString() {
	i := 0
	lex.nextchar()
	if lex.eof {
		luautil.Raise("Unexpected EOF while reading a string", luautil.ErrTypGenLexer)
	}
	for lex.match("=") {
		i++
		lex.nextchar()
		if lex.eof {
			luautil.Raise("Unexpected EOF while reading a string", luautil.ErrTypGenLexer)
		}
	}
	lex.nextchar()
	if lex.eof {
		luautil.Raise("Unexpected EOF while reading a string", luautil.ErrTypGenLexer)
	}

next:
	for {
		if lex.eof {
			luautil.Raise("Unexpected EOF while reading a string", luautil.ErrTypGenLexer)
		}

		if lex.match("]") && lex.nmatch("=]") {
			// Make sure the closing long bracket is the same level as the opener
			lex.nextchar()
			if lex.eof {
				luautil.Raise("Unexpected EOF while reading a string", luautil.ErrTypGenLexer)
			}

			k := 0
			buff := []rune{']'}
			if i > 0 {
				for ; k < i; k++ {
					if !lex.match("=") {
						lex.lexeme = append(lex.lexeme, buff...)
						continue next
					}
					buff = append(buff, lex.char)
					lex.nextchar()
					if lex.eof {
						luautil.Raise("Unexpected EOF while reading a string", luautil.ErrTypGenLexer)
					}
				}
			}

			if !lex.match("]") {
				lex.lexeme = append(lex.lexeme, buff...)
				continue
			}
			lex.nextchar()
			break
		}
		lex.addLexeme()
		lex.nextchar()
	}
	lex.exlook = &token{string(lex.lexeme), tknString, lex.tokenline}
}

// Token

type token struct {
	Lexeme string
	Type   int
	Line   int
}

func (t *token) String() string {
	return tokenTypeToString(t.Type)
}

func tokenTypeToString(typ int) string {
	if typ < 0 || typ > tknString {
		return "<INVALID|EOS>"
	}

	return [...]string{
		// Keywords
		"and",
		"or",
		"not",
		"while",
		"for",
		"repeat",
		"until",
		"in",
		"do",
		"break",
		"end",
		"if",
		"then",
		"else",
		"elseif",
		"function",
		"goto",
		"local",
		"return",
		"true",
		"false",
		"nil",
		"continue",

		// Operators
		"+",
		"-",
		"*",
		"/",
		"//",
		"%",
		"^",
		"#",
		"=",
		"==",
		">",
		">=",
		"<",
		"<=",
		"~=",
		"<<",
		">>",
		"~",
		"|",
		"&",
		":",
		"::",
		".",
		"..",
		"...",
		",",
		";",
		"[",
		"]",
		"{",
		"}",
		"(",
		")",

		// Values
		"<integer>",
		"<float>",
		"<identifier>",
		"<string>",
	}[typ]
}

// Panics with a message formatted like one of the following:
//	Invalid token: Found: thecurrenttoken. Expected: expected1, expected2, or expected3.
//	Invalid token: Found: thecurrenttoken. Expected: expected1 or expected2.
//	Invalid token: Found: thecurrenttoken. Expected: expected.
//	Invalid token: Found: thecurrenttoken (Lexeme: test). Expected: expected1, expected2, or expected3.
//	Invalid token: Found: thecurrenttoken (Lexeme: test). Expected: expected1 or expected2.
//	Invalid token: Found: thecurrenttoken (Lexeme: test). Expected: expected.
// If the lexeme is long (>20 chars) it is truncated.
func exitOnTokenExpected(token *token, expected ...int) {
	expectedString := ""
	expectedCount := len(expected) - 1
	for i, val := range expected {
		// Is the only value
		if expectedCount == 0 {
			expectedString = tokenTypeToString(val)
			continue
		}

		// Is last of a list (2 or more)
		if i == expectedCount && expectedCount > 0 {
			expectedString += "or " + tokenTypeToString(val)
			continue
		}

		// Is the first of two
		if expectedCount == 1 {
			expectedString += tokenTypeToString(val) + " "
			continue
		}

		// Is any but the last of a list of 3 or more
		expectedString += tokenTypeToString(val) + ", "
	}

	found := token.String()
	if token.Lexeme != "" {
		if len(token.Lexeme) <= 20 {
			found += " (Lexeme: " + token.Lexeme + ")"
		} else {
			found += " (Lexeme: " + token.Lexeme[:17] + "...)"
		}
	}
	luautil.Raise("Invalid token: Found: "+found+" Expected: "+expectedString, luautil.ErrTypGenSyntax)
}
