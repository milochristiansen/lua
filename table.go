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

// Encoding canary: '§' ALT+0167, ALT+21
// If the above line does not contain a "section" character then this file has not been properly recognized as UTF-8.
// (sometimes Notepad++ thinks this is encoded in TIS-620 (Thai) and breaks the section characters)

package lua

import "math"
import "runtime"

// Set to 0 for zero based table indexing. This is only partly tested!
var TableIndexOffset = 1

// table is the VM's table type.
type table struct {
	meta *table
	l    *State

	array      []value
	length     int // The stored sequence length, negative values signify that the length needs to be recalculated.

	hash map[value]value
	
	// For use with next
	iorder []value
	ikeys  map[value]int
}

func newTable(l *State, as, hs int) *table {
	t := new(table)
	t.l = l
	
	if as > 0 {
		t.array = make([]value, as)
	}
	if hs > 0 {
		t.hash = make(map[value]value, hs)
	} else {
		t.hash = make(map[value]value)
	}
	
	return t
}

// extend grows the table's underlying array part until it is last elements long, then tries to move as many
// items from the hash part to the array part as possible.
func (tbl *table) extend(last int) {
	tbl.array = append(tbl.array, make([]value, last-len(tbl.array))...)
	for k, v := range tbl.hash {
		switch idx := k.(type) {
		case float64:
			if i := int(idx); float64(i) == idx {
				i2 := i - TableIndexOffset
				if 0 <= i2 && i2 < len(tbl.array) {
					tbl.array[i2] = v
					delete(tbl.hash, k)
				}
			}
		case int64:
			if i2 := int(idx) - TableIndexOffset; 0 <= i2 && i2 < len(tbl.array) {
				tbl.array[i2] = v
				delete(tbl.hash, k)
			}
		}
	}
}

// maybeExtend checks to see if it is worth extending the array part of the table to fit key.
// If the return value is true then the array can fit the the key (either it was extended or was already large enough).
// This takes an index from 0!
func (tbl *table) maybeExtend(key int) bool {
	if key < len(tbl.array) {
		// Array long enough.
		return true
	}

	// TODO: Is there a better way to handle this kind of situation?
	// The current algorithm is kinda slow, is there some way to cut down on hash lookups?
	
	occupancy := 0
	for _, v := range tbl.array {
		if v != nil {
			occupancy++
		}
	}

	for k, _ := range tbl.hash {
		switch idx := k.(type) {
		case float64:
			if i := int(idx); float64(i) == idx {
				if i < key {
					occupancy++
				}
			}
		case int64:
			if int(idx) < key {
				occupancy++
			}
		}
	}
	
	if occupancy == 0 && key == 0 {
		tbl.array = make([]value, 1, 32)
		return true
	}
	if occupancy > key/2 {
		// Scan upward looking for the first place occupancy falls below 50%.
		// This is required to guarantee that contiguous array segments are not left out, if this
		// was not done then building an array starting from the last index and working down would
		// result in everything being stored in hash indexes. Building from the top down is still
		// a bad idea (as you will have half the items added before crossing the threshold for
		// creating an array), but at least it works now...
		o := occupancy
		k := key
		n := 0
		for ; o > k/2; k++ {
			if tbl.hash[int64(k)] != nil {
				o++
				n++
			}
		}
		// If array candidates are found in the >50% area above key, extend the array to the top of the overall >50% area.
		if n > 0 {
			tbl.extend(o)
			return true
		}
		
		occupancy *= 2
		if occupancy > key {
			tbl.extend(occupancy)
			return true
		}

		tbl.extend(key)
		return true
	}
	return false
}

