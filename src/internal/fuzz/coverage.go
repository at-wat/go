// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fuzz

import (
	"fmt"
	"internal/unsafeheader"
	"math/bits"
	"unsafe"
)

// coverage returns a []byte containing unique 8-bit counters for each edge of
// the instrumented source code. This coverage data will only be generated if
// `-d=libfuzzer` is set at build time. This can be used to understand the code
// coverage of a test execution.
func coverage() []byte {
	addr := unsafe.Pointer(&_counters)
	size := uintptr(unsafe.Pointer(&_ecounters)) - uintptr(addr)

	var res []byte
	*(*unsafeheader.Slice)(unsafe.Pointer(&res)) = unsafeheader.Slice{
		Data: addr,
		Len:  int(size),
		Cap:  int(size),
	}
	return res
}

// ResetCovereage sets all of the counters for each edge of the instrumented
// source code to 0.
func ResetCoverage() {
	cov := coverage()
	for i := range cov {
		cov[i] = 0
	}
}

// SnapshotCoverage copies the current counter values into coverageSnapshot,
// preserving them for later inspection. SnapshotCoverage also rounds each
// counter down to the nearest power of two. This lets the coordinator store
// multiple values for each counter by OR'ing them together.
func SnapshotCoverage() {
	cov := coverage()
	for i, b := range cov {
		b |= b >> 1
		b |= b >> 2
		b |= b >> 4
		b -= b >> 1
		coverageSnapshot[i] = b
	}
}

// diffCoverage returns a set of bits set in snapshot but not in base.
// If there are no new bits set, diffCoverage returns nil.
func diffCoverage(base, snapshot []byte) []byte {
	if len(base) != len(snapshot) {
		panic(fmt.Sprintf("the number of coverage bits changed: before=%d, after=%d", len(base), len(snapshot)))
	}
	found := false
	for i := range snapshot {
		if snapshot[i]&^base[i] != 0 {
			found = true
			break
		}
	}
	if !found {
		return nil
	}
	diff := make([]byte, len(snapshot))
	for i := range diff {
		diff[i] = snapshot[i] &^ base[i]
	}
	return diff
}

// countNewCoverageBits returns the number of bits set in snapshot that are not
// set in base.
func countNewCoverageBits(base, snapshot []byte) int {
	n := 0
	for i := range snapshot {
		n += bits.OnesCount8(snapshot[i] &^ base[i])
	}
	return n
}

// hasCoverageBit returns true if snapshot has at least one bit set that is
// also set in base.
func hasCoverageBit(base, snapshot []byte) bool {
	for i := range snapshot {
		if snapshot[i]&base[i] != 0 {
			return true
		}
	}
	return false
}

func countBits(cov []byte) int {
	n := 0
	for _, c := range cov {
		n += bits.OnesCount8(c)
	}
	return n
}

var (
	coverageEnabled  = len(coverage()) > 0
	coverageSnapshot = make([]byte, len(coverage()))

	// _counters and _ecounters mark the start and end, respectively, of where
	// the 8-bit coverage counters reside in memory. They're known to cmd/link,
	// which specially assigns their addresses for this purpose.
	_counters, _ecounters [0]byte
)
