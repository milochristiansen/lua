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

import "encoding/binary"
import "bytes"

import "github.com/milochristiansen/lua/luautil"

type dumper struct {
	w *bytes.Buffer
}

func (d dumper) write(data interface{}) {
	binary.Write(d.w, binary.LittleEndian, data)
}

func (d dumper) writeInt(i int32) {
	d.write(i)
}

func (d dumper) writeByte(b byte) {
	d.write(b)
}

func (d dumper) writeString(s string) {
	l := len(s)
	if l == 0 {
		d.writeByte(0)
		return
	}
	
	l++ // Plus one for the non-existent zero terminator
	if l >= 0xff {
		d.writeByte(0xff)
		d.write(int64(l))
	} else {
		d.writeByte(byte(l))
	}
	
	d.write([]byte(s))
}

func (d dumper) writeCode(fp *funcProto) {
	d.writeInt(int32(len(fp.code)))
	
	d.write(fp.code)
}

func (d dumper) writeConstants(fp *funcProto) {
	d.writeInt(int32(len(fp.constants)))
	
	for _, v := range fp.constants {
		switch v2 := v.(type) {
		case nil:
			d.writeByte(0) // LUA_TNIL

		case bool:
			d.writeByte(1) // LUA_TBOOLEAN
			if v2 {
				d.writeByte(1)
			} else {
				d.writeByte(0)
			}

		case float64:
			d.writeByte(3 | (0 << 4)) // LUA_TNUMFLT
			d.write(v2)
		
		case int64:
			d.writeByte(3 | (1 << 4)) // LUA_TNUMINT
			d.write(v2)

		case string:
			if len(v2) > 40 { // LUAI_MAXSHORTLEN
				d.writeByte(4 | (1 << 4)) // LUA_TLNGSTR
			} else {
				d.writeByte(4 | (0 << 4)) // LUA_TSHRSTR
			}
			d.writeString(v2)
		
		default:
			luautil.Raise("Bin Dumper: Invalid constant type", luautil.ErrTypBinDumper)
		}
	}
}

func (d dumper) writeUpValues(fp *funcProto) {
	d.writeInt(int32(len(fp.upVals)))
	
	for _, v := range fp.upVals {
		if v.isLocal {
			d.writeByte(1)
		} else {
			d.writeByte(0)
		}
		d.writeByte(byte(v.index))
	}
}

func (d dumper) writeProto(fp *funcProto) {
	d.writeInt(int32(len(fp.prototypes)))

	for _, v := range fp.prototypes {
		d.writeFunction(fp.source, &v)
	}
}

func (d dumper) writeDebug(fp *funcProto) {
	d.writeInt(int32(len(fp.lineInfo)))

	for _, v := range fp.lineInfo {
		d.writeInt(int32(v))
	}

	d.writeInt(int32(len(fp.localVars)))

	for _, v := range fp.localVars {
		d.writeString(v.name)
		d.writeInt(int32(v.sPC))
		d.writeInt(int32(v.ePC))
	}

	d.writeInt(int32(len(fp.upVals)))
	
	for _, v := range fp.upVals {
		d.writeString(v.name)
	}
}

func (d dumper) writeFunction(psrc string, fp *funcProto) {
	if fp.source == psrc {
		d.writeString("")
	} else {
		d.writeString(fp.source)
	}
	
	d.writeInt(int32(fp.lineDefined))
	d.writeInt(int32(fp.lastLineDefined))
	d.writeByte(byte(fp.parameterCount))
	d.writeByte(fp.isVarArg)
	d.writeByte(byte(fp.maxStackSize))

	d.writeCode(fp)
	d.writeConstants(fp)
	d.writeUpValues(fp)
	d.writeProto(fp)
	d.writeDebug(fp)
}

func dumpBin(fp *funcProto) []byte {
	out := new(bytes.Buffer)
	d := dumper{out}
	
	d.write([]byte(binHeader64))
	d.writeByte(byte(len(fp.upVals)))
	d.writeFunction("", fp)
	
	return out.Bytes()
}
