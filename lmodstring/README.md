
The standard Lua string library is pretty pathetic in my opinion. Many simple operations are not possible without
lots of complicated code and/or regular expressions. Of course this is probably because Lua is written in C, and C
also has a pathetic string library (it doesn't even have an actual string type!).

I added some new string handling functions on top of the default. Some of these are similar to what Lua already has, but
without the regular expression support, others fill critical holes in the default API, and a few are lazy conveniences.

Most of these functions assume strings are UTF-8, be careful.

You will only have these functions if you import the `string` module, see the package example.

If you do *not* want these extensions simply set the `_NO_STRING_EXTS` registry key to any non-false value before
loading the string module. This can be done like so:

	l.Push("_NO_STRING_EXTS")
	l.Push(true)
	l.SetTableRaw(lua.RegistryIndex)

* * *

	function string.count(str, sub)

Returns the number of non-overlapping occurrences of `sub` in `str`.

* * *

	function string.hasprefix(str, prefix)

Returns true if `str` starts with `prefix`.

* * *

	function string.hassuffix(str, suffix)

Returns true if `str` ends with `suffix`.

* * *

	function string.join(table, [sep])

Joins all the values in `table` with `sep`. If `sep` is not specified then it defaults to ", "

Yes, I know there is a function in the `table` module that does something similar.

* * *

	function string.replace(str, old, new, [n])

Replaces `n` occurrences of `old` with `new` in `str`.
If `n` < 0 then there is no limit on replacements.

* * *

	function string.split(str, sep, [n])

Split `str` into `n` substrings at ever occurrence of `sep`.

* `n` > 0: at most n substrings; the last substring will be the unsplit remainder
* `n` == 0: the result is an empty table
* `n` < 0: all substrings

* * *

	function string.splitafter(str, sep, [n])

This is exactly like `strings.split`, except `sep` is retained as part of the substrings.

* * *

	function string.title(str)

Convert the first character of every word in `str` to it's title case.

* * *

	function string.trim(str, cut)

Returns `str` with any chars in `cut` removed from it's beginning and end.

* * *

	function string.trimprefix(str, prefix)

Returns `str` with `prefix` removed from it's start. `str` is returned unchanged if it does not start with `prefix`.

* * *

	function string.trimspace(str)

Returns `str` with all whitespace trimmed from it's beginning and end.

* * *

	function string.trimsuffix(str, suffix)

Returns `str` with `suffix` removed from it's end. `str` is returned unchanged if it does not end with `suffix`.

* * *

	function string.unquote(str)

If `str` begins and ends with a quote char (one of `` "'` ``) then it will be unquoted using the rules for the
[Go](golang.org) language. This includes escape sequence expansion.
