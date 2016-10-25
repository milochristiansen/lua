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

package lua

import "bytes"
import "fmt"
import "text/tabwriter"

// Anything marked "Debug info" may or may not be set.
// Internally the compiler uses the debug info fields to keep track of some things, but they may
// be striped later and/or never set if a striped binary is loaded directly.
type funcProto struct {
	constants  []value
	code       []instruction
	prototypes []funcProto
	lineInfo   []int // Debug info
	upVals     []upValue
	localVars  []localVar // Debug info

	source          string // Debug info
	lineDefined     int    // Debug info
	lastLineDefined int    // Debug info

	parameterCount int
	maxStackSize   int
	
	// 0 = is not variadic
	// 2 = is variadic but `...` is never used (so there is no need to actually save the parameters)
	// 1 = is variadic and has at least one occurrence of `...`
	isVarArg byte
}

func (f funcProto) String() string {
	return f.str("")
}

func (f funcProto) str(prefix string) string {
	out := new(bytes.Buffer)
	fmt.Fprintf(out, "%v:%v:%v\n", f.source, f.lineDefined, f.lastLineDefined)
	
	w := tabwriter.NewWriter(out, 2, 8, 2, ' ', 0)
	_ = w
	fmt.Fprintf(out, "%v Code:\n", prefix)
	for j, i := range f.code {
		op := i.getOpCode()
		iout := opNames[op]
		extra := ""
		mode := opModes[op]
		if mode.a != 0 {
			iout = fmt.Sprintf("%s\tA:%d", iout, i.a())
		}
		if mode.ax != 0 {
			iout = fmt.Sprintf("%s\tAX:%d", iout, i.ax())
		}
		if mode.a == 0 && mode.ax == 0 {
			iout += "\t"
		}
		
		switch mode.b {
		case 1:
			iout = fmt.Sprintf("%s\tB:%d", iout, i.b())
		case 2:
			if isK(i.b()) {
				iout = fmt.Sprintf("%s\tB:k(%d)", iout, indexK(i.b()))
				extra = fmt.Sprintf("%s BK:%v", extra, f.constants[indexK(i.b())])
			} else {
				iout = fmt.Sprintf("%s\tB:r(%d)", iout, i.b())
			}
		case 3:
			iout = fmt.Sprintf("%s\tB:float8(%d)", iout, intFromFloat8(float8(i.b())))
		}
		if mode.bx != 0 {
			iout = fmt.Sprintf("%s\tBX:%d", iout, i.bx())
		}
		if mode.sbx != 0 {
			iout = fmt.Sprintf("%s\tSBX:%d", iout, i.sbx())
			extra = fmt.Sprintf("%s to:%d", extra, j+i.sbx()+1)
		}
		if mode.b == 0 && mode.bx == 0 && mode.sbx == 0 {
			iout += "\t"
		}
		
		switch mode.c {
		case 1:
			iout = fmt.Sprintf("%s\tC:%d", iout, i.c())
		case 2:
			if isK(i.c()) {
				iout = fmt.Sprintf("%s\tC:k(%d)", iout, indexK(i.c()))
				extra = fmt.Sprintf("%s CK:%v", extra, f.constants[indexK(i.c())])
			} else {
				iout = fmt.Sprintf("%s\tC:r(%d)", iout, i.c())
			}
		case 3:
			iout = fmt.Sprintf("%s\tC:float8(%d)", iout, intFromFloat8(float8(i.c())))
		default:
			iout += "\t"
		}
		
		if extra != "" {
			fmt.Fprintf(w, "%v  [%v]\t%v\t;%v\n", prefix, j, iout, extra)
		} else {
			fmt.Fprintf(w, "%v  [%v]\t%v\t\n", prefix, j, iout)
		}
	}
	w.Flush()
	
	fmt.Fprintf(out, "%v Locals:\n", prefix)
	if len(f.localVars) == 0 {
		fmt.Fprintf(out, "%v  None.\n", prefix)
	}
	for i, v := range f.localVars {
		fmt.Fprintf(w, "%v  [%v]\t\"%v\":\t[%v,%v]\n", prefix, i, v.name, v.sPC-1, v.ePC-1)
	}
	w.Flush()
	
	fmt.Fprintf(out, "%v UpValues:\n", prefix)
	if len(f.upVals) == 0 {
		fmt.Fprintf(out, "%v  None.\n", prefix)
	}
	for i, v := range f.upVals {
		fmt.Fprintf(w, "%v  [%v]\t\"%v\":\tIdx:%v\tIsLocal:%v\n", prefix, i, v.name, v.index, v.isLocal)
	}
	w.Flush()
	
	fmt.Fprintf(out, "%v Constants:\n", prefix)
	if len(f.constants) == 0 {
		fmt.Fprintf(out, "%v  None.\n", prefix)
	}
	for i, v := range f.constants {
		fmt.Fprintf(w, "%v  [%v]\t%#v\n", prefix, i, v)
	}
	w.Flush()
	
	fmt.Fprintf(out, "%v Closures:\n", prefix)
	if len(f.prototypes) == 0 {
		fmt.Fprintf(out, "%v  None.\n", prefix)
	}
	for i, p := range f.prototypes {
		fmt.Fprintf(out, "%v  [%v] %v\n", prefix, i, p.str(fmt.Sprintf("%v  [%v]", prefix, i)))
	}
	
	return string(bytes.TrimSpace(out.Bytes()))
}

type localVar struct {
	name string
	sPC  int32
	ePC  int32
}

type upValue struct {
	// Is this upvalue a reference to one of its parent's locals? (Else it
	// is a reference to one of its parent's upvalues)
	isLocal bool 
	index   int // Index into the parent function's locals or upvalues
	name    string // Debug info
}

// NativeFunction is the prototype to which native API functions must conform.
type NativeFunction func(l *State) int

// function is a Lua or native function with its upvalues.
type function struct {
	proto  funcProto
	native NativeFunction

	uvDefs    []upValue
	uvClosed  []bool
	upVals    []value
	uvAbsIdxs []int
}

// addUp adds a new up value to the function and returns it's index.
func (f *function) addUp(v value) int {
	i := len(f.uvDefs)
	f.uvDefs = append(f.uvDefs, upValue{})
	f.uvClosed = append(f.uvClosed, true)
	f.upVals = append(f.upVals, v)
	f.uvAbsIdxs = append(f.uvAbsIdxs, -1)
	return i
}
