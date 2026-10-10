package main

import (
    "errors"
    "syscall"
)

func connectionRefused(err error) bool {
    // Winsock WSAECONNREFUSED differs from syscall's application error code.
    const wsaConnectionRefused = syscall.Errno(10061)
    return errors.Is(err,wsaConnectionRefused)
}
