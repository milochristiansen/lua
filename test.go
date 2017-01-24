// +build ignore

// This file was used by me for testing the compiler, it compiles a chunk of code in both my compiler and the Lua
// reference compiler, then lists and runs the results.
package main

import (
	"github.com/milochristiansen/lua"
	"github.com/milochristiansen/lua/lmodbase"
	"github.com/milochristiansen/lua/lmodmath"
	"github.com/milochristiansen/lua/lmodpackage"
	"github.com/milochristiansen/lua/lmodstring"
	"github.com/milochristiansen/lua/lmodtable"

	"fmt"
	"strings"
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

	// I try not to delete old code here, I just comment it out in case I need it later.
	// Probably not the best idea, but it is interesting to come back and see the things
	// I had so much... fun. (or not) debugging...
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
	--local x, y, z = -1, -2, -3
	--function a(f)
	--	local r = f(function() return y end)
	--	return r
	--end
	--print(a(function (f)
	--	local r = f()
	--	return r
	--end))
	
	-- Conflicts in multiple assignment
	--local a = {'a', 'b'}
	--a[1], a = 1, 1
	--a, a[1] = 1, 1
	
	-- Mutual clobber test, upvalues edition
	--local a = {'a', 'b'}
	--local b = a
	--local function foo()
	--	a[1], a = 1, 1
	--end
	--foo()
	--assert(b[1] == 1)
	
	-- Mutual clobber test, locals+upvalues edition
	--local t = {}
	--(function(a)
	--	t[a], a = 10, 20
	--end)(1)
	--assert(t[1] == 10)
	
	-- Test methods
	--local a = {}
	--function a:x() return self end -- Note to "self": Don't forget that this adds an implicit self argument!
	--print(a.x)
	--print(a:x())
	
	-- Test truncating multiple returns
	--local a, b, c = 1, 2, 3
	--a, b, c = (table.unpack({"a", "b"}))
	--print(a, b, c)
	
	-- Multiple values truncated, take two
	--function f(...)
	--	print((...))
	--end
	--f(1,2,3)
	
	-- Test closures sharing a upvalue
	--local a = {}
	--for i = 1, 10 do
	--	a[i] = {
	--		set = function(x)
	--			i = x
	--		end,
	--		get = function()
	--			return i
	--		end,
	--	}
	--	if i == 3 then break end -- Make sure break closes properly for good measure
	--end
	--
	--assert(a[2].get() == 2)
	--a[2].set('a')
	--assert(a[2].get() == 'a')
	--assert(a[3].get() == 3)
	--a[3].set('a')
	--assert(a[3].get() == 'a')
	
	-- Test closing upvalues after an error
	--local b = 2
	--function f(x)
	--	local a = 1
	--	
	--	b = function(y)
	--		print(a, x, y)
	--	end
	--	error()
	--end
	--pcall(f, 2)
	--b(3)
	
	-- Make sure generic for loops close properly
	--local a = {1, 2, 3}
	--local b = {}
	--for k, v in ipairs(a) do
	--	b[k] = {
	--		set = function(x)
	--			v = x
	--		end,
	--		get = function()
	--			return v
	--		end,
	--	}
	--end
	--assert(b[2].get() == 2)
	--b[2].set('a')
	--assert(b[2].get() == 'a')
	--assert(b[3].get() == 3)
	--b[3].set('a')
	--assert(b[3].get() == 'a')
	
	-- Closures with control variables, take two
	--local a = {}
	--local t = {"a", "b"}
	--
	--for i = 1, #t do
	--	local k = t[i]
	--	
	--	a[i] = {
	--		set = function(x, y)
	--			print(i, k)
	--			i = x
	--			k = y -- get can't see k's new value, but set can (as shown by multiple calls) (same result for both compilers)
	--			print(i, k)
	--		end,
	--		get = function()
	--			return i, k
	--		end
	--	}
	--	if i == 2 then
	--		break
	--	end
	--end
	--a[1].set(10, 20)
	--print(a[1].get())
	--print(a[2].get())
	--a[2].set(10, 20)
	--print(a[2].get())
	
	-- Closures in repeat-until loops
	--local a = {}
	--local i = 1
	--repeat
	--	local x = i
	--	a[i] = function()
	--		i = x + 1
	--		return x
	--	end
	--until i > 10 or a[i]() ~= x
	--assert(i == 11 and a[1]() == 1 and a[3]() == 3 and i == 4)
	
	-- luac prints "x"
	-- my compiler prints nil
	--local a, b, c, d, e = 1, 2, false, 1, "x"
	--print(a >= b or c or ( d and e ) or nil)

	--print(a >= b or c or true or nil) -- Prints true for both
	--print(d and e or nil) -- Same as the first one
	--print(e or nil) -- "x" for both
	--print(d and e) -- mine prints the address of "print"

	--local a, b, c = 1, 2
	--c = a and b
	--print(c) -- Prints nil on my compiler.

	--local a, b, c = 1, 2
	--c = a or b
	--print(c) -- Prints "1" for both.

	--local a, i = {}, 1
	--repeat
	--	local x = i
	--	a[i] = function () i = x+1; print(x, i); return x end
	--	print(i > 10 or a[i]() ~= x)
	--	i = i - 1
	--until i > 10 or a[i]() ~= x

	--if 1 > 2 or 2 > 10 then
	if false or false then
		print("ok")
	end

	--print(1 > 10 or 2 ~= 2)

	--local a, i = {}, 1
	--repeat
	--	local x = i
	--	a[i] = function () i = x+1; return x end
	--	print(x, i)
	--until i > 10 or a[i]() ~= x
	--print(i, a[1], a[3])
	--print(i == 11, a[1]() == 1, a[3]() == 3, i == 4)
	--i = 11
	--assert(i == 11 and a[1]() == 1 and a[3]() == 3 and i == 4)

	--local a = {}
	--local i = 1
	--a[1] = function() i = 2 return 1 end
	--a[3] = function() return 3 end
	--assert(i == 1 and a[1]() == 1 and a[3]() == 3 and i == 2)

	`

	run := true
	luac := true

	luac = false

	var err error

	if luac {
		l.Println("luac:")
		err = l.LoadTextExternal(strings.NewReader(script), "test.go", 0)
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
