package sys

import (
	"runtime"
	"syscall"
	"unsafe"

	"github.com/cilium/ebpf/internal/unix"
)

// Raw wraps SYS_BPF.
//
// Any pointers contained in attr must use the Pointer type from this package.
func Raw(cmd Cmd, attr unsafe.Pointer, size uintptr) (uintptr, error) {
	for {
		r1, _, errNo := unix.Syscall(unix.SYS_BPF, uintptr(cmd), uintptr(attr), size)
		runtime.KeepAlive(attr)

		// As of ~4.20 the verifier can be interrupted by a signal,
		// and returns EAGAIN in that case.
		if errNo == unix.EAGAIN && cmd == BPF_PROG_LOAD {
			continue
		}

		var err error
		if errNo != 0 {
			err = wrappedErrno{errNo}
		}

		return r1, err
	}
}

type Attr interface {
	cmd() (Cmd, unsafe.Pointer, uintptr)
}

func BPF(attr Attr) (uintptr, error) {
	return Raw(attr.cmd())
}

func BPFFd(attr Attr) (*FD, error) {
	fd, err := BPF(attr)
	if err != nil {
		return nil, err
	}

	return NewFD(int(fd)), nil
}

// BPFObjName is a null-terminated string made up of
// 'A-Za-z0-9_' characters.
type ObjName [unix.BPF_OBJ_NAME_LEN]byte

// NewObjName truncates the result if it is too long.
func NewObjName(name string) ObjName {
	var result ObjName
	copy(result[:unix.BPF_OBJ_NAME_LEN-1], name)
	return result
}

// wrappedErrno wraps syscall.Errno to prevent direct comparisons with
// syscall.E* or unix.E* constants.
//
// You should never export an error of this type.
type wrappedErrno struct {
	syscall.Errno
}

func (we wrappedErrno) Unwrap() error {
	return we.Errno
}

type syscallError struct {
	error
	errno syscall.Errno
}

func Error(err error, errno syscall.Errno) error {
	return &syscallError{err, errno}
}

func (se *syscallError) Is(target error) bool {
	return target == se.error
}

func (se *syscallError) Unwrap() error {
	return se.errno
}