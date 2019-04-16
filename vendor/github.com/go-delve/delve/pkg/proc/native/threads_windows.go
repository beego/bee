package native

import (
	"errors"
	"syscall"

	sys "golang.org/x/sys/windows"

	"github.com/go-delve/delve/pkg/proc"
	"github.com/go-delve/delve/pkg/proc/winutil"
)

// WaitStatus is a synonym for the platform-specific WaitStatus
type WaitStatus sys.WaitStatus

// OSSpecificDetails holds information specific to the Windows
// operating system / kernel.
type OSSpecificDetails struct {
	hThread syscall.Handle
}

func (t *Thread) singleStep() error {
	context := winutil.NewCONTEXT()
	context.ContextFlags = _CONTEXT_ALL

	// Set the processor TRAP flag
	err := _GetThreadContext(t.os.hThread, context)
	if err != nil {
		return err
	}

	context.EFlags |= 0x100

	err = _SetThreadContext(t.os.hThread, context)
	if err != nil {
		return err
	}

	_, err = _ResumeThread(t.os.hThread)
	if err != nil {
		return err
	}

	for {
		var tid, exitCode int
		t.dbp.execPtraceFunc(func() {
			tid, exitCode, err = t.dbp.waitForDebugEvent(waitBlocking | waitSuspendNewThreads)
		})
		if err != nil {
			return err
		}
		if tid == 0 {
			t.dbp.postExit()
			return proc.ErrProcessExited{Pid: t.dbp.pid, Status: exitCode}
		}

		if t.dbp.os.breakThread == t.ID {
			break
		}

		t.dbp.execPtraceFunc(func() {
			err = _ContinueDebugEvent(uint32(t.dbp.pid), uint32(t.dbp.os.breakThread), _DBG_CONTINUE)
		})
	}

	_, err = _SuspendThread(t.os.hThread)
	if err != nil {
		return err
	}

	t.dbp.execPtraceFunc(func() {
		err = _ContinueDebugEvent(uint32(t.dbp.pid), uint32(t.ID), _DBG_CONTINUE)
	})
	if err != nil {
		return err
	}

	// Unset the processor TRAP flag
	err = _GetThreadContext(t.os.hThread, context)
	if err != nil {
		return err
	}

	context.EFlags &= ^uint32(0x100)

	return _SetThreadContext(t.os.hThread, context)
}

func (t *Thread) resume() error {
	var err error
	t.dbp.execPtraceFunc(func() {
		//TODO: Note that we are ignoring the thread we were asked to continue and are continuing the
		//thread that we last broke on.
		err = _ContinueDebugEvent(uint32(t.dbp.pid), uint32(t.ID), _DBG_CONTINUE)
	})
	return err
}

func (t *Thread) Blocked() bool {
	// TODO: Probably incorrect - what are the runtime functions that
	// indicate blocking on Windows?
	regs, err := t.Registers(false)
	if err != nil {
		return false
	}
	pc := regs.PC()
	fn := t.BinInfo().PCToFunc(pc)
	if fn == nil {
		return false
	}
	switch fn.Name {
	case "runtime.kevent", "runtime.usleep":
		return true
	default:
		return false
	}
}

// Stopped returns whether the thread is stopped at the operating system
// level. On windows this always returns true.
func (t *Thread) Stopped() bool {
	return true
}

func (t *Thread) WriteMemory(addr uintptr, data []byte) (int, error) {
	if t.dbp.exited {
		return 0, proc.ErrProcessExited{Pid: t.dbp.pid}
	}
	if len(data) == 0 {
		return 0, nil
	}
	var count uintptr
	err := _WriteProcessMemory(t.dbp.os.hProcess, addr, &data[0], uintptr(len(data)), &count)
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

var ErrShortRead = errors.New("short read")

func (t *Thread) ReadMemory(buf []byte, addr uintptr) (int, error) {
	if t.dbp.exited {
		return 0, proc.ErrProcessExited{Pid: t.dbp.pid}
	}
	if len(buf) == 0 {
		return 0, nil
	}
	var count uintptr
	err := _ReadProcessMemory(t.dbp.os.hProcess, addr, &buf[0], uintptr(len(buf)), &count)
	if err == nil && count != uintptr(len(buf)) {
		err = ErrShortRead
	}
	return int(count), err
}

func (t *Thread) restoreRegisters(savedRegs proc.Registers) error {
	return _SetThreadContext(t.os.hThread, savedRegs.(*winutil.AMD64Registers).Context)
}
