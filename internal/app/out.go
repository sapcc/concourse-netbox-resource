package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/sapcc/concourse-netbox-resource/internal/concourse"
	"github.com/sapcc/concourse-netbox-resource/internal/helper"
)

var (
	UsageOut string = `This command implements the Concourse out interface as a noop. It reads the input, validates it, and outputs the version.

	Example: out /tmp/build/put < source.json
	`
)

func Out() {
	var (
		input  concourse.Input
		output concourse.Output
		err    error
	)

	input, _, err = validateOutInput(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("input validation failed: %w", err))
		os.Exit(1)
	}

	output.Version = input.Version
	if err := json.NewEncoder(os.Stdout).Encode(output); err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("failed to write JSON to stdout: %w", err))
	}
}

func validateOutInput(stdin io.Reader) (concourse.Input, concourse.Input, error) {
	var (
		sourceParsed  concourse.Input
		versionParsed concourse.Input
		err           error
	)

	if len(os.Args) < 2 {
		return concourse.Input{}, concourse.Input{}, fmt.Errorf("destination path argument is required")
	}

	path := os.Args[1]
	if path != helper.ConcourseInputPath {
		return concourse.Input{}, concourse.Input{}, fmt.Errorf("invalid source path: %s", path)
	}

	file, err := os.ReadFile(path + "/version.json")
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("failed to read input file %s: %w", path+"/version.json", err))
		os.Exit(1)
	}

	err = json.NewDecoder(bytes.NewReader(file)).Decode(&versionParsed)
	if err != nil && err != io.EOF {
		return concourse.Input{}, concourse.Input{}, fmt.Errorf("failed to decode version from %s: %w", path+"/version.json", err)
	}

	err = json.NewDecoder(stdin).Decode(&sourceParsed)
	if err != nil && err != io.EOF {
		return concourse.Input{}, concourse.Input{}, fmt.Errorf("failed to decode stdin: %w", err)
	}

	if sourceParsed.Source.Url == "" {
		return concourse.Input{}, concourse.Input{}, fmt.Errorf("source.url containing the NetBox URL is required")
	}
	return sourceParsed, versionParsed, nil
}
