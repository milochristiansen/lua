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

import "github.com/milochristiansen/lua/luautil"

// stack is a special "segmented" stack structure. Each callFrame added to the stack gets it's own little
// isolated "segment". Once you add a new callFrame you cannot push and pop values in the previous
// callFrame(s) segments, but you can modify them by index.
type stack struct {
	data     []value
	frames   []*callFrame
	
	// List of all unclosed upvalues (which by definition are on the stack somewhere).
	// This list is ordered higher indexes to lower indexes by requirement and construction.
	unclosed *upValue
}

func newStack() *stack {
	stk := &stack{
		data:   make([]value, 0, 1024),
		frames: make([]*callFrame, 1, 64),
	}
	
	stk.frames[0] = &callFrame{
		stk: stk,
		base: -1,
	}
	
	return stk
}

// bounds returns the boundary values for the given frame.
// Negative indexes are relative to TOS, positive indexes are absolute.
// "segC" is the index of the last item of the previous frame.
// "segN" is the index of the last item of the current frame.
func (stk *stack) bounds(seg int) (segC int, segN int) {
	if len(stk.frames) == 0 {
		luautil.Raise("No frames on the stack.", luautil.ErrTypMajorInternal)
	}
	
	if seg < 0 {
		// Relative index.
		seg = len(stk.frames) + seg
		if seg < 0 {
			luautil.Raise("Requested frame not on the stack.", luautil.ErrTypMajorInternal)
		}
	} else {
		// Absolute index.
		if seg >= len(stk.frames) {
			luautil.Raise("Requested frame not on the stack.", luautil.ErrTypMajorInternal)
		}
	}

	frame := stk.frames[seg]
	
	// segC is easy.
	segC = frame.base
	if frame.holdArgs {
		segC += frame.nArgs
	}

	// segN is a little harder.
	if seg+1 < len(stk.frames) {
		segN = stk.frames[seg+1].base
	} else {
		segN = len(stk.data) - 1
	}

	return segC, segN
}

// topBounds is the same as bounds, but always for the index -1.
// When the stack has no segments this will return "-1, -1" instead of raising an error.
// Using this function instead of bounds(-1) produces a small, but measurable, performance improvement.
func (stk *stack) topBounds() (segC int, segN int) {
	if len(stk.frames) == 0 {
		return -1, -1
	}
	
	seg := len(stk.frames) - 1
	frame := stk.frames[seg]
	
	// segC is easy.
	segC = frame.base
	if frame.holdArgs {
		segC += frame.nArgs
	}

	// segN is a little harder.
	if seg+1 < len(stk.frames) {
		segN = stk.frames[seg+1].base
	} else {
		segN = len(stk.data) - 1
	}

	return segC, segN
}

// ensure makes sure that the index i is valid and is writable. The index is global, not an index into a particular frame.
func (stk *stack) ensure(i int) {
	ssize := len(stk.data)
	if i < ssize {
		return
	}
	needed := i - ssize
	if ssize + needed <= cap(stk.data) {
		stk.data = stk.data[:ssize + needed + 1]
	} else {
		stk.data = append(stk.data, make([]value, needed)...)
	}
}

// cFrame returns the current stack frame.
func (stk *stack) cFrame() *callFrame {
	index := len(stk.frames) - 1
	if index < 0 {
		luautil.Raise("No frames on the stack.", luautil.ErrTypMajorInternal)
	}
	return stk.frames[index]
}

func (stk *stack) frame(lvl int) *callFrame {
	index := len(stk.frames) - lvl
	if index < 0 {
		return nil
	}
	return stk.frames[index]
}

// absIndex turns a frame index into an absolute index.
func (stk *stack) absIndex(index int) int {
	segC, segN := stk.topBounds()

	if index >= 0 {
		return segC+index+1
	}
	return segN+index+1
}

// TopIndex returns the index of the top item in the current stack frame.
// If the stack is empty -1 will be returned.
func (stk *stack) TopIndex() int {
	segC, segN := stk.topBounds()
	if segC < 0 {
		return segN
	}
	// segN - segC is the segment length, - 1 for the index
	return segN - segC - 1
}

// SetTop drops values from the stack so that t is the new top index.
// If the top index >= t then nothing is done.
func (stk *stack) SetTop(t int) {
	r := stk.TopIndex() - t
	if r <= 0 {
		return
	}
	
	stk.Pop(r)
}

