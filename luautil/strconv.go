/*
Copyright 2015-2016 by Milo Christiansen

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

import "strings"
import "strconv"
import "math"

// ConvNumber converts a string to a number.
// This is intended for internal use by various Lua related packages, you should not use this unless you know what you are doing.
func ConvNumber(s string, integer, float bool) (valid, iok bool, i int64, f float64) {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return false, false, 0, 0.0
	}
	
	if integer {
		if i, ok := convInt(s); ok {
			return true, true, i, 0.0
		}
	}
	if float {
		if f, ok := convFloat(s); ok {
			return true, false, 0, f
		}
	}
	return false, false, 0, 0.0
}

func cton(r byte) byte {
	if r >= 'a' && r <= 'f' {
		return r - 'a' + 10
	} else if r >= 'A' && r <= 'F' {
		return r - 'A' + 10
	} else if r >= '0' && r <= '9' {
		return r - '0'
	}
	panic("IMPOSSIBLE!")
}

func convInt(s string) (int64, bool) {
	a := int64(0)
	i := 0
	empty := true
	
	neg := false
	if len(s) >= 1 && s[0] == '-' {
		neg = true
		i += 1
	}
	
	hex := false
	if len(s) >= i + 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		i += 2
		hex = true
	}
	
	for ; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' || hex && (s[i] >= 'a' && s[i] <= 'f' || s[i] >= 'A' && s[i] <= 'F') {
			if hex {
				a = a * 16 + int64(cton(s[i]))
			} else {
				a = a * 10 + int64(cton(s[i]))
			}
			empty = false
			continue
		}
		return 0, false // Invalid character
	}
	
	if empty {
		return 0, false
	}
	if neg {
		return -a, true
	}
	return a, true
}

// No hexidecamal float support, but who uses hexadecimal floats anyway?
func convFloat(s string) (float64, bool) {
	if len(s) > 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		Raise("Sadly hexadecimal floating point literals are currently not supported, use decimal literals", ErrTypGenLexer)
	}
	
	f, err := strconv.ParseFloat(s, 64)
	return f, err == nil
}

// My attempt a a float converter, basically implemented from the hexadecimal converter in "lobject.c".
// The problem with this is that it doesn't work and I don't know why or how to fix it, I suck at this kind of math...
func convFloatBroken(s string) (float64, bool) {
	a := 0.0
	e := 0
	i := 0
	empty := true
	dot := false
	
	neg := false
	if len(s) >= 1 && s[0] == '-' {
		neg = true
		i += 1
	}
	
	hex := false
	if len(s) > i + 2 && s[i+1] == '0' && (s[i+2] == 'x' || s[i+2] == 'X') {
		i += 2
		hex = true
	}
	
	for ; i < len(s); i++ {
		if s[i] == '.' {
			if dot {
				return 0, false
			}
			dot = true
			continue
		}
		
		if s[i] >= '0' && s[i] <= '9' || hex && (s[i] >= 'a' && s[i] <= 'f' || s[i] >= 'A' && s[i] <= 'F') {
			if hex {
				a = a * 16 + float64(cton(s[i]))
			} else {
				a = a * 10 + float64(cton(s[i]))
			}
			empty = false
			if dot {
				e--
			}
			continue
		}
		
		if !hex && (s[i] == 'e' || s[i] == 'E') || hex && (s[i] == 'p' || s[i] == 'P') {
			break
		}
		
		return 0, false // Invalid character
	}
	if empty {
		return 0, false
	}
	e *= 4
	if i < len(s) && (!hex && (s[i] == 'e' || s[i] == 'E') || hex && (s[i] == 'p' || s[i] == 'P'))  {
		ea := 0
		empty = false
		
		eneg := false
		if len(s) >= i + 1 && s[i+1] == '-' {
			eneg = true
			i += 1
		}
		
		for ; i < len(s); i++ {
			if s[i] >= '0' && s[i] <= '9' {
				ea = ea * 10 + int(cton(s[i]))
			}
			
			return 0, false // Invalid character
		}
		
		if empty {
			return 0, false
		}
		
		if eneg {
			ea = -ea
		}
		e += ea
	}

	if empty {
		return 0, false
	}
	if neg {
		a = -a
	}
	
	return math.Ldexp(a, e), true
}