// Internal helper
func (tbl *table) setInt(k int, v value) {
	// Store the value or clear the key.
	hash := false
	k2 := k - TableIndexOffset
	if k2 >= 0 && k2 < len(tbl.array) {
		tbl.array[k2] = v
	} else if k2 >= 0 && v != nil && tbl.maybeExtend(k2) {
		tbl.array[k2] = v
	} else if v == nil {
		hash = true
		delete(tbl.hash, int64(k))
	} else {
		hash = true
		tbl.hash[int64(k)] = v
	}

	// Decide if the stored length was invalidated and fix it if possible.
	
	// No need to do anything if the value was stored/removed from the hash part.
	// We don't need to worry about values added to the hash when the array portion is full, as this is impossible.
	if hash {
		return
	}

	// If we removed a key, check to see if it is inside the old sequence, then shorten the sequence as needed.
	if v == nil {
		if k2 < tbl.length {
			// All items before the one we just removed are known valid.
			tbl.length = k2
		}
		return
	}

	// If we set a key that is immediately after the end of the old sequence then invalidate the stored length.
	// We don't try to calculate the new length because we have no way of knowing how long it will take or even
	// if it will ever be needed. It would be possible to store the old length to give the next scan a running
	// start, but that would complicate things for a gain that would likely be very small.
	if k2 == tbl.length {
		tbl.length = -1
	}
}

// Exists returns true if the given index exists in the table.
func (tbl *table) Exists(k value) bool {
	switch idx := k.(type) {
	case nil:
		return false
	case float64:
		if math.IsNaN(idx) {
			return false
		}

		if i := int(idx); float64(i) == idx {
			return tbl.existsInt(i)
		}
		v, ok := tbl.hash[k]
		return ok && v != nil
	case int64:
		return tbl.existsInt(int(idx))
	default:
		v, ok := tbl.hash[k]
		return ok && v != nil
	}
}

func (tbl *table) existsInt(k int) bool {
	k2 := k - TableIndexOffset
	if 0 <= k2 && k2 < len(tbl.array) {
		return tbl.array[k2] != nil
	}
	v, ok := tbl.hash[int64(k)]
	return ok && v != nil
}

// SetRaw sets a key k in the table to the value v without using any meta methods.
func (tbl *table) SetRaw(k, v value) {
	switch idx := k.(type) {
	case nil:
		return
	case float64:
		if math.IsNaN(idx) {
			return
		}

		if i := int(idx); float64(i) == idx {
			tbl.setInt(i, v)
		} else if v == nil {
			delete(tbl.hash, k)
		} else {
			tbl.hash[k] = v
		}
	case int64:
		tbl.setInt(int(idx), v)
	default:
		if v == nil {
			delete(tbl.hash, k)
		} else {
			tbl.hash[k] = v
		}
	}
}

// Internal helper
func (tbl *table) getInt(k int) value {
	k2 := k - TableIndexOffset
	if 0 <= k2 && k2 < len(tbl.array) {
		return tbl.array[k2]
	}
	return tbl.hash[int64(k)]
}

// GetRaw reads the value at index k from the table without using any meta methods.
func (tbl *table) GetRaw(k value) value {
	switch idx := k.(type) {
	case nil:
		return nil
	case float64:
		if math.IsNaN(idx) {
			return nil
		}

		if i := int(idx); float64(i) == idx {
			return tbl.getInt(i)
		}
	case int64:
		return tbl.getInt(int(idx))
	}
	// Non-number or non-integral float.
	return tbl.hash[k]
}

// length returns the raw table length as would be returned by the length operator.
//
// The Lua spec does not match the reference implementation here, ironically the example the spec
// gives as something that won't work works fine in practice. Here I follow the spec (mostly
// because it is easier that way).
//
// Actually the reference implementation seems to be weirdly inconsistent...
//	print(#{1, 2, 3, nil, 4}) -- Prints "5" (spec says should be "3")
//	print(#{1, 2, 3, nil}) -- Prints "3" (this matches the spec)
//	print(#{a = 1, 1, [3] = 2, b = 2, c = 3}) -- Prints "1" (also correct)
//	print(#{[1] = "", [2] = "", [3] = nil, [4] = ""}) -- Prints "4" (spec says should be "2")
//
// Selected quotes from the Lua Reference Manual (for Lua 5.3):
//
// From §2.1:
//	Any key with value nil is not considered part of the table. Conversely, any key that is not part of a
//	table has an associated value nil.
//
// From §2.1:
//	We use the term sequence to denote a table where the set of all positive numeric keys is equal to {1..n}
//	for some non-negative integer n, which is called the length of the sequence (see §3.4.7).
//
// From §3.4.7:
//	Unless a __len metamethod is given, the length of a table t is only defined if the table is a sequence,
//	that is, the set of its positive numeric keys is equal to {1..n} for some non-negative integer n. In that
// 	case, n is its length.
//
// I read this to mean that a sequence starts with index 1 and runs to the last integer index that is non-nil
// where there are no nil (aka missing) values between them. Technically the spec makes it sound like ANY holes
// in the array should keep it from being a sequence AT ALL, but this seems like overkill and too hard to implement.
func (tbl *table) Length() int {
	// If possible use the stored length.
	if tbl.length >= 0 {
		return tbl.length
	}

	// Find the length of the the array up to the first nil value.
	// Due to the way tbl.extend works we don't have to worry about the hash part
	// (adding an item to the end of a non-sparse array will always extend it).
	length := 0
	for _, v := range tbl.array {
		if v == nil {
			break
		}
		length++
	}
	tbl.length = length
	return length
}

