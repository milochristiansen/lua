
DCLua - Go Lua Compiler and VM:
========================================================================================================================

This is a Lua 5.3 VM and compiler written in [Go](http://golang.org/). This is intended to allow easy embedding into Go
programs, with minimal fuss and bother.

I have been using this VM/compiler as the primary script host in Rubble (a scripted templating system used to generate
data files for the game Dwarf Fortress) for over a year now, so they are fairly well tested. In addition to the real-world
"testing" that this has received I am slowly adding proper tests based on the official Lua test suite. These tests are
far from complete, but are slowly getting more so as time passes.

Most (if not all) of the API functions may cause a panic, but only if things go REALLY wrong. If a function does not
state that it can panic or "raise an error" it will only do so if a critical internal assumption proves to be wrong
(AKA there is a bug in the code somewhere). These errors will have a special prefix prepended onto the error message
stating that this error indicates an internal VM bug. If you ever see such an error I want to know about it ASAP.

That said, if an API function *can* "raise an error" it can and will panic if something goes wrong. This is not a
problem inside a native function (as the VM is prepared for this), but if you need to call these functions outside of
code to be run by the VM you may want to use Protect or Recover to properly catch these errors.
 
The VM itself does not provide any Lua functions, the standard library is provided entirely by other packages. This
means that the standard library never does anything that your own code cannot do (there is no "private API" that is used
by the standard library). 

Anything to do with the OS or file IO is not provided. Such things do not belong in the core libraries of an embedded
scripting language (do you really want scripts to be able to read and write random files without restriction?).

All functions (including most of the internal functions) are documented to one degree or another, most quite well. The
API is designed to be easy to use, and everything was added because I needed it. There are no "bloat" functions added
because I thought they could be useful.

Note that another version of this exists over at [ofunc/lua](https://github.com/ofunc/lua). That version has some
interesting changes/features, I suggest you give it look to see if it suits your needs better.


Loading Code:
------------------------------------------------------------------------------------------------------------------------

This VM fully supports binary chunks, so if you want to precompile your script it is possible. To precompile a script
for use with this VM you can either build a copy of `luac` (the reference Lua compiler) or use any other third party Lua
complier provided that it generates code compatible with the reference compiler. There is no separate compiler binary
that you can build, but it wouldn't be hard to write one. Note that the VM does not handle certain instructions in pairs
like the reference Lua VM does, and I don't remember if I made the compiler take advantage of this or not. If I did then
binaries generated by my compiler may not work with the reference VM.

If you want to use a third-party compiler it will need to produce binaries with the following settings:

* 64 *or* 32 bit pointers (C type `size_t`), 64 bit preferred.
* 32 bit integers (C type `int`).
* 64 bit float numbers.
* 64 bit integer numbers.
* Little Endian byte order.

When building the reference compiler on most systems these settings should be the default.

The VM API has a function that wraps `luac` to load code, but the way it does this may or may not fit your needs. To use
this wrapper you will need to have `luac` on your path or otherwise placed so the VM can find it. See the documentation
for `State.LoadTextExternal` for more information. Keep in mind that due to limitations in Go and `luac`, this function
is not reentrant! If you need concurrency support it would be better to use `State.LoadBinary` and write your own wrapper.

The default compiler provided by this library does not support constant folding, and some special instructions are not
used at all (instead preferring simpler sequences of other instructions). Expressions use a simple "recursive" code
generation style, meaning that it wastes registers like crazy in some (rare) cases.

One of the biggest code quality offenders is `or` and `and`, as they can result in sequences like this one:

	[4]   LT        A:1  B:r(0)   C:k(2)  ; CK:5
	[5]   JMP       A:0  SBX:1            ; to:7
	[6]   LOADBOOL  A:2  B:1      C:1
	[7]   LOADBOOL  A:2  B:0      C:0
	[8]   TEST      A:2           C:1
	[9]   JMP       A:0  SBX:7            ; to:17
	[10]  EQ        A:1  B:r(1)   C:k(3)  ; CK:<nil>
	... (7 more instructions to implement next part of condition)

As you can see this is terrible. That sequence would be better written as:

	[4]   LT        A:1  B:r(0)   C:k(2)  ; CK:5
	[5]   JMP       A:0  SBX:2            ; to:8
	[6]   EQ        A:1  B:r(1)   C:k(3)  ; CK:<nil>
	... (1 more instruction to implement next part of condition)

But the current expression compiler is not smart enough to do it that way. Luckily this is the worst offender, most
things produce code that is very close or identical to what `luac` produces. Note that the reason why this code is so
bad is entirely because the expression used `or` (and the implementation of `and` and `or` is very bad).

To my knowledge there is only one case where my compiler does a better job than `luac`, namely when compiling loops or
conditionals with constant conditions, impossible conditions are elided (so if you say `while false do x(y z) end` the
compiler will do nothing). AFAIK there is no way to jump into such blocks anyway, so eliding them should have no effect
on the correctness of the program.

The compiler provides an implementation of a `continue` keyword, but the keyword definition in the lexer is commented
out. If you want `continue` all you need to do is uncomment the indicated line (near the top of `ast/lexer.go`). There
is also a flag in the VM that *should* make tables use 0 based indexing. This feature has received minimal testing, so
it probably doesn't work properly. If you want to try 0 based indexing just set the variable `TableIndexOffset` to 0.
Note that `TableIndexOffset` is strictly a VM setting, the standard modules do not respect this setting (for example the
`table` module and `ipairs` will still insist on using 1 as the first index).


Missing Stuff:
------------------------------------------------------------------------------------------------------------------------

The following standard functions/variables are not available:

* `collectgarbage` (not possible, VM uses the Go collector)
* `dofile` (violates my security policy)
* `loadfile` (violates my security policy)
* `xpcall` (VM has no concept of a message handler)
* `package.config` (violates my security policy)
* `package.cpath` (VM has no support for native modules)
* `package.loadlib` (VM has no support for native modules)
* `package.path` (violates my security policy)
* `package.searchpath` (violates my security policy)
* `string.gmatch` (No pattern matching support)
* `string.gsub` (No pattern matching support)
* `string.match` (No pattern matching support)
* `string.pack` (too lazy to implement, ask if you need it)
* `string.packsize` (too lazy to implement, ask if you need it)
* `string.unpack` (too lazy to implement, ask if you need it)


* * *

The following standard modules are not available:

* `coroutine` (no coroutine support yet, ask if you need it)
* `io` (violates my security policy)
* `os` (violates my security policy)
* `debug` (violates my security policy, if you really need something from here ask)

Coroutine support is not available. I can implement something based on goroutines fairly easily, but I will only do so
if someone actually needs it and/or if I get really bored...


* * *

In addition to the stuff that is not available at all the following functions are not implemented exactly as the Lua
5.3 specification requires:

* `string.find` does not allow pattern matching yet (the fourth option is effectively always set to `true`).
* Only one searcher is added to `package.searchers`, the one for finding modules in `package.preloaded`.
* `next` is not reentrant for a single table, as it needs to store state information about each table it is used to iterate.
  Starting a new iteration for a particular table invalidates the state information for the previous iteration of
  that table. *Never* use this function for iterating a table unless you absolutely *have* to, use the non-standard
  `getiter` function instead. `getiter` works the way `next` should have, namely it uses a single iterator value that
  stores all required iteration state internally (the way the default `next` works is only possible if your hash table
  is implemented a certain way).

Finally there are a few things that are implemented exactly as the Lua 5.3 specification requires, where the reference
Lua implementation does not follow the specification exactly:

* The `#` (length) operator always returns the exact length of a (table) sequence, not the total length of the array
  portion of the table. See the comment in `table.go` (about halfway down) for more details (including quotes from the
  spec and examples).
* My modulo operator (`%`) is implemented the same way most languages implement it, not the way Lua does. This does not
  matter unless you are using negative operands, in which case it may not provide the results a Lua programmer may expect
  (although C or Go programmers will be fine :P).


* * *

The following *core language* features are not supported:

* Hexadecimal floating point literals are not supported at this time. This "feature" is not supported for two reasons:
  I hate floating point in general (so trying to write a converter is pure torture), and when have you *ever* used 
  hexadecimal floating point literals? Lua is the only language I have ever used that supports them, so they are not
  exactly popular...
* Weak references of any kind are not supported. This is because I use Go's garbage collector, and it does not support
  weak references.
* I do not currently support finalizers. It would probably be possible to support them, but it would be a lot of work
  for a feature that is of limited use (I have only ever needed to use a finalizer once, ironically in this library).
  If you have a compelling reason why you need finalizers I could probably add them...
* The reference compiler allows you to use `goto` to jump to a label at the end of a block ignoring any variables in said
  block. For example:
  
		do
			goto x
			local a
			::x::
		end
  
  My compiler does not currently allow this, treating it as a jump into the scope of a local variable. I consider this a
  bug, and will probably fix it sooner or later...

  Note that AFAIK there is nothing in the Lua spec that implies this is allowed, but it seems like a logical thing to
  permit so I suppose I'll have to fix it, *sigh*.


TODO:
------------------------------------------------------------------------------------------------------------------------

Stuff that should be done sometime. Feel free to help out :)

The list is (roughly) in priority order.

* Write more tests for the compiler and VM.
* (supermeta) Allow using byte slices as strings and vice-versa. Maybe attach a method to byte slices that allows conversion
  back and forth? (this would probably be fairly easy to do)
  * Do the same with rune slices?
* Write better stack traces for errors.
* Improve compilation of `and` and `or`.
* Fix jumping to a label at the end of a block.
* Fix `CONCAT` so it performs better when there is a value with a `__concat` metamethod.
* (supermeta) Look into allowing scripts to call functions/methods. It's certainly possible, but possibly difficult
  (possible not as difficult as I think).
* Marshaling the AST as XML works poorly at best. The main problem is that some items retain their type info, and others
  have it stripped in favor of their parent field name.


Changes:
------------------------------------------------------------------------------------------------------------------------

A note on versions:

For this project I more-or-less follow semantic versioning, so I try to maintain backwards compatibility across point
releases. That said I feel free to break minor things in the name of bugfixes. Read the changelog before upgrading!


* * *

1.1.8

* Fixed `State.ConvertString` so it actually works (based on PR #21 by ofunc)
* Fixed a really weird issue where statements that started with a parenthesized expression would be assumed to be a function
  call and error out if they were not. (Fixed #23)
* Fixed issue in `string.byte`. (PR #24 by ofunc)
* Fixed `math.huge` to have the correct value. (PR #25 by ofunc)
* Added `utf8` package (`github.com/milochristiansen/lua/lmodutf8`) (PR #26 by ofunc)

* * *

1.1.7

* Function calls or parenthesized expressions that are followed by table indexers are now properly compiled (Fixed #13).
* The compiler sometimes did not always mark "used" the proper number of registers when compiling identifiers (Fixed #16).
* Fixed the table iterator not finalizing (Fixed #17).
* Removed my hacky slice library and just did things properly (Fixed #18).
* Fix `pcall` not returning `true` on success (Fixed #19).
* Fixed setting a nil index in a table not raising an error (Fixed #20).

* * *

1.1.6

Fun with tables! Ok, not so much fun.

* Fixed scripts with lots of constants overflowing RK fields in certain instructions. The proper constant load instructions
  are emitted in this case now.
* Tables with lots of empty space at the beginning of the array portion will no longer cause crashes when the array portion
  is resized.

* * *

1.1.5

And, another stupid little bug.

* Constructs similar to the following `[=[]==]]=]` were not working properly. The lexer was not properly constructing the
  lexeme, and it would return the wrong number of equals signs and it would eat the last square bracket. As a bonus I
  greatly simplified the string lexing code. (ast/lexer.go)

* * *

1.1.4

Not sure how I missed this one... Oh well, it should work now.

* `require` was not checking `package.loaded` properly. (lmodpackage/functions.go)

* * *

1.1.3

One of the tests was failing on 32 bit systems, now it isn't.

* Integer table keys that fit into a script integer but not a system default int value will no longer be truncated sometimes.
  Such keys were always supposed to go in the hash part of the table, but before this fix the keys were being truncated first
  in some cases. (table.go)

* * *

1.1.2

More script tests, but no real compiler bugs this time. Instead I found several minor issues with a few of the API functions
and a few other miscellaneous VM issues (mostly related to metatables).

This version also adds a minor new feature, nothing to get excited about... Basically I made it so that JSON or XML encoding
an AST produces slightly more readable results for operator expression nodes. Someone else suggested the idea (actually they
submitted a patch, yay them!). I never would have thought to do this myself (never needed it), but now that I have it, it
seems like it could be useful for debugging the compiler among other things.

Unfortunately due to the way the AST and most encodings work, it is impossible to unmarshal the AST. I am not 100% sure if
it is possible with XML or not, but it certainly will not work with JSON. This could maybe be fixed, but would be way too
much work.

Anyway, these improvements are still useful if you want to examine the AST for whatever reason...


* Added another set of script tests. (script_test.go)
* Fixed the `tostring` script function and the `ConvertString` API function so they pass the return value from a
  `_tostring` metamethod through unchanged (instead of converting the result to a string, for example `"nil"`). (api.go)
* You may now use `nil` as a metatable value for the `SetMetaTable` API function (and the `setmetatable` script
  function). (api.go)
* Made some changes to the compiler tester so it is easier to tweak the output for specific error types (for example it is
  now possible to suppress the assembly listing). (test.go)
* Fixed variadic functions that also have named parameters (this was an issue with the new stack frame code, not the compiler).
  (stack.go)
* Fixed comparisons of non-matching types via metamethods, before if the types did not match the comparison would always fail
  (I must have been sleep deprived when I wrote that bit). (value.go)
* Changed the way greater-than and greater-than or equals (`>` and `>=`) where implemented. The old way was `2 > 1 == !(2 <= 1)`
  and `2 >= 1 == !(2 < 1)`, the new way is `2 > 1 == 1 < 2` and `2 >= 1 == 1 <= 2`. The old way worked fine (and was slightly
  easier to implement), but it was not what the spec required (and so could cause problems with metamethods). (compile_expr.go)
* Changed the way the `CONCAT` instruction is implemented so that the `__concat` metamethod works correctly. **Warning:** using
  `__concat` even once will cause string concatenation performance to nosedive for that group of concatenation operations! (vm.go)
* Fixed the `rawset` and `rawget` script functions so they ignore extra arguments. (lmodbase/functions.go)
* Values with a `__newindex` metamethod that is a table now properly use a regular set (triggers metamethods) when indexing this
  table. I'm not sure how I missed this, I did it properly for gets (`__index`). (value.go)
* AST operator type constants now marshal and unmarshal as text rather than raw (more-or-less meaningless) integers. Also
  when printed via `fmt` functions they should use text names by default. (ast/expr.go, commit by "erizocosmico")
* When encoded as JSON, each Node now has an extra key named after the node type that contains an object with all fields
  common to all nodes (namely the line number). This greatly enhances readability of a JSON encoded AST. (ast/ast.go)


* * *

1.1.1

More script tests, more compiler bugs fixed. Same song, different verse.

* Added another set of script tests. (script_test.go)
* Fixed unary operators after a power operator, for example `2 ^ - -2`. To fix this issue I totally rewrote how operators
  are parsed. (ast/parse_expr.go)
* Fixed semicolons immediately after a return statement. (ast/parse.go)
* Fixed an improper optimization or repeat-until loops. Basically if the loop had a constant for the loop condition its
  sense was being reversed (so a false condition resulted in the loop being compiled as a simple block, and a true condition
  resulted in an infinite loop). (compile.go)
* Fixed `and` in non-boolean contexts. Also `and` and `or` *may* produce slightly better code now. (compile_expr.go)


* * *

1.1.0

I was a little bored recently, so I threw together a generic metatable API. It was a quick little project, based on
earlier work for one of my many toy languages. This new API is kinda cool, but it in no way replaces proper metatables!
Basically it is intended for quick projects and temporarily exposing data to scripts. It was fun to write, and so even
if no one uses it, it has served its purpose :P

I really should have been working on more script tests, but this was more fun... I have no doubt responsibility will
reassert itself soon.

Anyway, I also added two new convenience methods for table iteration, as well as some minor changes to the old one (you
can still use it, but it is now a thin wrapper over one of the new functions, so you shouldn't).

* Ran all code through `go fmt`. I often forget to do this, but I recently switched to a new editor that formats files
  automatically whenever they are saved. Anyway, everything is formatted now. (almost every file in minor ways)
* Added `Protect` and `Recover`, simple error handlers for native code. They are to be used when calling native APIs
  outside of code otherwise protected (such as by a call to PCall). `Recover` is the old handler from `PCall`, wrapped
  so it can be used by itself. `Protect` simply wraps `Recover` so it is easier to use. (api.go)
* Added `ForEachRaw`, basically `ForEachInTable`, but the passed in function returns a boolean specifying if you want to
  break out of the loop early. In other news `ForEachInTable` is now depreciated. (api.go)
* Added `ForEach`, a version of `ForEachRaw` that respects the `__pairs` metamethod. `ForEachRaw` uses the table iterator
  directly and does much less stack manipulation, so it is probably a little faster. (api.go)
* Added a new sub-package: `supermeta` adds "generic" metatables for just about any Go type. For obvious reasons this
  makes heavy use of reflection, so it is generally much faster to write your own metatables, that said this is really
  nice for quickly exposing native data to scripts. From the user's perspective you just call `supermeta.New(l, &object)`
  and `object` is suddenly a script value on the top of `l`'s stack. Arrays, slices, maps, structs, etc should all work
  just fine. Note that this is very new, and as of yet has received little real-world testing! (supermeta/supermeta.go,
  supermeta/tables.go)
* Added a new sub-package: `testhelp` contains a few test helper functions I find useful when writing tests that interact
  with the VM. Better to have all this stuff in one place rather than copied and pasted all over... (testhelp/testhelp.go)
* Modified the script tests in the base package to use the helper functions in `testhelp` rather than their own copies.
  The API tests still have their own copies of some of the functions, as they need to be in the base package so they can
  access internal APIs (stupid circular imports). (script_test.go)
* Clarified what API functions may panic, I think I got them all... (api.go) 


* * *

1.0.2

More tests, more (compiler) bugs fixed. Damn compiler will be the death of me yet...

In addition to the inevitable compiler bugs I also fixed the way the VM handles upvalues. Before I was giving each
closure its own copy of each upvalue, so multiple closures never properly shared values. This change fixes several
subtle (and several not so subtle) bugs.

Oh, and `pcall` works now (it didn't work at all before. Sorry, I never used it).

* Added more script tests. I still have a lot more to do... (script_test.go)
* Fixed incorrect compilation of method declarations (`function a:x() end`). Depressingly the issue was only one
  incorrect word, but it resulted in *very* wrong results (I am really starting to remember why I hated writing the
  compiler, the VM was fun, the compiler... not.) (ast/parse.go)
* Parenthesized expression that would normally (without the parenthesis) return multiple values (for example: `(...)`)
  were not properly truncating the result to a single value. (compile_expr.go)
* Fixed a semi-major VM issue with upvalues. Closures that should have a single shared upvalue were instead each using
  their own private copy after said upvalue was closed. This required an almost total rewrite of the way upvalues are
  stored internally. (all over the place, but mainly callframe.go, function.go, api.go, and vm.go)
* JMP instructions created by `break` and `continue` statements are now properly patched by the compiler to close any
  upvalues there may be. (compile.go)
* Fixed the `pcall` script function so it actually works. (lmodbase/functions.go)
* On a recovered error each stack frame's upvalues are closed before the stack is stripped. This corrects incorrect
  behavior that arises when a function stores a closure to an unclosed upvalue then errors out (the closure may still be
  referenced, but it's upvalues may be invalid). (api.go, callframe.go)


* * *

1.0.1

This version adds a bunch of tests (still not nearly as many as I would like), and fixes a ton of minor compiler errors.
Most of the compiler errors were simple oversights, usually syntax constructs that I never used in my own code (and hence
never tested).

The VM itself seems to be mostly bug free, but the compiler is a different story. I'm fixing bugs as fast as I discover
them, but sometimes it's really tempting to just use `luac` and call it a day :P

* Fixed a issue with State.Pop possibly causing a panic if you pop values when the stack is empty (or if you try to pop
  more values than the stack contains), it now does nothing in this case. (stack.go)
* Added some tests for the VM native API (api_test.go)
* Added some script tests based on the official Lua 5.3 test suite. These tests are not (even close to) complete yet,
  (many) more are on the way. (script_test.go)
* Added a `String` method to `STypeID` to match the one for `TypeID`. (value.go)
* Made the custom `string` module extensions optional. (lmodstring/functions.go, lmodstring/README.md)
* Fixed an issue with the `ForEachInTable` helper function, it left the table iterator object on the stack when it
  returned. (api.go)
* Fixed inexplicably missing lexer entry for the semicolon (I know it was there before, it must have gotten removed by
  accident at some point). (ast/lexer.go)
* Lexer errors now contain the line number where the problem resides (or at least close to it). (ast/parse.go)
* Fixed that numeric for loops required all three arguments. I always use the full form, so I forgot that a short two
  argument form is legal... (ast/parse.go)
* Fixed that you could not repeat two unary operators in a row. (ast/parse_expr.go)
* You may now use semicolons as well as commas as field separators in table constructors (did you know that was legal? I
  didn't until I rechecked the BNF). (ast/parse_expr.go)
* Fixed certain cases in expression/name parsing. Some things are less permissive, others are more. (ast/parse.go
  ast/parse_expr.go)
* Fixed certain multiple assignment statements involving table assignments and direct assignments to the same variable.
  If the table assignment came first the direct assignment would clobber its register/upvalue and you would get an error
  or (even worse) unexpected behavior. This affected statements such as the following: `local a = {}; a[1], a = 1, 1`
  (compile.go)
* All numeric constants were *always* being treated as floats, leading to errors when you tried to use a hexadecimal
  constants (and probably other subtle issues). (ast/lexer.go)
* You may now use the shorthand null string escape sequence ('\0'). Thank you to whoever wrote the Lua spec, not having
  a proper list of valid escape sequences is really helpful /s. (ast/lexer.go)
* Both sides of a shift are now converted to an *unsigned* integer for the duration of the shift, then converted back to
  the proper signed type. This resolves some strangeness with bitwise shifts. (value.go)
* Removed various debugging print statements that I forgot to remove earlier. The only ones still in were a few that
  printed just before an error triggered, so it is unlikely anyone ever saw one... (all over the place)
