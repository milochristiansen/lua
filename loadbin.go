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

import "encoding/binary"
import "io"

import "github.com/milochristiansen/lua/luautil"

// On all systems this VM uses 64 bit ints and floats with LE byte order.
// If you want something else feel free to write a better loader and send it to me :)
//
//  * 4 bytes: magic prefix (<ESC>Lua)
//	* 1 byte: hex version
//	* 6 bytes: more magic crap
//	* 1 byte: int size in bytes (4) (really should be 8, but C is stupid so I need to use 4)
//	* 1 byte: pointer size in bytes (8)
//	* 1 byte: instruction size in bytes (4)
//	* 1 byte: int number type size in bytes (8)
//	* 1 byte: float number type size in bytes (8)
//	* 8 bytes: more magic. A type int number (0x7856000000000000 as encoded)
//	* 8 bytes: more magic. A type float number (0x0000000000287740 as encoded)
var binHeader64 = "\x1bLua\x53\x00\x19\x93\x0d\x0a\x1a\x0a\x04\x08\x04\x08\x08\x78\x56\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x28\x77\x40"

// Same as binHeader64 but with 4 byte pointer size.
var binHeader32 = "\x1bLua\x53\x00\x19\x93\x0d\x0a\x1a\x0a\x04\x04\x04\x08\x08\x78\x56\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x28\x77\x40"

type loader struct {
	rdr io.Reader
	b32 bool // file compiled with 32 bit pointers?
}

func (l loader) read(data interface{}) error {
	return binary.Read(l.rdr, binary.LittleEndian, data)
}

func (l loader) readInt() (int32, error) {
	var i int32
	err := l.read(&i)
	return i, err
}

func (l loader) readByte() (byte, error) {
	var b byte
	err := l.read(&b)
	return b, err
}

func (l loader) readString() (string, error) {
	sb, err := l.readByte()
	if err != nil {
		return "", err
	}

	if sb == 0 {
		return "", nil
	}

	// For some stupid reason they use a size_t value for this...
	size := 0
	if l.b32 {
		s := int32(sb)
		if sb == 0xff {
			err = l.read(&s)
			if err != nil {
				return "", err
			}
		}
		size = int(s)
	} else {
		s := int64(sb)
		if sb == 0xff {
			err = l.read(&s)
			if err != nil {
				return "", err
			}
		}
		size = int(s)
	}

	rstr := make([]byte, size-1)
	err = l.read(rstr)
	if err != nil {
		return "", err
	}
	return string(rstr), nil
}

func (l loader) readCode(fp *funcProto) error {
	n, err := l.readInt()
	if err != nil {
		return err
	}

	code := make([]instruction, n)
	err = l.read(code)
	if err != nil {
		return err
	}

	fp.code = code
	return nil
}

func (l loader) readConstants(fp *funcProto) error {
	n, err := l.readInt()
	if err != nil {
		return err
	}

	constants := make([]value, n)
	for i := range constants {
		t, err := l.readByte()
		if err != nil {
			return err
		}

		switch t {
		case 0: // LUA_TNIL
			constants[i] = nil

		case 1: // LUA_TBOOLEAN
			b, err := l.readByte()
			if err != nil {
				return err
			}
			constants[i] = b != 0

		case 3 | (0 << 4): // LUA_TNUMFLT
			var n float64
			err := l.read(&n)
			if err != nil {
				return err
			}
			constants[i] = n

		case 3 | (1 << 4): // LUA_TNUMINT
			var n int64
			err := l.read(&n)
			if err != nil {
				return err
			}
			constants[i] = n

		case 4 | (0 << 4): // LUA_TSHRSTR
			fallthrough
		case 4 | (1 << 4): // LUA_TLNGSTR
			v, err := l.readString()
			if err != nil {
				return err
			}
			constants[i] = v

		default:
			//  cvartag
			// 0110 0100
			// 0001 0100
			return luautil.Error{Msg: "Bin Loader: Invalid constant type", Type: luautil.ErrTypBinLoader}
		}
	}

	fp.constants = constants
	return nil
}