// Get returns the value at the given index relative to the top of stack (negative indexes) or the
// frame boundary (positive indexes).
// If the index is out of the current frame's bounds then nil will be returned.
func (stk *stack) Get(index int) value {
	segC, segN := stk.topBounds()

	if index >= 0 {
		if segC+index+1 > segN {
			return nil
		}
		return stk.data[segC+index+1]
	}

	if segN+index+1 <= segC {
		return nil
	}
	return stk.data[segN+index+1]
}

// GetArgs reads any values "protected" by `frame.holdArgs`.
// Reading items beyond the protected range returns nil.
// Positive indexes only!
func (stk *stack) GetArgs(index int) value {
	segC, _ := stk.topBounds()
	
	frame := stk.cFrame()
	if !frame.holdArgs || index >= frame.nArgs {
		return nil
	}
	
	return stk.data[segC-frame.nArgs+index+1]
}

// GetInFrame acts like Get, except it looks the index up in the specified stack frame.
// Used for handling upvalues.
func (stk *stack) GetInFrame(frame, index int) value {
	segC, segN := stk.bounds(frame)

	if index >= 0 {
		if segC+index+1 > segN {
			return nil
		}
		return stk.data[segC+index+1]
	}

	if segN+index+1 <= segC {
		return nil
	}
	return stk.data[segN+index+1]
}

// GetAbs acts like Get, except it looks an absolute index up.
// Used for handling upvalues.
func (stk *stack) GetAbs(index int) value {
	if index < 0 || index >= len(stk.data) {
		return nil
	}
	return stk.data[index]
}

// Set modifies the value at the given index relative to the top of stack (negative indexes) or the
// frame boundary (positive indexes).
// If the given index is absolute and outside of the frame bounds then the frame is extended.
func (stk *stack) Set(index int, val value) {
	segC, segN := stk.topBounds()
	
	if index >= 0 {
		stk.ensure(segC+index+1)
		stk.data[segC+index+1] = val
		return
	}

	if segN+index+1 <= segC {
		return
	}
	stk.data[segN+index+1] = val
}

// Set modifies the value at the given index relative to the top of stack (negative indexes) or the
// frame boundary (positive indexes).
// If the given index is absolute and outside of the frame bounds then the frame is extended.
func (stk *stack) SetInFrame(frame, index int, val value) {
	segC, segN := stk.bounds(frame)
	
	if index >= 0 {
		stk.ensure(segC+index+1)
		stk.data[segC+index+1] = val
		return
	}

	if segN+index+1 <= segC {
		return
	}
	stk.data[segN+index+1] = val
}

// SetAbs modifies the value at the given absolute index.
// If the given index is outside of the frame bounds then the frame is extended.
func (stk *stack) SetAbs(index int, val value) {
	if index >= 0 {
		stk.ensure(index)
		stk.data[index] = val
		return
	}

	if index < 0 {
		return
	}
	stk.data[index] = val
}

// Pop removes n values from the top of the current segment.
func (stk *stack) Pop(n int) {
	segC, segN := stk.topBounds()
	
	for i := 0; i < n; i++ {
		if segN-i <= segC {
			break
		}
		stk.data[segN-i] = nil
	}
	
	if (segN+1)-n <= segC {
		stk.data = stk.data[:segC+1]
		return
	}
	stk.data = stk.data[:(segN+1)-n]
}

// Push adds a new value to the stack.
// If there are no frames on the stack nothing is done.
func (stk *stack) Push(val value) {
	if len(stk.frames) == 0 {
		luautil.Raise("No frames on the stack.", luautil.ErrTypMajorInternal)
	}

	stk.data = append(stk.data, val)
}

// Insert the given item at the given index, shifting the other stack items up to allow it to fit.
func (stk *stack) Insert(i int, v value) {
	segC, segN := stk.topBounds()
	
	if i < 0 {
		i = segN+i+1
	} else {
		i = segC+i+1
	}
	
	if i > segN || i <= segC {
		luautil.Raise("Index out of range for Insert.", luautil.ErrTypGenRuntime)
	}
	
	stk.data = append(stk.data, nil)
	segN++
	
	for k := segN; k > i; k-- {
		stk.data[k] = stk.data[k-1]
	}
	stk.data[i] = v
}

// FrameIndex returns the current frame index or -1 if there are no frames on the stack.
func (stk *stack) FrameIndex() int {
	return len(stk.frames) - 1
}

