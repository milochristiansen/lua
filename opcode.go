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

import "fmt"

// I want to be able to load binaries written by the standard Lua compiler, so I may as well
// use the standard opcode format.
//
// From a performance view point it would probably be better to waste some memory and use
// structures instead of hoping the go compiler will inline all this bit twiddling...

type opCode uint

const (
	opMove opCode = iota
	opLoadK
	opLoadKEx
	opLoadBool
	opLoadNil

	opGetUpValue
	opGetTableUp
	opGetTable

	opSetTableUp
	opSetUpValue
	opSetTable

	opNewTable

	opSelf

	// These exported values are used in calls to Arith
	OpAdd
	OpSub
	OpMul
	OpMod
	OpPow
	OpDiv
	OpIDiv
	OpBinAND
	OpBinOR
	OpBinXOR
	OpBinShiftL
	OpBinShiftR
	OpUMinus
	OpBinNot
	opNot
	opLength

	opConcat

	// These exported values are used in calls to Compare
	opJump
	OpEqual
	OpLessThan
	OpLessOrEqual

	opTest
	opTestSet

	opCall
	opTailCall
	opReturn

	opForLoop
	opForPrep

	opTForCall
	opTForLoop

	opSetList

	opClosure

	opVarArg

	opExtraArg

	opCodeCount int = iota
)

var opNames = []string{
	"MOVE",
	"LOADK",
	"LOADKX",
	"LOADBOOL",
	"LOADNIL",

	"GETUPVAL",
	"GETTABUP",
	"GETTABLE",

	"SETTABUP",
	"SETUPVAL",
	"SETTABLE",

	"NEWTABLE",

	"SELF",

	"ADD",
	"SUB",
	"MUL",
	"MOD",
	"POW",
	"DIV",
	"IDIV",
	"BAND",
	"BOR",
	"BXOR",
	"SHL",
	"SHR",
	"UNM",
	"BNOT",
	"NOT",
	"LEN",

	"CONCAT",

	"JMP",
	"EQ",
	"LT",
	"LE",

	"TEST",
	"TESTSET",

	"CALL",
	"TAILCALL",
	"RETURN",

	"FORLOOP",
	"FORPREP",

	"TFORCALL",
	"TFORLOOP",

	"SETLIST",

	"CLOSURE",

	"VARARG",

	"EXTRAARG",
}

const (
	sizeC  = 9
	sizeB  = 9
	sizeBx = sizeC + sizeB
	sizeA  = 8
	sizeAx = sizeC + sizeB + sizeA

	sizeOp = 6

	posOp = 0
	posA  = posOp + sizeOp
	posC  = posA + sizeA
	posB  = posC + sizeC
	posBx = posC
	posAx = posA

	bitRK      = 1 << (sizeB - 1)
	maxIndexRK = bitRK - 1

	maxArgAx  = 1<<sizeAx - 1
	maxArgBx  = 1<<sizeBx - 1
	maxArgSBx = maxArgBx >> 1
	maxArgA   = 1<<sizeA - 1
	maxArgB   = 1<<sizeB - 1
	maxArgC   = 1<<sizeC - 1

	fieldsPerFlush = 50
)

func isK(x int) bool   { return 0 != x&bitRK }
func indexK(r int) int { return r & ^bitRK }
func rkAsK(r int) int  { return r | bitRK }

type instruction uint32

// creates a mask with 'n' 1 bits at position 'p'
func mask1(n, p uint) instruction { return ^(^instruction(0) << n) << p }

// creates a mask with 'n' 0 bits at position 'p'
func mask0(n, p uint) instruction { return ^mask1(n, p) }

func (i instruction) getOpCode() opCode {
	return opCode(i >> posOp & (1<<sizeOp - 1))
}

func (i *instruction) setOpCode(op opCode) {
	i.setArg(posOp, sizeOp, int(op))
}

func (i instruction) getArg(pos, size uint) int {
	return int(i >> pos & mask1(size, 0))
}

func (i *instruction) setArg(pos, size uint, arg int) {
	*i = *i&mask0(size, pos) | instruction(arg)<<pos&mask1(size, pos)
}

// Inline for performance.
func (i instruction) a() int   { return int(i >> posA & maxArgA) }
func (i instruction) b() int   { return int(i >> posB & maxArgB) }
func (i instruction) c() int   { return int(i >> posC & maxArgC) }
func (i instruction) bx() int  { return int(i >> posBx & maxArgBx) }
func (i instruction) ax() int  { return int(i >> posAx & maxArgAx) }
func (i instruction) sbx() int { return int(i>>posBx&maxArgBx) - maxArgSBx }

func (i *instruction) setA(arg int)   { i.setArg(posA, sizeA, arg) }
func (i *instruction) setB(arg int)   { i.setArg(posB, sizeB, arg) }
func (i *instruction) setC(arg int)   { i.setArg(posC, sizeC, arg) }
func (i *instruction) setBx(arg int)  { i.setArg(posBx, sizeBx, arg) }
func (i *instruction) setAx(arg int)  { i.setArg(posAx, sizeAx, arg) }
func (i *instruction) setSBx(arg int) { i.setArg(posBx, sizeBx, arg+maxArgSBx) }

func createABC(op opCode, a, b, c int) instruction {
	return instruction(op)<<posOp | instruction(a)<<posA | instruction(b)<<posB | instruction(c)<<posC
}

func createABx(op opCode, a, bx int) instruction {
	return instruction(op)<<posOp | instruction(a)<<posA | instruction(bx)<<posBx
}

