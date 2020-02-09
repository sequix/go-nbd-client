package main

import (
	"syscall"

	"github.com/davecgh/go-spew/spew"
)

func main() {
	a, b, c := syscall.Syscall(syscall.SYS_IOCTL, 0, 0, 0)
	spew.Dump(a, b, c)
}
