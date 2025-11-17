package app

import (
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/sapcc/concourse-netbox-resource/internal/helper"
)

func TestValidateInInput(t *testing.T) {
	tests := []struct {
		name    string
		stdin   string
		osArg   string
		wantErr bool
	}{
		{"noInput", "", "", true},
		{"missingOutputPath", helper.ConcourseVersion, "", true},
		{"validInput", helper.ConcourseVersion, helper.ConcourseOutputPath, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			os.Args[1] = test.osArg
			flag.Parse()

			stdinReader := strings.NewReader(test.stdin)
			_, err := validateInInput(stdinReader)
			if (err != nil) != test.wantErr {
				t.Errorf("validateInput() error: '%v', error expected: %v", err, test.wantErr)
			}
		})
	}
}