// Next allows you to iterate over all keys in a table.
// This function is NOT reentrant! You cannot iterate over a table while another iteration (of the same table) is
// ongoing, trying to do so will mess up the iteration order causing some keys to not be visited or to be visited twice.
// To start (or restart) an iteration pass nil as the key, subsequent passes should pass the key
// gotten from the previous call. Once there are no more items this function returns nil for both key and value.
// Passing in a key that does not exist will also make this return nil, nil.
func (tbl *table) Next(key value) (value, value) {
	if key == nil || tbl.ikeys == nil {
		tbl.ikeys = make(map[value]int, len(tbl.hash))
		tbl.iorder = make([]value, 0, len(tbl.hash))
		var last value
		for i := range tbl.hash {
			last = i
			tbl.iorder = append(tbl.iorder, i)
			tbl.ikeys[i] = len(tbl.iorder)
		}
		if last != nil {
			tbl.ikeys[last] = -1
		}
	}
	
	// Try the key as a hash index.
	// We need a loop here to handle the case where a key was removed while iterating.
	idx, ok := 0, true
	if key != nil {
		idx, ok = tbl.ikeys[key]
	}
	for ok && idx >= 0 && idx < len(tbl.iorder) {
		k := tbl.iorder[idx]
		v := tbl.hash[k]
		if v == nil {
			idx, ok = tbl.ikeys[k]
			continue
		}
		return k, v
	}
	if idx == -1 || key == nil {
		key = int64(0)
	}
	
	// Not in hash or all hash keys used, try to use the key as an array index.
	i, ok := tryInt(key)
	if ok {
		// Find the next valid key after i
		i++
		for i <= int64(len(tbl.array)) {
			v := tbl.array[i-int64(TableIndexOffset)]
			if v != nil {
				return i, v
			}
			i++
		}
	}
	
	// Key did not exist when iteration started or is the last key.
	tbl.ikeys = nil
	tbl.iorder = nil
	return nil, nil
}

// The real table iterator
// This one is reentrant.
type tableIter struct {
	kill chan bool
	result chan []value
	
	data *table
}

func newTableIter(d *table) *tableIter {
	kill := make(chan bool)
	result := make(chan []value)
	
	i := &tableIter{
		kill: kill,
		result: result,
		data: d,
	}
	
	// So long as this function contains no references to i it will die when i is finalized (references here
	// will keep i from ever being finalized unless you visit every key).
	// d is not a problem, as i will always be collected first.
	go func(){
		for k, v := range d.array {
			if v == nil {
				continue
			}
			
			select {
			case <- kill:
				close(result) // Just in case...
				return
			case result <- []value{k+TableIndexOffset, v}:
			}
		}
		
		for k, v := range d.hash {
			select {
			case <- kill:
				close(result) // Just in case...
				return
			case result <- []value{k, v}:
			}
		}
		close(result) // Needed so Next will not block after the last key is visited.
	}()
	
	runtime.SetFinalizer(i, func(i *tableIter){
		i.kill <- true
	})
	
	return i
}

func (i *tableIter) Next() (value, value) {
	k, ok := <- i.result
	if !ok {
		return nil, nil
	}
	return k[0], k[1]
}