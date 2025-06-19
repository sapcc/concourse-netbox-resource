package app

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"os"
	"testing"

	"github.com/sapcc/concourse-netbox-resource/internal/concourse"
	"github.com/sapcc/concourse-netbox-resource/internal/helper"
)

func TestValidateOutInput(t *testing.T) {
	var (
		sourceConfig      concourse.Input
		fetchedVersion    concourse.Input
		sourceConfigBytes []byte
		err               error
	)

	tests := []struct {
		name    string
		stdin   string
		osArg   string
		wantErr bool
	}{
		{"missingInputPath", helper.ConcourseSourceConfig, "", true},
		{"missingSourceUrl", helper.ConcourseInvalidSourceConfig, helper.ConcourseInputPath, true},
		{"validInput", helper.ConcourseSourceConfig, helper.ConcourseInputPath, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err = json.Unmarshal([]byte(test.stdin), &sourceConfig)
			if err != nil && err != io.EOF {
				t.Errorf("failed to decode test.input: %v", err)
			}

			sourceConfigBytes, err = json.Marshal(sourceConfig)
			if err != nil && err != io.EOF {
				t.Errorf("failed to encode test.input: %v", err)
			}

			if len(test.osArg) > 0 {
				filepath := test.osArg + "/version.json"

				err = helper.EnsureFolder(test.osArg)
				if err != nil {
					t.Errorf("error creating folder %s: '%v', error expected: %v", test.osArg, err, test.wantErr)
				}

				file, err := os.Create(filepath)
				if err != nil {
					t.Errorf("failed to create %s file: %v", filepath, err)
				}
				defer func() {
					if err := file.Close(); err != nil {
						t.Fatal(err)
					}
				}()

				err = json.Unmarshal([]byte(helper.ConcourseVersion), &fetchedVersion)
				if err != nil && err != io.EOF {
					t.Errorf("failed to decode stdin: %v", err)
				}

				if err := json.NewEncoder(file).Encode(fetchedVersion); err != nil {
					t.Errorf("failed to write JSON to %s: %v", filepath, err)
				}

				os.Args[1] = test.osArg
				flag.Parse()
			}

			stdinReader := bytes.NewReader(sourceConfigBytes)
			_, _, err := validateOutInput(stdinReader)
			if (err != nil) != test.wantErr {
				t.Errorf("validateInput() error: '%v', error expected: %v", err, test.wantErr)
			}
		})
	}
}
