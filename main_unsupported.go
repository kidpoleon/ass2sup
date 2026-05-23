//go:build !windows

// This file provides a meaningful compile-time-visible entry point on
// non-Windows platforms. ass2sup depends on Spp2Pgs.exe and xy-VSSppf.dll,
// which are Windows-only binaries, so the program cannot function on any
// other operating system.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "  ass2sup is a Windows-only application.")
	fmt.Fprintln(os.Stderr, "  It requires Spp2Pgs.exe and xy-VSSppf.dll, which are Windows binaries.")
	fmt.Fprintln(os.Stderr, "  Please build and run this program on a Windows machine.")
	fmt.Fprintln(os.Stderr, "")
	os.Exit(1)
}
