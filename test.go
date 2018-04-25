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
	--if false or false then
	--	print("ok")
	--end

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

	--local a, t = {}, {}
	--function f (x, ...) return x, {...} end
	--t.__call = f
	--setmetatable(a, t)
	--local x,y = a(table.unpack{'a', 1})
	--assert(x==a and y[1]=='a' and y[2]==1 and y[3]==nil)
	--x,y = a()
	--assert(x==a and y[1]==nil)

	--a = {}
	--rawset(a, "x", 1, 2, 3)
	--assert(a.x == 1 and rawget(a, "x", 3) == 1)

	--print([=[[]]=])

	forward = {137, 176, 178, 181, 189, 193, 196, 197, 204, 205, 210, 211, 213, 215,
 218, 218, 221, 222, 222, 223, 224, 225, 228, 229, 230, 241, 241, 242, 243, 247,
 254, 255, 256, 258, 258, 266, 278, 280, 282, 288, 296, 299, 299, 301, 303, 305,
 306, 306, 310, 310, 313, 315, 319, 320, 321, 322, 323, 327, 328, 332, 335, 340,
 343, 344, 345, 348, 349, 352, 354, 357, 358, 358, 359, 360, 362, 362, 363, 365,
 366, 368, 368, 369, 370, 371, 372, 373, 375, 376, 376, 377, 381, 381, 382, 384,
 385, 386, 387, 388, 389, 390, 391, 392, 392, 393, 395, 396, 397, 399, 399, 400,
 400, 401, 402, 403, 404, 405, 407, 408, 409, 410, 411, 412, 412, 414, 414, 415,
 415, 416, 416, 417, 418, 420, 420, 421, 421, 422, 423, 425, 426, 427, 428, 429,
 430, 431, 432, 433, 434, 435, 437, 437, 438, 438, 439, 440, 441, 442, 443, 444,
 445, 610, 623, 623, 627, 627, 628, 629, 630, 641, 653, 655, 656, 656, 656, 659,
 661, 661, 662, 664, 666, 670, 678, 682, 690, 698, 700, 702, 703, 704, 704, 712,
 716, 719, 720, 725, 726, 727, 733, 734, 735, 739, 740, 742, 743, 745, 746, 749,
 753, 754, 755, 758, 759, 760, 761, 765, 767, 768, 775, 776, 777, 778, 780, 781,
 783, 784, 784, 785, 786, 787, 790, 792, 794, 796, 796, 799, 800, 800, 800, 801,
 803, 805, 806, 808, 808, 809, 811, 814, 815, 816, 817, 819, 821, 822, 827, 828,
 829, 830, 832, 833, 835, 836, 837, 838, 839, 840, 841, 842, 843, 844, 846, 847,
 853, 854, 856, 857, 859, 860, 861, 862, 862, 863, 864, 865, 867, 868, 869, 870,
 871, 872, 873, 875, 876, 877, 879, 879, 880, 880, 881, 882, 883, 884, 885, 887,
 888, 888, 889, 890, 891, 893, 893, 894, 895, 1017, 1072, 1085, 1088, 1094,
 1102, 1117, 1121, 1121, 1124, 1125, 1129, 1135, 1136, 1139, 1141, 1142, 1145,
 1146, 1147, 1149, 1152,  1154, 1156, 5113, 5114, 5114, 5115, 5116, 5117, 5118,
 5119, 5120, 5120, 5120, 5121, 5123, 5123, 5124, 5125, 5125, 5127, 5129, 5131,
 5132, 5134, 5135, 5136, 5138, 5139, 5140, 5141, 5142, 5142, 5143, 5143, 5143,
 5144, 5145, 5146, 5147, 5148, 5149, 5150, 5151, 5152, 5153, 5153, 5154, 5154,
 5155, 5157, 5157, 5158, 5159, 5159, 5159, 5160, 5161, 5162, 5163, 5164, 5166,
 5167, 5168, 5169, 5171, 5176, 5177, 5178, 5179, 5180, 5181, 5182, 5182, 5183,
 5184, 5184, 5188, 5189, 5191, 5192, 5194, 5195, 5197, 5197, 5198, 5199, 5200,
 5201, 5202, 5203, 5204, 5207, 5208, 5209, 5210, 5210, 5212, 5215, 5215, 5216,
 5218, 5219, 5220, 5222, 5225, 5226, 5227, 5228, 5229, 5231, 5232, 5233, 5234,
 5235, 5236, 5236, 5236, 5236, 5237, 5238, 5240, 5241, 5241, 5242, 5243, 5244,
 5246, 5247, 5247, 5248, 5249, 5250, 5251, 5252, 5253, 5254, 5254, 5255, 5256,
 5256, 5257, 5259, 5260, 5261, 5262, 5262, 5263, 5264, 5266, 5266, 5267, 5268,
 5269, 5273, 5274, 5275, 5276, 5277, 5277, 5278, 5279, 5279, 5281, 5283, 5283,
 5284, 5286, 5288, 5289, 5291, 5292, 5294, 5295, 5298, 5300, 5302, 5303, 5304,
 5305, 5306, 5307, 5307, 5309, 5311, 5312, 5313, 5318, 5319, 5321, 5322, 5322,
 5323, 5326, 5327, 5328, 5329, 5332, 5334, 5339, 5340, 5344, 5345, 5347, 5348,
 5350, 5351, 5352, 5353, 5353, 5354, 5355, 5356, 5357, 5358, 5359, 5361, 5362,
 5363, 5364, 5365, 5367, 5371, 5371, 5372, 5372, 5379, 5380, 5381, 5382, 5387,
 5387, 5389, 5392, 5393, 5402, 5405, 5406, 5406, 5416, 5417, 5418, 5420, 5423,
 5423, 5424, 5425, 5426, 5427, 5429, 5435, 5436, 5437, 5437, 5439, 5451, 5452,
 5453, 5474, 5475, 5476, 5487, 5488, 5489, 5490, 5491}

--print(#forward)
reverse = {}
for i,v in ipairs(forward) do
  reverse[v] = i
end
	`

	run := true
	list := true
	luac := true

	// Easier to uncomment than change the value
	//run = false
	list = false
	luac = false

	l.NativeTrace = true

	var err error

	if luac {
		l.Println("Build luac:")
		err = l.LoadTextExternal(strings.NewReader(script), "test.go", 0)
		if err != nil {

			fmt.Println(err)
			return
		} else {
			l.Println("Build OK.")
		}

		if list {
			l.ListFunc(-1)
		}

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

	l.Println("\nBuild github.com/milochristiansen/lua:")
	err = l.LoadText(strings.NewReader(script), "test.go", 0)
	if err != nil {
		fmt.Println(err)
		return
	} else {
		l.Println("Build OK.")
	}

	if list {
		l.ListFunc(-1)
	}

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
