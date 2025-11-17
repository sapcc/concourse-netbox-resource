package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sapcc/concourse-netbox-resource/internal/app"
	"github.com/sapcc/concourse-netbox-resource/internal/helper"
)

func main() {
	var (
		cmdlineflags   helper.CmdLineFlags
		executableName string
	)

	cmdlineflags = helper.AddFlags()
	if cmdlineflags.Buildinfo {
		if _, writeErr := os.Stdout.WriteString(helper.ShowBuildInfo()); writeErr != nil {
			fmt.Fprintln(os.Stderr, fmt.Errorf("failed to query build info: %w", writeErr))
			os.Exit(1)
		}
		os.Exit(0)
	}

	if cmdlineflags.Versioninfo {
		if _, writeErr := os.Stdout.WriteString(helper.ShowVersionInfo()); writeErr != nil {
			fmt.Fprintln(os.Stderr, fmt.Errorf("failed to query version info: %w", writeErr))
			os.Exit(1)
		}
		os.Exit(0)
	}

	if len(cmdlineflags.Command) > 0 {
		executableName = cmdlineflags.Command
	} else {
		executablePath := os.Args[0]
		executableName = filepath.Base(executablePath)
	}

	switch executableName {
	case "check":
		cmdlineHelp(cmdlineflags, app.UsageCheck)
		app.Check()
	case "in":
		cmdlineHelp(cmdlineflags, app.UsageIn)
		app.In()
	case "out":
		cmdlineHelp(cmdlineflags, app.UsageOut)
		app.Out()
	default:
		fmt.Fprintf(os.Stderr, "unknown executable name: %s\n", executableName)
		os.Exit(1)
	}
}

func cmdlineHelp(cmdlineflags helper.CmdLineFlags, cmdHelp string) {
	if cmdlineflags.Help {
		if _, writeErr := os.Stdout.WriteString(cmdHelp); writeErr != nil {
			fmt.Fprintln(os.Stderr, fmt.Errorf("failed print usage info: %w", writeErr))
			os.Exit(1)
		}
		flag.PrintDefaults()
		os.Exit(0)
	}
}
