// +build ignore

// This file was used by me for testing the compiler, it compiles a chunk of code in both my compiler and the Lua
// reference compiler, then lists and runs the results.
package main

import (
	"github.com/milochristiansen/lua"
	"github.com/milochristiansen/lua/lmodbase"
	"github.com/milochristiansen/lua/lmodpackage"
	"github.com/milochristiansen/lua/lmodstring"
	"github.com/milochristiansen/lua/lmodtable"
	"github.com/milochristiansen/lua/lmodmath"
	
	"strings"
	"fmt"
	
	//"runtime/trace"
	//"os"
)

func main() {
	// The go trace tool does not seem to work properly on my system. The data generates fine (AFAIK),
	// but the HTML UI does not seem to work.
	//file, _ := os.Create("./test.trace")
	//trace.Start(file)
	
	l := lua.NewState()
	
	l.NativeTrace = true
	
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
	
	script := `
	-- Tail call, self recursion, local functions:
	--local function a(v)
	--	print(v)
	--	v = v - 1
	--	if v < 0 then
	--		return v
	--	end
	--	return a(v)
	--end
	--print(a(5))
	
	-- Redeclare local in child block:
	--local a = 5
	--if 1 == 2 then
	--	local a = 10
	--	return a
	--end
	
	-- Upvalues over multiple levels:
	--local a
	--function x()
	--	--a = 1
	--	return function()
	--		a = 5
	--	end
	--
	--end
	
	-- Unconditional jumps:
	--local a = {"a", "b", "c", "d", "e"}
	--for k, v in pairs(a) do
	--	if v == "c" then
	--		goto continue
	--	end
	--	print(k, v)
	--	::continue::
	--end
	
	-- Method calls:
	--local a = {b = function(self) return self.c end, c = 5}
	--print(a:b())
	
	-- Variadic call:
	--function a(...)
	--	local a = {...}
	--	for k, v in ipairs(a) do
	--		print(k, v)
	--	end
	--end
	--a("a", "b", "c")
	
	-- not:
	--local a = true
	--if not a then
	--	print("Error!")
	--end
	--print(a)
	
	-- Literals in suffix expressions:
	--print(({1, 2, 3})[2])
	
	-- Variadic return:
	--function a(...)
	--	return table.unpack({...})
	--end
	--print(a("a", "b", "c"))
	
	-- break:
	--local a = {"a", "b", "c", "d", "e"}
	--local k = 0
	--while true do
	--	k = k + 1
	--	if a[k] == "d" then
	--		break
	--	end
	--	print(k, a[k])
	--end
	
	--local a = {"a", "b"}
	--local b = {"a", "b", "c"}
	--if #b ~= 1 and #b-1 > #a then
	--	print("Error")
	--end
	
	--local a, b = true, false
	--if a and b then
	--	print("A")
	--end
	
	--local a = true
	--if a then
	--	print("ok")
	--end
	
	--print("a".."b".."c" == "abc")
	
	--function a(v) return v.."!" end
	--local b = a "test"
	--print(b)
	
	--print((""):len())
	
	--local k, v
	--local a = {a = 1, b = 2, "x", "y", "z"}
	--while true do
	--	k, v = next(a, k)
	--	if k == nil then
	--		break
	--	end
	--	print(k, v)
	--end
	
	-- Upvalue closure in loops:
	--local a = {}
	--for i = 1, 5, 1 do
	--	table.insert(a, function() return i end)
	--end
	--for k, v in ipairs(a) do
	--	print(v())
	--end
	
	--local a
	--local function b() return a() end
	--a = function() return 5 end
	--print(b())
	
	--local x = (function()
	--	local a
	--	local function b() return a() end
	--	a = function() return 5 end
	--	return b
	--end)()
	--print(x())
	
	-- Unclosed upvalues across multiple levels:
	--local x, y, z = -1, -2, -3
	--local a = function() return y end
	--function b(f)
	--	local x, y, z = -9, -8, -7
	--	local r = f()
	--	return r
	--end
	--print(b(a))
	
	--a = {"a", "b", "c"}
	--function z(b)
	--	local typ = a[rawget(b, "_magic")]
	--	print(typ)
	--end
	--z{_magic = 2}
	
	--local a = nil
	--if a == nil or not next(a) then
	--	print("RTN")
	--end
	
	--if false or true or nil then
	--	print("RTN")
	--end
	
	--print(1 == 2 or 3)
	
	--local a, b, c = 1, 2, 3
	--for i = 1, 5, 1 do
	--	print(a, b, c)
	--	a, b, c = b, c, c + 1
	--end
	
	-- Accessing upvalues across "dead zones" (functions in the middle of the chain that do no contain the upvalue)
	local x, y, z = -1, -2, -3
	function a(f)
		local r = f(function() return y end)
		return r
	end
	print(a(function (f)
		local r = f()
		return r
	end))
	
	`
	
	run := true
	
	l.Println("luac:")
	err := l.LoadTextExternal(strings.NewReader(script), "test.go", 0)
	if err != nil {
		fmt.Println(err)
		return
	}
	
	l.ListFunc(-1)
	
	if run {
		l.Println("\nRun luac:")
		err = l.PCall(0, 0)
		if err != nil {
			fmt.Println(err)
			return
		}
	} else {
		l.Pop(1)
	}
	
	l.Println("\ngithub.com/milochristiansen/lua:")
	err = l.LoadText(strings.NewReader(script), "test.go", 0)
	if err != nil {
		fmt.Println(err)
		return
	}
	
	l.ListFunc(-1)
	
	if run {
		l.Println("\nRun github.com/milochristiansen/lua:")
		err = l.PCall(0, 0)
		if err != nil {
			fmt.Println(err)
			return
		}
	} else {
		l.Pop(1)
	}
	//trace.Stop()
}
