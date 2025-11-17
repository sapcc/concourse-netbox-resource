package app

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/sapcc/concourse-netbox-resource/internal/concourse"
	"github.com/sapcc/concourse-netbox-resource/internal/helper"
)

var (
	UsageIn string = `This command implements the Concourse in interface as a noop. It reads the input, validates it, and outputs the version.

	Example: in /tmp/build/get < request.json
	`
)

func In() {
	var (
		input  concourse.Input
		output concourse.Output
		err    error
	)

	input, err = validateInInput(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("input validation failed: %w", err))
		os.Exit(1)
	}

	outPath := os.Args[1]
	file, err := os.Create(outPath + "/version.json")
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("failed to create output file: %w", err))
		os.Exit(1)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Fprintln(os.Stderr, fmt.Errorf("failed to close file after writing output: %w", err))
			os.Exit(1)
		}
	}()

	output.Version = input.Version

	err = json.NewEncoder(file).Encode(output)
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("failed to write JSON output to %s: %w", (outPath+"/version.json"), err))
	}

	err = json.NewEncoder(os.Stdout).Encode(output)
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("failed to write JSON to stdout: %w", err))
	}
}

func validateInInput(stdin io.Reader) (concourse.Input, error) {
	var (
		inputParsed concourse.Input
		err         error
	)

	if len(os.Args) < 2 {
		return concourse.Input{}, fmt.Errorf("destination path argument is required")
	}

	path := os.Args[1]
	if path != helper.ConcourseOutputPath {
		return concourse.Input{}, fmt.Errorf("invalid destination path: %s", path)
	}

	err = json.NewDecoder(stdin).Decode(&inputParsed)
	if err != nil && err != io.EOF {
		return concourse.Input{}, fmt.Errorf("failed to decode stdin: %w", err)
	}
	return inputParsed, nil
}
