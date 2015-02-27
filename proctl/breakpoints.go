package proctl

import (
	"fmt"
	"runtime"
)

// Represents a single breakpoint. Stores information on the break
// point including the byte of data that originally was stored at that
// address.
type BreakPoint struct {
	FunctionName string
	File         string
	Line         int
	Addr         uint64
	OriginalData []byte
	ID           int
	temp         bool
}

// Returned when trying to set a breakpoint at
// an address that already has a breakpoint set for it.
type BreakPointExistsError struct {
	file string
	line int
	addr uint64
}

func (bpe BreakPointExistsError) Error() string {
	return fmt.Sprintf("Breakpoint exists at %s:%d at %x", bpe.file, bpe.line, bpe.addr)
}

// InvalidAddressError represents the result of
// attempting to set a breakpoint at an invalid address.
type InvalidAddressError struct {
	address uint64
}

func (iae InvalidAddressError) Error() string {
	return fmt.Sprintf("Invalid address %#v\n", iae.address)
}

// Returns whether or not a breakpoint has been set for the given address.
func (dbp *DebuggedProcess) BreakpointExists(addr uint64) bool {
	for _, bp := range dbp.HWBreakPoints {
		// TODO(darwin)
		if runtime.GOOS == "darwin" {
			break
		}
		if bp != nil && bp.Addr == addr {
			return true
		}
	}
	if _, ok := dbp.BreakPoints[addr]; ok {
		return true
	}
	return false
}

func (dbp *DebuggedProcess) newBreakpoint(fn, f string, l int, addr uint64, data []byte) *BreakPoint {
	dbp.breakpointIDCounter++
	return &BreakPoint{
		FunctionName: fn,
		File:         f,
		Line:         l,
		Addr:         addr,
		OriginalData: data,
		ID:           dbp.breakpointIDCounter,
	}
}

func (dbp *DebuggedProcess) setBreakpoint(tid int, addr uint64) (*BreakPoint, error) {
	var f, l, fn = dbp.GoSymTable.PCToLine(uint64(addr))
	if fn == nil {
		return nil, InvalidAddressError{address: addr}
	}
	if dbp.BreakpointExists(addr) {
		return nil, BreakPointExistsError{f, l, addr}
	}
	// Try and set a hardware breakpoint.
	for i, v := range dbp.HWBreakPoints {
		// TODO(darwin)
		if runtime.GOOS == "darwin" {
			break
		}
		if v == nil {
			if err := setHardwareBreakpoint(i, tid, addr); err != nil {
				return nil, fmt.Errorf("could not set hardware breakpoint: %v", err)
			}
			dbp.HWBreakPoints[i] = dbp.newBreakpoint(fn.Name, f, l, addr, nil)
			return dbp.HWBreakPoints[i], nil
		}
	}
	// Fall back to software breakpoint. 0xCC is INT 3, software
	// breakpoint trap interrupt.
	thread := dbp.Threads[tid]
	originalData := make([]byte, 1)
	if _, err := readMemory(thread, uintptr(addr), originalData); err != nil {
		return nil, err
	}
	if _, err := writeMemory(thread, uintptr(addr), []byte{0xCC}); err != nil {
		return nil, err
	}
	dbp.BreakPoints[addr] = dbp.newBreakpoint(fn.Name, f, l, addr, originalData)
	return dbp.BreakPoints[addr], nil
}

func (dbp *DebuggedProcess) clearBreakpoint(tid int, addr uint64) (*BreakPoint, error) {
	// Check for hardware breakpoint
	for i, bp := range dbp.HWBreakPoints {
		if bp == nil {
			continue
		}
		if bp.Addr == addr {
			dbp.HWBreakPoints[i] = nil
			if err := clearHardwareBreakpoint(i, tid); err != nil {
				return nil, err
			}
			return bp, nil
		}
	}
	// Check for software breakpoint
	if bp, ok := dbp.BreakPoints[addr]; ok {
		thread := dbp.Threads[tid]
		if _, err := writeMemory(thread, uintptr(bp.Addr), bp.OriginalData); err != nil {
			return nil, fmt.Errorf("could not clear breakpoint %s", err)
		}
		delete(dbp.BreakPoints, addr)
		return bp, nil
	}
	return nil, fmt.Errorf("No breakpoint currently set for %#v", addr)
}
