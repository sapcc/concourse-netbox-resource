package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/sapcc/concourse-netbox-resource/internal/concourse"
	"github.com/sapcc/concourse-netbox-resource/internal/netbox"
)

var (
	UsageCheck string = `This command queries the NetBox API for objects matching the specified filter and returns their latest versions.
The input is required in JSON format from stdin, which should include the NetBox URL, the API token and optional filters.
The output will be a JSON array of objects with their latest versions.

{
  "source": {
    "url": "https://netbox.example.local",
    "token": "your-api-token"
		},
		"filter": {
			"site_name": ["My Site"],
			"tag": ["my-tag"],
			"role": ["server"],
			"device_id": [123],
			"device_name": ["my-server"],
			"device_type": ["server type"],
			"device_status": ["active"],
			"get_config_context": true,
			"server_interface": {
				"interface_id": [456],
				"interface_name": ["eth0"],
				"enabled": true,
				"mgmt_only": false,
				"connected": true,
				"cabled": true,
				"type": ["1000base-t"]
			}
	},
  "version": {
    "id": 123,
		"last_updated": "2023-10-01T12:00:00Z",
		"object_type": "interfaces",
		"device_id": 123,
		"device_name": "my-server",
		"device_role": "server",
		"interface_name": "eth0",
		"interface_type": "1000base-t"
  }
}

Example: check < source.json
Parameters:
`
)

func Check() {
	var (
		input  concourse.Input
		output []concourse.Version
		err    error
	)

	input, err = validateCheckInput(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("input validation failed: %w", err))
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	output, err = netbox.Query(input, ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("netbox query failed: %w", err))
		os.Exit(1)
	}

	if err := json.NewEncoder(os.Stdout).Encode(output); err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("failed to write JSON to stdout: %w", err))
		os.Exit(1)
	}
}

func validateCheckInput(stdin io.Reader) (concourse.Input, error) {
	var (
		inputParsed concourse.Input
		err         error
	)

	err = json.NewDecoder(stdin).Decode(&inputParsed)
	if err != nil && err != io.EOF {
		return concourse.Input{}, fmt.Errorf("failed to decode stdin: %w", err)
	}

	if inputParsed.Source.Url == "" {
		return concourse.Input{}, fmt.Errorf("source.url containing the NetBox URL is required")
	}
	return inputParsed, nil
}
