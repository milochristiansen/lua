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

package lua_test

import "testing"
import "strings"

import "github.com/milochristiansen/lua"
import "github.com/milochristiansen/lua/lmodbase"
import "github.com/milochristiansen/lua/lmodpackage"
import "github.com/milochristiansen/lua/lmodstring"
import "github.com/milochristiansen/lua/lmodtable"
import "github.com/milochristiansen/lua/lmodmath"

func assertBlock(t *testing.T, blk string, v interface{}) {
	l := lua.NewState()
	
	// Don't include the string extensions.
	l.Push("_NO_STRING_EXTS")
	l.Push(true)
	l.SetTableRaw(lua.RegistryIndex)
	
	// Require the standard libraries
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
	
	err := l.LoadText(strings.NewReader(blk), "error", 0)
	if err != nil {
		t.Error(err)
		return
	}
	
	err = l.PCall(0, 1)
	if err != nil {
		t.Error(err)
		return
	}
	
	l.Push(v)
	if !l.Compare(-1, -2, lua.OpEqual) {
		t.Error("Test did not return expected value: %v vs %v", l.ToString(-1), l.ToString(-2))
	}
}

// The tests in this file run blocks of code from the official Lua test suite. Most (if not all) of these tests are
// modified in some way, mostly to remove stuff dependent on APIs not available in my VM.