func createAsBx(op opCode, a, sbx int) instruction {
	return instruction(op)<<posOp | instruction(a)<<posA | instruction(sbx+maxArgSBx)<<posBx
}

func createAx(op opCode, a int) instruction {
	return instruction(op)<<posOp | instruction(a)<<posAx
}

type opType struct {
	a, ax, b, bx, sbx, c int8 // 0: unused, 1: used, 2: used RK, 3: used float 8
}

var opModes = []opType{
	//     a, ax, b, bx, sbx, c      opCode
	opType{1, 0, 1, 0, 0, 0}, // opMove
	opType{1, 0, 0, 1, 0, 0}, // opLoadK
	opType{1, 0, 0, 0, 0, 0}, // opLoadKEx
	opType{1, 0, 1, 0, 0, 1}, // opLoadBool
	opType{1, 0, 1, 0, 0, 0}, // opLoadNil

	opType{1, 0, 1, 0, 0, 0}, // opGetUpValue
	opType{1, 0, 1, 0, 0, 2}, // opGetTableUp
	opType{1, 0, 1, 0, 0, 2}, // opGetTable

	opType{1, 0, 2, 0, 0, 2}, // opSetTableUp
	opType{1, 0, 1, 0, 0, 0}, // opSetUpValue
	opType{1, 0, 2, 0, 0, 2}, // opSetTable

	opType{1, 0, 3, 0, 0, 3}, // opNewTable

	opType{1, 0, 1, 0, 0, 2}, // opSelf

	opType{1, 0, 2, 0, 0, 2}, // opAdd
	opType{1, 0, 2, 0, 0, 2}, // opSub
	opType{1, 0, 2, 0, 0, 2}, // opMul
	opType{1, 0, 2, 0, 0, 2}, // opMod
	opType{1, 0, 2, 0, 0, 2}, // opPow
	opType{1, 0, 2, 0, 0, 2}, // opDiv
	opType{1, 0, 2, 0, 0, 2}, // opIDiv
	opType{1, 0, 2, 0, 0, 2}, // opBinAND
	opType{1, 0, 2, 0, 0, 2}, // opBinOR
	opType{1, 0, 2, 0, 0, 2}, // opBinXOR
	opType{1, 0, 2, 0, 0, 2}, // opBinShiftL
	opType{1, 0, 2, 0, 0, 2}, // opBinShiftR
	opType{1, 0, 2, 0, 0, 0}, // opUMinus
	opType{1, 0, 2, 0, 0, 0}, // opBinNot
	opType{1, 0, 2, 0, 0, 0}, // opNot
	opType{1, 0, 1, 0, 0, 0}, // opLength

	opType{1, 0, 1, 0, 0, 1}, // opConcat

	opType{1, 0, 0, 0, 1, 0}, // opJump
	opType{1, 0, 2, 0, 0, 2}, // opEqual
	opType{1, 0, 2, 0, 0, 2}, // opLessThan
	opType{1, 0, 2, 0, 0, 2}, // opLessOrEqual

	opType{1, 0, 0, 0, 0, 1}, // opTest
	opType{1, 0, 1, 0, 0, 1}, // opTestSet

	opType{1, 0, 1, 0, 0, 1}, // opCall
	opType{1, 0, 1, 0, 0, 0}, // opTailCall
	opType{1, 0, 1, 0, 0, 0}, // opReturn

	opType{1, 0, 0, 0, 1, 0}, // opForLoop
	opType{1, 0, 0, 0, 1, 0}, // opForPrep

	opType{1, 0, 0, 0, 0, 1}, // opTForCall
	opType{1, 0, 0, 0, 1, 0}, // opTForLoop

	opType{1, 0, 1, 0, 0, 1}, // opSetList

	opType{1, 0, 0, 1, 0, 0}, // opClosure

	opType{1, 0, 1, 0, 0, 0}, // opVarArg

	opType{0, 1, 0, 0, 0, 0}, // opExtraArg
}

func (i instruction) String() string {
	op := i.getOpCode()
	out := opNames[op]
	mode := opModes[op]
	if mode.a != 0 {
		out = fmt.Sprintf("%s\tA:%d", out, i.a())
	}
	if mode.ax != 0 {
		out = fmt.Sprintf("%s\tAX:%d", out, i.ax())
	}

	switch mode.b {
	case 1:
		out = fmt.Sprintf("%s\tB:%d", out, i.b())
	case 2:
		if isK(i.b()) {
			out = fmt.Sprintf("%s\tB:k(%d)", out, indexK(i.b()))
		} else {
			out = fmt.Sprintf("%s\tB:r(%d)", out, i.b())
		}
	case 3:
		out = fmt.Sprintf("%s\tB:float8(%d)", out, intFromFloat8(float8(i.b())))
	}
	if mode.bx != 0 {
		out = fmt.Sprintf("%s\tBX:%d", out, i.bx())
	}
	if mode.sbx != 0 {
		out = fmt.Sprintf("%s\tSBX:%d", out, i.sbx())
	}

	switch mode.c {
	case 1:
		out = fmt.Sprintf("%s\tC:%d", out, i.c())
	case 2:
		if isK(i.c()) {
			out = fmt.Sprintf("%s\tC:k(%d)", out, indexK(i.c()))
		} else {
			out = fmt.Sprintf("%s\tC:r(%d)", out, i.c())
		}
	case 3:
		out = fmt.Sprintf("%s\tC:float8(%d)", out, intFromFloat8(float8(i.c())))
	}
	return out
}
