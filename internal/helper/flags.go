package helper

import (
	"flag"
)

type CmdLineFlags struct {
	Command     string
	Versioninfo bool
	Buildinfo   bool
	Help        bool
}

var (
	flags CmdLineFlags
)

func AddFlags() CmdLineFlags {
	flag.StringVar(&flags.Command, "c", "", "[check|in|out] command to execute")
	flag.BoolVar(&flags.Versioninfo, "v", false, "return program version")
	flag.BoolVar(&flags.Buildinfo, "b", false, "return build information")
	flag.BoolVar(&flags.Help, "h", false, "return this help message")
	flag.Parse()
	return flags
}