func TestAssign(t *testing.T) {
	assertBlock(t, `-- attrib.lua
local res, res2 = 27

a, b = 1, 2+3
assert(a==1 and b==5)
a={}
function f() return 10, 11, 12 end
a.x, b, a[1] = 1, 2, f()
assert(a.x==1 and b==2 and a[1]==10)
a[f()], b, a[f()+3] = f(), a, 'x'
assert(a[10] == 10 and b == a and a[13] == 'x')

do
  local f = function (n) local x = {}; for i=1,n do x[i]=i end;
                         return table.unpack(x) end;
  local a,b,c
  a,b = 0, f(1)
  assert(a == 0 and b == 1)
  A,b = 0, f(1)
  assert(A == 0 and b == 1)
  a,b,c = 0,5,f(4)
  assert(a==0 and b==5 and c==1)
  a,b,c = 0,5,f(0)
  assert(a==0 and b==5 and c==nil)
end

a, b, c, d = 1 and nil, 1 or nil, (1 and (nil or 1)), 6
assert(not a and b and c and d==6)

d = 20
a, b, c, d = f()
assert(a==10 and b==11 and c==12 and d==nil)
a,b = f(), 1, 2, 3, f()
assert(a==10 and b==1)

assert(a<b == false and a>b == true)
assert((10 and 2) == 2)
assert((10 or 2) == 10)
assert((10 or assert(nil)) == 10)
assert(not (nil and assert(nil)))
assert((nil or "alo") == "alo")
assert((nil and 10) == nil)
assert((false and 10) == false)
assert((true or 10) == true)
assert((false or 10) == 10)
assert(false ~= nil)
assert(nil ~= false)
assert(not nil == true)
assert(not not nil == false)
assert(not not 1 == true)
assert(not not a == true)
assert(not not (6 or nil) == true)
assert(not not (nil and 56) == false)
assert(not not (nil and true) == false)
assert(not 10 == false)
assert(not {} == false)
assert(not 0.5 == false)
assert(not "x" == false)

assert({} ~= {})

a = {}
a[true] = 20
a[false] = 10
assert(a[1<2] == 20 and a[1>2] == 10)

function f(a) return a end

local a = {}
for i=3000,-3000,-1 do a[i + 0.0] = i; end
a[10e30] = "alo"; a[true] = 10; a[false] = 20
assert(a[10e30] == 'alo' and a[not 1] == 20 and a[10<20] == 10)
for i=3000,-3000,-1 do assert(a[i] == i); end
a[print] = assert
a[f] = print
a[a] = a
assert(a[a][a][a][a][print] == assert)
a[print](a[a[f]] == a[print])
assert(not pcall(function () local a = {}; a[nil] = 10 end))
assert(not pcall(function () local a = {[nil] = 10} end))
assert(a[nil] == nil)
a = nil

a = {10,9,8,7,6,5,4,3,2; [-3]='a', [f]=print, a='a', b='ab'}
a, a.x, a.y = a, a[-3]
assert(a[1]==10 and a[-3]==a.a and a[f]==print and a.x=='a' and not a.y)
a[1], f(a)[2], b, c = {['alo']=assert}, 10, a[1], a[f], 6, 10, 23, f(a), 2
a[1].alo(a[2]==10 and b==10 and c==print)


-- test of large float/integer indices 

-- compute maximum integer where all bits fit in a float
local maxint = math.maxinteger

while maxint - 1.0 == maxint - 0.0 do   -- trim (if needed) to fit in a float
  maxint = maxint // 2
end

maxintF = maxint + 0.0   -- float version

assert(math.type(maxintF) == "float" and maxintF >= 2.0^14)

-- floats and integers must index the same places
a[maxintF] = 10; a[maxintF - 1.0] = 11;
a[-maxintF] = 12; a[-maxintF + 1.0] = 13;

assert(a[maxint] == 10 and a[maxint - 1] == 11 and
       a[-maxint] == 12 and a[-maxint + 1] == 13)

a[maxint] = 20
a[-maxint] = 22

assert(a[maxintF] == 20 and a[maxintF - 1.0] == 11 and
       a[-maxintF] == 22 and a[-maxintF + 1.0] == 13)

a = nil


-- test conflicts in multiple assignment
do
  local a,i,j,b
  a = {'a', 'b'}; i=1; j=2; b=a
  i, a[i], a, j, a[j], a[i+j] = j, i, i, b, j, i
  assert(i == 2 and b[1] == 1 and a == 1 and j == b and b[2] == 2 and
         b[3] == 1)
end

-- repeat test with upvalues
do
  local a,i,j,b
  a = {'a', 'b'}; i=1; j=2; b=a
  local function foo ()
    i, a[i], a, j, a[j], a[i+j] = j, i, i, b, j, i
  end
  foo()
  assert(i == 2 and b[1] == 1 and a == 1 and j == b and b[2] == 2 and
         b[3] == 1)
  local t = {}
  (function (a) t[a], a = 10, 20  end)(1);
  assert(t[1] == 10)
end

-- bug in 5.2 beta
local function foo ()
  local a
  return function ()
    local b
    a, b = 3, 14    -- local and upvalue have same index
    return a, b
  end
end

local a, b = foo()()
assert(a == 3 and b == 14)

return res
`, 27)
}

func TestBits(t *testing.T) {
	assertBlock(t, `-- bitwise.lua
local numbits = 64

assert(~0 == -1)

assert((1 << (numbits - 1)) == math.mininteger)

-- basic tests for bitwise operators;
-- use variables to avoid constant folding
local a, b, c, d
a = 0xFFFFFFFFFFFFFFFF
assert(a == -1 and a & -1 == a and a & 35 == 35)
a = 0xF0F0F0F0F0F0F0F0
assert(a | -1 == -1)
assert(a ~ a == 0 and a ~ 0 == a and a ~ ~a == -1)
assert(a >> 4 == ~a)
a = 0xF0; b = 0xCC; c = 0xAA; d = 0xFD
assert(a | b ~ c & d == 0xF4)

a = 0xF0000000; b = 0xCC000000;
c = 0xAA000000; d = 0xFD000000
assert(a | b ~ c & d == 0xF4000000)
assert(~~a == a and ~a == -1 ~ a and -d == ~d + 1)

a = a << 32
b = b << 32
c = c << 32
d = d << 32
assert(a | b ~ c & d == 0xF4000000 << 32)
assert(~~a == a and ~a == -1 ~ a and -d == ~d + 1)

assert(-1 >> 1 == (1 << (numbits - 1)) - 1 and 1 << 31 == 0x80000000)
assert(-1 >> (numbits - 1) == 1)
assert(-1 >> numbits == 0 and
       -1 >> -numbits == 0 and
       -1 << numbits == 0 and
       -1 << -numbits == 0)

assert((2^30 - 1) << 2^30 == 0)
assert((2^30 - 1) >> 2^30 == 0)

assert(1 >> -3 == 1 << 3 and 1000 >> 5 == 1000 << -5)


-- coercion from strings to integers
assert("0xffffffffffffffff" | 0 == -1)
assert("0xfffffffffffffffe" & "-1" == -2)
assert(" \t-0xfffffffffffffffe\n\t" & "-1" == 2)
assert("   \n  -45  \t " >> "  -2  " == -45 * 4)

-- embedded zeros
assert(not pcall(function () return "0xffffffffffffffff\0" | 0 end))
`, nil)
}