// AddFrame adds a new callFrame to the stack.
// args must be the real argument count, and there must be at least that many items between fi and the TOS.
func (stk *stack) AddFrame(fn *function, fi, args, rtns int) {
	pbase := -1
	if len(stk.frames) > 0 {
		pbase = stk.cFrame().base
	}
	
	// Remove any items that are above the last argument
	// It is rare, but possible, to have such values.
	stk.SetTop(fi + args)
	
	base := len(stk.data) - 1 - args
	if base < pbase {
		luautil.Raise("Invalid argument count for AddFrame.", luautil.ErrTypMajorInternal)
	}
	
	frame := &callFrame{
		fn:  fn,
		stk: stk,

		base: base,

		holdArgs: fn.native == nil && fn.proto.isVarArg == 1,
		
		nArgs:   args,
		nRet:    rtns,
		retBase: -1,
		retTo:   fi,
	}

	//println("New Frame:")
	//for i := base + 1; i < len(stk.data); i++ {
	//	println(" ", toString(stk.data[i]))
	//}
	
	stk.frames = append(stk.frames, frame)
}

// TailFrame prepares the frame on TOS for use in a tail call.
func (stk *stack) TailFrame(fn *function, fi, args int) {
	if len(stk.frames) == 0 {
		luautil.Raise("No frames on the stack to use for tail call.", luautil.ErrTypMajorInternal)
	}
	frame := stk.cFrame()
	segC, segN := stk.topBounds()
	
	// Calculate the "real segC" accounting for any protected arguments
	rsegC := segC
	if frame.holdArgs {
		rsegC -= frame.nArgs
	}
	
	frame.pc = 0
	frame.fn = fn
	
	frame.holdArgs = fn.native == nil && fn.proto.isVarArg == 1
	
	frame.nArgs = args
	frame.nRet = -1
	frame.retBase = -1
	// frame.retTo = fi // No touch! This is the index in the PREVIOUS FRAME, not the current frame!
	
	// shortcut
	if args <= 0 {
		for i := rsegC + 1; i <= segN; i++ {
			stk.data[i] = nil
		}
		stk.data = stk.data[:rsegC+1]
		return
	}
	
	// Shift the arguments to the bottom of the frame
	for i := 0; i < args; i++ {
		stk.data[rsegC + 1 + i] = stk.data[segC + 1 + fi + 1 + i]
	}
	for i := rsegC + 1 + args; i < len(stk.data); i++ {
		stk.data[i] = nil
	}
	stk.data = stk.data[:rsegC+args+1]
}

// ReturnFrame drops the current frame, saving the return values by adding them to the top of the previous frame.
// Information about which values to save is gotten from the current frame data.
func (stk *stack) ReturnFrame() {
	if len(stk.frames) <= 1 {
		luautil.Raise("Not enough frames on the stack.", luautil.ErrTypMajorInternal)
	}
	frame := stk.cFrame()
	
	segC, segN := stk.bounds(-1)
	psegC, _ := stk.bounds(-2)
	
	if frame.retBase < 0 {
		luautil.Raise("Frame return with invalid (missing) return data.", luautil.ErrTypMajorInternal)
	}
	
	//println("Remove Frame:")
	//for i := frame.base + 1; i < len(stk.data); i++ {
	//	println(" ", toString(stk.data[i]))
	//}
	
	// Amazingly we don't need any special correction for frame.holdArgs
	
	for i := 0; i < stk.TopIndex() + 1 - frame.retBase; i++ {
		stk.data[psegC+1+frame.retTo+i] = stk.data[segC+1+i+frame.retBase]
	}
	
	retC := frame.retC // The number of items that were returned
	retE := frame.nRet // The number of return items required by the caller
	if retE < 0 {
		retE = retC
	} else if retE < retC {
		retC = retE
	}
	
	// Wipe everything but the return values
	for i := psegC+1+frame.retTo+retC; i <= segN; i++ {
		stk.data[i] = nil
	}
	stk.data = stk.data[:psegC+1+frame.retTo+retC]
	
	// Now correct for retC < retE
	for retC < retE {
		stk.data = append(stk.data, nil)
		retC++
	}
	
	stk.frames = stk.frames[:len(stk.frames)-1]
}

func (stk *stack) DropFrame() {
	segC, segN := stk.topBounds()
	
	if frame := stk.cFrame(); frame.holdArgs {
		segC -= frame.nArgs
	}
	
	for i := segC + 1; i <= segN; i++ {
		stk.data[i] = nil
	}
	
	stk.data = stk.data[:segC+1]
	stk.frames = stk.frames[:len(stk.frames)-1]
}
