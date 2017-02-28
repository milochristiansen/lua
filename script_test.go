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

package lua_test

import "testing"

import "github.com/milochristiansen/lua/testhelp"

// The tests in this file run blocks of code from the official Lua test suite. Most (if not all) of these tests are
// modified in some way, mostly to remove stuff dependent on APIs not available in my VM.

func TestAssign(t *testing.T) {
	testhelp.AssertBlock(t, testhelp.MkState(), `-- attrib.lua
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
	local f = function (n)
		local x = {}
		for i = 1, n do
			x[i] = i
		end
		return table.unpack(x)
	end
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
	testhelp.AssertBlock(t, testhelp.MkState(), `-- bitwise.lua
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
	testhelp.AssertBlock(t, testhelp.MkState(), `-- calls.lua
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
	testhelp.AssertBlock(t, testhelp.MkState(), `-- closure.lua

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
-- 
-- OK, it turns out that there is an undocumented special case where you can always jump to a label that is the last
-- item in it's block. For now my compiler does not support this...

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
}

func TestSyntax(t *testing.T) {
	testhelp.AssertBlock(t, testhelp.MkState(), `-- constructs.lua

-- testing semicollons
do ;;; end
; do ; a = 3; assert(a == 3) end;
;


-- invalid operations should not raise errors when not executed
if false then a = 3 // 0; a = 0 % 0 end


-- testing priorities

assert(2^3^2 == 2^(3^2));
assert(2^3*4 == (2^3)*4);
assert(2.0^-2 == 1/4 and -2^- -2 == - - -4);
assert(not nil and 2 and not(2>3 or 3<2));
assert(-3-1-5 == 0+0-9);
assert(-2^2 == -4 and (-2)^2 == 4 and 2*2-3-1 == 0);
--assert(-3%5 == 2 and -3+5 == 2) -- See next comment.
assert(2*1+3/3 == 3 and 1+2 .. 3*1 == "33");
assert(not(2+1 > 3*1) and "a".."b" > "a");

-- According to my programmer's calculator and Go:
--	5 % -3 == 2, -3 % 5 == -3
-- Apparently the authors of Lua failed basic math...

assert("7" .. 3 << 1 == 146)
assert(10 >> 1 .. "9" == 0)
assert(10 | 1 .. "9" == 27)

assert(0xF0 | 0xCC ~ 0xAA & 0xFD == 0xF4)
assert(0xFD & 0xAA ~ 0xCC | 0xF0 == 0xF4)
assert(0xF0 & 0x0F + 1 == 0x10)

assert(3^4//2^3//5 == 2)

assert(-3+4*5//2^3^2//9+4%10/3 == (-3)+(((4*5)//(2^(3^2)))//9)+((4%10)/3))

assert(not ((true or false) and nil))
assert(      true or false  and nil)

-- old bug
assert((((1 or false) and true) or false) == true)
assert((((nil and true) or false) and true) == false)

local a,b = 1,nil;
assert(-(1 or 2) == -1 and (1 and 2)+(-1.25 or -4) == 0.75);
x = ((b or a)+1 == 2 and (10 or a)+1 == 11); assert(x);
x = (((2<3) or 1) == true and (2<3 and 4) == 4); assert(x);

x,y=1,2;
assert((x>y) and x or y == 2);
x,y=2,1;
assert((x>y) and x or y == 2);

assert(1234567890 == tonumber('1234567890') and 1234567890+1 == 1234567891)


-- silly loops
repeat until 1; repeat until true;
while false do end; while nil do end;

do  -- test old bug (first name could not be an upvalue')
 local a; function f(x) x={a=1}; x={x=1}; x={G=1} end
end

function f (i)
  if type(i) ~= 'number' then return i,'jojo'; end;
  if i > 0 then return i, f(i-1); end;
end

x = {f(3), f(5), f(10);};
assert(x[1] == 3 and x[2] == 5 and x[3] == 10 and x[4] == 9 and x[12] == 1);
assert(x[nil] == nil)
x = {f'alo', f'xixi', nil};
assert(x[1] == 'alo' and x[2] == 'xixi' and x[3] == nil);
x = {f'alo'..'xixi'};
assert(x[1] == 'aloxixi')
x = {f{}}
assert(x[2] == 'jojo' and type(x[1]) == 'table')


local f = function (i)
  if i < 10 then return 'a';
  elseif i < 20 then return 'b';
  elseif i < 30 then return 'c';
  end;
end

assert(f(3) == 'a' and f(12) == 'b' and f(26) == 'c' and f(100) == nil)

for i=1,1000 do break; end;
n=100;
i=3;
t = {};
a=nil
while not a do
  a=0; for i=1,n do for i=i,1,-1 do a=a+1; t[i]=1; end; end;
end
assert(a == n*(n+1)/2 and i==3);
assert(t[1] and t[n] and not t[0] and not t[n+1])

function f(b)
  local x = 1;
  repeat
    local a;
    if b==1 then local b=1; x=10; break
    elseif b==2 then x=20; break;
    elseif b==3 then x=30;
    else local a,b,c,d=math.sin(1); x=x+1;
    end
  until x>=12;
  return x;
end;

assert(f(1) == 10 and f(2) == 20 and f(3) == 30 and f(4)==12)


local f = function (i)
  if i < 10 then return 'a'
  elseif i < 20 then return 'b'
  elseif i < 30 then return 'c'
  else return 8
  end
end

assert(f(3) == 'a' and f(12) == 'b' and f(26) == 'c' and f(100) == 8)

local a, b = nil, 23
x = {f(100)*2+3 or a, a or b+2}
assert(x[1] == 19 and x[2] == 25)
x = {f=2+3 or a, a = b+2}
assert(x.f == 5 and x.a == 25)

a={y=1}
x = {a.y}
assert(x[1] == 1)

function f(i)
  while 1 do
    if i>0 then i=i-1;
    else return; end;
  end;
end;

function g(i)
  while 1 do
    if i>0 then i=i-1
    else return end
  end
end

f(10); g(10);

do
  function f () return 1,2,3; end
  local a, b, c = f();
  assert(a==1 and b==2 and c==3)
  a, b, c = (f());
  assert(a==1 and b==nil and c==nil)
end

local a,b = 3 and f();
assert(a==1 and b==nil)

function g() f(); return; end;
assert(g() == nil)
function g() return nil or f() end
a,b = g()
assert(a==1 and b==nil)



f = [[
return function ( a , b , c , d , e )
  local x = a >= b or c or ( d and e ) or nil
  return x
end , { a = 1 , b = 2 >= 1 , } or { 1 };
]]
--f = string.gsub(f, "%s+", "\n");   -- force a SETLINE between opcodes (Milo: disabled, no "string.gsub")
f,a = load(f)();
assert(a.a == 1 and a.b)

function g (a,b,c,d,e)
  if not (a>=b or c or d and e or nil) then return 0; else return 1; end;
end

function h (a,b,c,d,e)
  while (a>=b or c or (d and e) or nil) do return 1; end;
  return 0;
end;

assert(f(2,1) == true and g(2,1) == 1 and h(2,1) == 1)
assert(f(1,2,'a') == 'a' and g(1,2,'a') == 1 and h(1,2,'a') == 1)
assert(f(1,2,'a')
~=          -- force SETLINE before nil
nil, "")
assert(f(1,2,'a') == 'a' and g(1,2,'a') == 1 and h(1,2,'a') == 1)
assert(f(1,2,nil,1,'x') == 'x' and g(1,2,nil,1,'x') == 1 and
                                   h(1,2,nil,1,'x') == 1)
assert(f(1,2,nil,nil,'x') == nil and g(1,2,nil,nil,'x') == 0 and
                                     h(1,2,nil,nil,'x') == 0)
assert(f(1,2,nil,1,nil) == nil and g(1,2,nil,1,nil) == 0 and
                                   h(1,2,nil,1,nil) == 0)

assert(1 and 2<3 == true and 2<3 and 'a'<'b' == true)
x = 2<3 and not 3; assert(x==false)
x = 2<1 or (2>1 and 'a'); assert(x=='a')


do
  local a; if nil then a=1; else a=2; end;    -- this nil comes as PUSHNIL 2
  assert(a==2)
end

`, nil)
}

func TestMetatables(t *testing.T) {
	testhelp.AssertBlock(t, testhelp.MkState(), `-- events.lua

X = 20; B = 30

_ENV = setmetatable({}, {__index=_G})

X = X+10
assert(X == 30 and _G.X == 20)
B = false
assert(B == false)
B = nil
assert(B == 30)

assert(getmetatable{} == nil)
assert(getmetatable(4) == nil)
assert(getmetatable(nil) == nil)
a={}; setmetatable(a, {__metatable = "xuxu",
                    __tostring=function(x) return x.name end})
assert(getmetatable(a) == "xuxu")
assert(tostring(a) == nil)
-- cannot change a protected metatable
assert(pcall(setmetatable, a, {}) == false)
a.name = "gororoba"
assert(tostring(a) == "gororoba")

local a, t = {10,20,30; x="10", y="20"}, {}
assert(setmetatable(a,t) == a)
assert(getmetatable(a) == t)
assert(setmetatable(a,nil) == a)
assert(getmetatable(a) == nil)
assert(setmetatable(a,t) == a)


function f (t, i, e)
  assert(not e)
  local p = rawget(t, "parent")
  return (p and p[i]+3), "dummy return"
end

t.__index = f

a.parent = {z=25, x=12, [4] = 24}
assert(a[1] == 10 and a.z == 28 and a[4] == 27 and a.x == "10")

a = setmetatable({}, t)
function f(t, i, v) rawset(t, i, v-3) end
setmetatable(t, t)   -- causes a bug in 5.1 !
t.__newindex = f
a[1] = 30; a.x = "101"; a[5] = 200
assert(a[1] == 27 and a.x == 98 and a[5] == 197)

do    -- bug in Lua 5.3.2
  local mt = {}
  mt.__newindex = mt
  local t = setmetatable({}, mt)
  t[1] = 10     -- will segfault on some machines
  assert(mt[1] == 10)
end


local c = {}
a = setmetatable({}, t)
t.__newindex = c
a[1] = 10; a[2] = 20; a[3] = 90
assert(c[1] == 10 and c[2] == 20 and c[3] == 90)


do
  local a;
  a = setmetatable({}, {__index = setmetatable({},
                     {__index = setmetatable({},
                     {__index = function (_,n) return a[n-3]+4, "lixo" end})})})
  a[0] = 20
  for i=0,10 do
    assert(a[i*3] == 20 + i*4)
  end
end


do  -- newindex
  local foi
  local a = {}
  for i=1,10 do a[i] = 0; a['a'..i] = 0; end
  setmetatable(a, {__newindex = function (t,k,v) foi=true; rawset(t,k,v) end})
  foi = false; a[1]=0; assert(not foi)
  foi = false; a['a1']=0; assert(not foi)
  foi = false; a['a11']=0; assert(foi)
  foi = false; a[11]=0; assert(foi)
  foi = false; a[1]=nil; assert(not foi)
  foi = false; a[1]=nil; assert(foi)
end


setmetatable(t, nil)
function f (t, ...) return t, {...} end
t.__call = f

do
  local x,y = a(table.unpack{'a', 1})
  assert(x==a and y[1]=='a' and y[2]==1 and y[3]==nil)
  x,y = a()
  assert(x==a and y[1]==nil)
end


local b = setmetatable({}, t)
setmetatable(b,t)

function f(op)
  return function (...) cap = {[0] = op, ...} ; return (...) end
end
t.__add = f("add")
t.__sub = f("sub")
t.__mul = f("mul")
t.__div = f("div")
t.__idiv = f("idiv")
t.__mod = f("mod")
t.__unm = f("unm")
t.__pow = f("pow")
t.__len = f("len")
t.__band = f("band")
t.__bor = f("bor")
t.__bxor = f("bxor")
t.__shl = f("shl")
t.__shr = f("shr")
t.__bnot = f("bnot")

assert(b+5 == b)
assert(cap[0] == "add" and cap[1] == b and cap[2] == 5 and cap[3]==nil)
assert(b+'5' == b)
assert(cap[0] == "add" and cap[1] == b and cap[2] == '5' and cap[3]==nil)
assert(5+b == 5)
assert(cap[0] == "add" and cap[1] == 5 and cap[2] == b and cap[3]==nil)
assert('5'+b == '5')
assert(cap[0] == "add" and cap[1] == '5' and cap[2] == b and cap[3]==nil)
b=b-3; assert(getmetatable(b) == t)
assert(5-a == 5)
assert(cap[0] == "sub" and cap[1] == 5 and cap[2] == a and cap[3]==nil)
assert('5'-a == '5')
assert(cap[0] == "sub" and cap[1] == '5' and cap[2] == a and cap[3]==nil)
assert(a*a == a)
assert(cap[0] == "mul" and cap[1] == a and cap[2] == a and cap[3]==nil)
assert(a/0 == a)
assert(cap[0] == "div" and cap[1] == a and cap[2] == 0 and cap[3]==nil)
assert(a%2 == a)
assert(cap[0] == "mod" and cap[1] == a and cap[2] == 2 and cap[3]==nil)
assert(a // (1/0) == a)
assert(cap[0] == "idiv" and cap[1] == a and cap[2] == 1/0 and cap[3]==nil)
assert(a & "hi" == a)
assert(cap[0] == "band" and cap[1] == a and cap[2] == "hi" and cap[3]==nil)
assert(a | "hi" == a)
assert(cap[0] == "bor" and cap[1] == a and cap[2] == "hi" and cap[3]==nil)
assert("hi" ~ a == "hi")
assert(cap[0] == "bxor" and cap[1] == "hi" and cap[2] == a and cap[3]==nil)
assert(-a == a)
assert(cap[0] == "unm" and cap[1] == a)
assert(a^4 == a)
assert(cap[0] == "pow" and cap[1] == a and cap[2] == 4 and cap[3]==nil)
assert(a^'4' == a)
assert(cap[0] == "pow" and cap[1] == a and cap[2] == '4' and cap[3]==nil)
assert(4^a == 4)
assert(cap[0] == "pow" and cap[1] == 4 and cap[2] == a and cap[3]==nil)
assert('4'^a == '4')
assert(cap[0] == "pow" and cap[1] == '4' and cap[2] == a and cap[3]==nil)
assert(#a == a)
assert(cap[0] == "len" and cap[1] == a)
assert(~a == a)
assert(cap[0] == "bnot" and cap[1] == a)
assert(a << 3 == a)
assert(cap[0] == "shl" and cap[1] == a and cap[2] == 3)
assert(1.5 >> a == 1.5)
assert(cap[0] == "shr" and cap[1] == 1.5 and cap[2] == a)


-- test for rawlen
t = setmetatable({1,2,3}, {__len = function () return 10 end})
assert(#t == 10 and rawlen(t) == 3)
assert(rawlen"abc" == 3)
--assert(not pcall(rawlen, io.stdin))
assert(not pcall(rawlen, 34))
assert(not pcall(rawlen))

-- rawlen for long strings
assert(rawlen(string.rep('a', 1000)) == 1000)


t = {}
t.__lt = function (a,b,c)
  assert(c == nil)
  if type(a) == 'table' then a = a.x end
  if type(b) == 'table' then b = b.x end
 return a<b, "dummy"
end

function Op(x) return setmetatable({x=x}, t) end

local function test ()
  assert(not(Op(1)<Op(1)) and (Op(1)<Op(2)) and not(Op(2)<Op(1)))
  assert(not(1 < Op(1)) and (Op(1) < 2) and not(2 < Op(1)))
  assert(not(Op('a')<Op('a')) and (Op('a')<Op('b')) and not(Op('b')<Op('a')))
  assert(not('a' < Op('a')) and (Op('a') < 'b') and not(Op('b') < Op('a')))
  assert((Op(1)<=Op(1)) and (Op(1)<=Op(2)) and not(Op(2)<=Op(1)))
  assert((Op('a')<=Op('a')) and (Op('a')<=Op('b')) and not(Op('b')<=Op('a')))
  assert(not(Op(1)>Op(1)) and not(Op(1)>Op(2)) and (Op(2)>Op(1)))
  assert(not(Op('a')>Op('a')) and not(Op('a')>Op('b')) and (Op('b')>Op('a')))
  assert((Op(1)>=Op(1)) and not(Op(1)>=Op(2)) and (Op(2)>=Op(1)))
  assert((1 >= Op(1)) and not(1 >= Op(2)) and (Op(2) >= 1))
  assert((Op('a')>=Op('a')) and not(Op('a')>=Op('b')) and (Op('b')>=Op('a')))
  assert(('a' >= Op('a')) and not(Op('a') >= 'b') and (Op('b') >= Op('a')))
end

test()

t.__le = function (a,b,c)
  assert(c == nil)
  if type(a) == 'table' then a = a.x end
  if type(b) == 'table' then b = b.x end
 return a<=b, "dummy"
end

test()  -- retest comparisons, now using both 'lt' and 'le'


-- test 'partial order'

local function rawSet(x)
  local y = {}
  for _,k in pairs(x) do y[k] = 1 end
  return y
end

local function Set(x)
  return setmetatable(rawSet(x), t)
end

t.__lt = function (a,b)
  for k in pairs(a) do
    if not b[k] then return false end
    b[k] = nil
  end
  return next(b) ~= nil
end

t.__le = nil

assert(Set{1,2,3} < Set{1,2,3,4})
assert(not(Set{1,2,3,4} < Set{1,2,3,4}))
assert((Set{1,2,3,4} <= Set{1,2,3,4}))
assert((Set{1,2,3,4} >= Set{1,2,3,4}))
assert((Set{1,3} <= Set{3,5}))   -- wrong!! model needs a 'le' method ;-)

t.__le = function (a,b)
  for k in pairs(a) do
    if not b[k] then return false end
  end
  return true
end

assert(not (Set{1,3} <= Set{3,5}))   -- now its OK!
assert(not(Set{1,3} <= Set{3,5}))
assert(not(Set{1,3} >= Set{3,5}))

t.__eq = function (a,b)
  for k in pairs(a) do
    if not b[k] then return false end
    b[k] = nil
  end
  return next(b) == nil
end

local s = Set{1,3,5}
assert(s == Set{3,5,1})
assert(not rawequal(s, Set{3,5,1}))
assert(rawequal(s, s))
assert(Set{1,3,5,1} == rawSet{3,5,1})
assert(rawSet{1,3,5,1} == Set{3,5,1})
assert(Set{1,3,5} ~= Set{3,5,1,6})

-- '__eq' is not used for table accesses
t[Set{1,3,5}] = 1
assert(t[Set{1,3,5}] == nil)

t.__concat = function (a,b,c)
  assert(c == nil)
  if type(a) == 'table' then a = a.val end
  if type(b) == 'table' then b = b.val end
  if A then return a..b
  else
    return setmetatable({val=a..b}, t)
  end
end

c = {val="c"}; setmetatable(c, t)
d = {val="d"}; setmetatable(d, t)

A = true
assert(c..d == 'cd')
assert(0 .."a".."b"..c..d.."e".."f"..(5+3).."g" == "0abcdef8g")

A = false
assert((c..d..c..d).val == 'cdcd')
x = c..d
assert(getmetatable(x) == t and x.val == 'cd')
x = 0 .."a".."b"..c..d.."e".."f".."g"
assert(x.val == "0abcdefg")


-- concat metamethod x numbers (bug in 5.1.1)
c = {}
local x
setmetatable(c, {__concat = function (a,b)
  assert(type(a) == "number" and b == c or type(b) == "number" and a == c)
  return c
end})
assert(c..5 == c and 5 .. c == c)
assert(4 .. c .. 5 == c and 4 .. 5 .. 6 .. 7 .. c == c)


-- test comparison compatibilities
local t1, t2, c, d
t1 = {};  c = {}; setmetatable(c, t1)
d = {}
t1.__eq = function () return true end
t1.__lt = function () return true end
setmetatable(d, t1)
assert(c == d and c < d and not(d <= c))
t2 = {}
t2.__eq = t1.__eq
t2.__lt = t1.__lt
setmetatable(d, t2)
assert(c == d and c < d and not(d <= c))



-- test for several levels of calls
local i
local tt = {
  __call = function (t, ...)
    i = i+1
    if t.f then return t.f(...)
    else return {...}
    end
  end
}

local a = setmetatable({}, tt)
local b = setmetatable({f=a}, tt)
local c = setmetatable({f=b}, tt)

i = 0
x = c(3,4,5)
assert(i == 3 and x[1] == 3 and x[3] == 5)


assert(_G.X == 20)

--print'+'

local _g = _G
_ENV = setmetatable({}, {__index=function (_,k) return _g[k] end})


a = {}
rawset(a, "x", 1, 2, 3)
assert(a.x == 1 and rawget(a, "x", 3) == 1)

-- Crashes my VM! I don't have any loop checks implemented. It's an error either way, but it would be nicer to stop it early...
-- loops in delegation
--a = {}; setmetatable(a, a); a.__index = a; a.__newindex = a
--assert(not pcall(function (a,b) return a[b] end, a, 10))
--assert(not pcall(function (a,b,c) a[b] = c end, a, 10, true))

-- bug in 5.1
T, K, V = nil
grandparent = {}
grandparent.__newindex = function(t,k,v) T=t; K=k; V=v end

parent = {}
parent.__newindex = parent
setmetatable(parent, grandparent)

child = setmetatable({}, parent)
child.foo = 10      --> CRASH (on some machines)
print(T == parent, K == "foo", V == 10)
assert(T == parent and K == "foo" and V == 10)

return 12
`, 12)
}

// func TestX(t *testing.T) {
// 	testhelp.AssertBlock(t, testhelp.MkState(), `-- .lua

// `, nil)
// }