func TestCalls(t *testing.T) {
	assertBlock(t, `-- calls.lua
-- get the opportunity to test 'type' too ;)

assert(type(1<2) == 'boolean')
assert(type(true) == 'boolean' and type(false) == 'boolean')
assert(type(nil) == 'nil'
   and type(-3) == 'number'
   and type'x' == 'string'
   and type{} == 'table'
   and type(type) == 'function')

assert(type(assert) == type(print))
function f (x) return a:x (x) end
assert(type(f) == 'function')
--assert(not pcall(type)) -- type() == type(nil) - Milo

-- testing local-function recursion
fact = false
do
  local res = 1
  local function fact (n)
    if n==0 then return res
    else return n*fact(n-1)
    end
  end
  assert(fact(5) == 120)
end
assert(fact == false)

-- testing declarations
a = {i = 10}
self = 20
function a:x (x) return x+self.i end
function a.y (x) return x+self end

assert(a:x(1)+10 == a.y(1))

a.t = {i=-100}
a["t"].x = function (self, a,b) return self.i+a+b end

assert(a.t:x(2,3) == -95)

do
  local a = {x=0}
  function a:add (x) self.x, a.y = self.x+x, 20; return self end
  assert(a:add(10):add(20):add(30).x == 60 and a.y == 20)
end

local a = {b={c={}}}

function a.b.c.f1 (x) return x+1 end
function a.b.c:f2 (x,y) self[x] = y end
assert(a.b.c.f1(4) == 5)
a.b.c:f2('k', 12); assert(a.b.c.k == 12)


t = nil   -- 'declare' t
function f(a,b,c) local d = 'a'; t={a,b,c,d} end

f(      -- this line change must be valid
  1,2)
assert(t[1] == 1 and t[2] == 2 and t[3] == nil and t[4] == 'a')
f(1,2,   -- this one too
      3,4)
assert(t[1] == 1 and t[2] == 2 and t[3] == 3 and t[4] == 'a')

function fat(x)
  if x <= 1 then return 1
  else return x*load("return fat(" .. x-1 .. ")", "")()
  end
end

assert(load "load 'assert(fat(6)==720)' () ")()
a = load('return fat(5), 3')
a,b = a()
assert(a == 120 and b == 3)


function err_on_n (n)
  if n==0 then error(); exit(1);
  else err_on_n (n-1); exit(1);
  end
end

do
  function dummy (n)
    if n > 0 then
      assert(not pcall(err_on_n, n))
      dummy(n-1)
    end
  end
end

dummy(10)

function deep (n)
  if n>0 then deep(n-1) end
end
deep(10)
deep(200)

-- testing tail call
function deep (n) if n>0 then return deep(n-1) else return 101 end end
assert(deep(30000) == 101)
a = {}
function a:deep (n) if n>0 then return self:deep(n-1) else return 101 end end
assert(a:deep(30000) == 101)


a = nil
(function (x) a=x end)(23)
assert(a == 23 and (function (x) return x*2 end)(20) == 40)


-- testing closures

-- fixed-point operator
Z = function (le)
      local function a (f)
        return le(function (x) return f(f)(x) end)
      end
      return a(a)
    end


-- non-recursive factorial

F = function (f)
      return function (n)
               if n == 0 then return 1
               else return n*f(n-1) end
             end
    end

fat = Z(F)

assert(fat(0) == 1 and fat(4) == 24 and Z(F)(5)==5*Z(F)(4))

local function g (z)
  local function f (a,b,c,d)
    return function (x,y) return a+b+c+d+a+x+y+z end
  end
  return f(z,z+1,z+2,z+3)
end

f = g(10)
assert(f(9, 16) == 10+11+12+13+10+9+16+10)

Z, F, f = nil

-- testing multiple returns

function unlpack (t, i)
  i = i or 1
  if (i <= #t) then
    return t[i], unlpack(t, i+1)
  end
end

function equaltab (t1, t2)
  assert(#t1 == #t2)
  for i = 1, #t1 do
    assert(t1[i] == t2[i])
  end
end

local pack = function (...) return (table.pack(...)) end

function f() return 1,2,30,4 end
function ret2 (a,b) return a,b end

local a,b,c,d = unlpack{1,2,3}
assert(a==1 and b==2 and c==3 and d==nil)
a = {1,2,3,4,false,10,'alo',false,assert}
equaltab(pack(unlpack(a)), a)
equaltab(pack(unlpack(a), -1), {1,-1})
a,b,c,d = ret2(f()), ret2(f())
assert(a==1 and b==1 and c==2 and d==nil)
a,b,c,d = unlpack(pack(ret2(f()), ret2(f())))
assert(a==1 and b==1 and c==2 and d==nil)
a,b,c,d = unlpack(pack(ret2(f()), (ret2(f()))))
assert(a==1 and b==1 and c==nil and d==nil)

a = ret2{ unlpack{1,2,3}, unlpack{3,2,1}, unlpack{"a", "b"}}
assert(a[1] == 1 and a[2] == 3 and a[3] == "a" and a[4] == "b")


-- testing calls with 'incorrect' arguments
rawget({}, "x", 1)
rawset({}, "x", 1, 2)
assert(math.sin(1,2) == math.sin(1))
table.sort({10,9,8,4,19,23,0,0}, function (a,b) return a<b end, "extra arg")

-- test for long method names
do
  local t = {x = 1}
  function t:_012345678901234567890123456789012345678901234567890123456789 ()
    return self.x
  end
  assert(t:_012345678901234567890123456789012345678901234567890123456789() == 1)
end

-- test for bug in parameter adjustment
assert((function () return nil end)(4) == nil)
assert((function () local a; return a end)(4) == nil)
assert((function (a) return a end)() == nil)

return nil
`, nil)
}

