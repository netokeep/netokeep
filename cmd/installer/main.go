package main

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

//go:embed nk
var nkBin []byte

//go:embed nks
var nksBin []byte

func main() {
	// Determine the binary name based on OS
}