func (l loader) readUpValues(fp *funcProto) error {
	n, err := l.readInt()
	if err != nil {
		return err
	}

	v := make([]struct{ IsLocal, Index byte }, n)
	err = l.read(v)
	if err != nil {
		return err
	}

	ups := make([]upDef, n)
	for i := range v {
		ups[i] = upDef{
			isLocal: v[i].IsLocal != 0,
			index:   int(v[i].Index),
		}
	}
	fp.upVals = ups
	return nil
}

func (l loader) readProto(fp *funcProto) error {
	n, err := l.readInt()
	if err != nil {
		return err
	}

	prototypes := make([]funcProto, n)
	for i := range prototypes {
		nfp, err := l.readFunction(fp.source)
		if err != nil {
			return err
		}
		prototypes[i] = *nfp
	}

	fp.prototypes = prototypes
	return nil
}

func (l loader) readDebug(fp *funcProto) error {
	n, err := l.readInt()
	if err != nil {
		return err
	}

	lineInfo := make([]int, n)
	for i := range lineInfo {
		line, err := l.readInt()
		if err != nil {
			return err
		}
		lineInfo[i] = int(line)
	}

	n, err = l.readInt()
	if err != nil {
		return err
	}

	localVars := make([]localVar, n)
	for i := range localVars {
		localVars[i].name, err = l.readString()
		if err != nil {
			return err
		}

		localVars[i].sPC, err = l.readInt()
		if err != nil {
			return err
		}

		localVars[i].ePC, err = l.readInt()
		if err != nil {
			return err
		}
	}

	n, err = l.readInt()
	if err != nil {
		return err
	}

	names := make([]string, n)
	for i := range names {
		names[i], err = l.readString()
		if err != nil {
			return err
		}
	}

	if len(fp.upVals) > len(names) {
		return luautil.Error{Msg: "Bin Loader: More upvals defined than names", Type: luautil.ErrTypBinLoader}
	}

	fp.lineInfo = lineInfo
	fp.localVars = localVars

	for i, name := range names {
		fp.upVals[i].name = name
	}
	return nil
}

func (l loader) readFunction(psrc string) (*funcProto, error) {
	fp := &funcProto{}

	src, err := l.readString()
	if err != nil {
		return nil, err
	}
	if src == "" {
		src = psrc
	}
	fp.source = src

	n, err := l.readInt()
	if err != nil {
		return nil, err
	}
	fp.lineDefined = int(n)

	n, err = l.readInt()
	if err != nil {
		return nil, err
	}
	fp.lastLineDefined = int(n)

	b, err := l.readByte()
	if err != nil {
		return nil, err
	}
	fp.parameterCount = int(b)

	b, err = l.readByte()
	if err != nil {
		return nil, err
	}
	fp.isVarArg = b

	b, err = l.readByte()
	if err != nil {
		return nil, err
	}
	fp.maxStackSize = int(b)

	err = l.readCode(fp)
	if err != nil {
		return nil, err
	}

	err = l.readConstants(fp)
	if err != nil {
		return nil, err
	}

	err = l.readUpValues(fp)
	if err != nil {
		return nil, err
	}

	err = l.readProto(fp)
	if err != nil {
		return nil, err
	}

	err = l.readDebug(fp)
	if err != nil {
		return nil, err
	}
	return fp, err
}

func loadBin(in io.Reader, name string) (*funcProto, error) {
	if len(name) > 0 && (name[0] == '@' || name[0] == '=') {
		name = name[1:]
	} else if len(name) > 0 && name[0] == binHeader64[0] {
		name = "binary string"
	}

	l := loader{in, false}
	header := make([]byte, len(binHeader64))
	err := l.read(header)
	if err != nil {
		return nil, luautil.Error{Msg: "Bin Loader", Err: err, Type: luautil.ErrTypBinLoader}
	}
	if string(header) != binHeader64 {
		if string(header) != binHeader32 {
			return nil, luautil.Error{Msg: "Bin Loader: Header mismatch, not binary chunk or incorrect format", Type: luautil.ErrTypBinLoader}
		}
		l.b32 = true
	}

	_, err = l.readByte() // The number of upvals the main chunk has
	if err != nil {
		return nil, err
	}

	fp, err := l.readFunction(name)
	if err != nil {
		if _, ok := err.(luautil.Error); !ok {
			return nil, luautil.Error{Msg: "Bin Loader", Err: err, Type: luautil.ErrTypBinLoader}
		}
		return nil, err
	}
	return fp, nil
}
