package app

import (
	"strings"
	"testing"

	"github.com/sapcc/concourse-netbox-resource/internal/helper"
)

func TestValidateCheckInput(t *testing.T) {
	tests := []struct {
		name    string
		stdin   string
		wantErr bool
	}{
		{"noInput", "", true},
		{"missingSourceUrl", helper.ConcourseInvalidSourceConfig, true},
		{"validInput", helper.ConcourseSourceConfig, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stdinReader := strings.NewReader(test.stdin)

			_, err := validateCheckInput(stdinReader)
			if (err != nil) != test.wantErr {
				t.Errorf("validateInput() error: '%v', error expected: %v", err, test.wantErr)
			}
		})
	}
}