func TestClosure(t *testing.T) {
	assertBlock(t, `-- closure.lua

-- Fails, equality does not work for closures unless they are references to the same underlying instance, not sure if I
-- can fix this or not.
-- testing equality
--a = {}
--for i = 1, 5 do  a[i] = function (x) return x + a + _ENV end  end
--assert(a[3] == a[4] and a[4] == a[5])
--
--for i = 1, 5 do  a[i] = function (x) return i + a + _ENV end  end
--assert(a[3] ~= a[4] and a[4] ~= a[5])
--
--local function f()
--  return function (x)  return math.sin(_ENV[x])  end
--end
--assert(f() == f())


-- testing closures with 'for' control variable
a = {}
for i=1,10 do
  a[i] = {set = function(x) i=x end, get = function () return i end}
  if i == 3 then break end
end
assert(a[4] == nil)
a[1].set(10)
assert(a[2].get() == 2)
a[2].set('a')
assert(a[3].get() == 3)
assert(a[2].get() == 'a')

a = {}
local t = {"a", "b"}
for i = 1, #t do
  local k = t[i]
  a[i] = {set = function(x, y) i=x; k=y end,
          get = function () return i, k end}
  if i == 2 then break end
end
a[1].set(10, 20)
local r,s = a[2].get()
assert(r == 2 and s == 'b')
r,s = a[1].get()
assert(r == 10 and s == 20)
a[2].set('a', 'b')
r,s = a[2].get()
assert(r == "a" and s == "b")


-- testing closures with 'for' control variable x break
for i=1,3 do
  f = function () return i end
  break
end
assert(f() == 1)

for k = 1, #t do
  local v = t[k]
  f = function () return k, v end
  break
end
assert(({f()})[1] == 1)
assert(({f()})[2] == "a")


-- testing closure x break x return x errors

local b
function f(x)
  local first = 1
  while 1 do
    if x == 3 and not first then return end
    local a = 'xuxu'
    b = function (op, y)
          if op == 'set' then
            a = x+y
          else
            return a
          end
        end
    if x == 1 then do break end
    elseif x == 2 then return
    else if x ~= 3 then error() end
    end
    first = nil
  end
end

for i=1,3 do
  f(i)
  assert(b('get') == 'xuxu')
  b('set', 10); assert(b('get') == 10+i)
  b = nil
end

pcall(f, 4);
assert(b('get') == 'xuxu')
b('set', 10); assert(b('get') == 14)


local w
-- testing multi-level closure
function f(x)
  return function (y)
    return function (z) return w+x+y+z end
  end
end

y = f(10)
w = 1.345
assert(y(20)(30) == 60+w)

-- testing closures x repeat-until

local a = {}
local i = 1
repeat
  local x = i
  a[i] = function () i = x+1; return x end
until i > 10 or a[i]() ~= x
assert(i == 11 and a[1]() == 1 and a[3]() == 3 and i == 4)


-- This next block runs afoul of the compiler, specifically the code that detects jumps into a local var's scope flags
-- this as invalid.
-- 
-- I tend to agree with it, and so does the Lua spec:
-- §3.3.4: "A goto may jump to any visible label as long as it does not enter into the scope of a local variable."
-- Jumping from 14a to 14b is illegal (local "y" comes into scope). That jump exists in the code, but is impossible due
-- to other jumps.
-- 
-- One of three things is happening here:
-- 1. luac analyzes jumps and ignores impossible code (very unlikely)
-- 2. luac does not actually follow the spec (unlikely but possible, look at the table length operator)
-- 3. There is some special case I am missing (for example since 14b is at the end of the block it may act like it is
--    out of "y"'s scope).

-- testing closures created in 'then' and 'else' parts of 'if's
--a = {}
--for i = 1, 10 do
--  if i % 3 == 0 then
--    local y = 0
--    a[i] = function (x) local t = y; y = x; return t end
--  elseif i % 3 == 1 then
--    goto L1
--    error'not here'
--  ::L1::
--    local y = 1
--    a[i] = function (x) local t = y; y = x; return t end
--  elseif i % 3 == 2 then
--    local t
--    goto l4
--    ::l4a:: a[i] = t; goto l4b
--    error("should never be here!")
--    ::l4::
--    local y = 2
--    t = function (x) local t = y; y = x; return t end
--    goto l4a
--    error("should never be here!")
--    ::l4b::
--  end
--end
--
--for i = 1, 10 do
--  assert(a[i](i * 10) == i % 3 and a[i]() == i * 10)
--end


-- test for correctly closing upvalues in tail calls of vararg functions
local function t ()
  local function c(a,b) assert(a=="test" and b=="OK") end
  local function v(f, ...) c("test", f() ~= 1 and "FAILED" or "OK") end
  local x = 1
  return v(function() return x end)
end
t()

return nil
`, nil)
}//*/

func TestX(t *testing.T) {
	assertBlock(t, `-- .lua

`, nil)
}
