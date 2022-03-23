package lua

import (
	"testing"
)

func TestEnsureGrowthAroundSliceCapacityBoundary(t *testing.T) {
	stk := newStack()
	c := cap(stk.data)

	//We explore behaviour around the earliest capacity increase in stk.data
	for i := c - 2; i < c+2; i++ {
		stk.ensure(i)
		stk.data[i] = i
	}
}
