package utils

import (
	"path/filepath"
	"runtime"
)

var (
	_, b, _, _ = runtime.Caller(0)
	// ProjectRoot is the absolute path to the project's root directory.
	ProjectRoot = filepath.Join(filepath.Dir(b), "..", "..")
)