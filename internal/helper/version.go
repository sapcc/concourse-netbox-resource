package helper

import (
	"fmt"
	"runtime"
)

var (
	gitVersion string = "unknown"
	gitCommit  string = "unknown"
	buildDate  string = "unknown"
)

func ShowVersionInfo() string {
	return gitVersion
}

func ShowBuildInfo() string {
	return fmt.Sprintf("version: %s\ncommitId: %s\nbuildDate: %s\ngoVersion: %s\ncompiler: %s\nplatform: %s\n", gitVersion, gitCommit, buildDate, runtime.Version(), runtime.Compiler, fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH))
}
