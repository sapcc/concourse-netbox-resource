package resource

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	fr "github.com/tbe/resource-framework/resource"
	"github.com/tbe/resource-framework/test"
)

func TestHandler_Check(t *testing.T) {
	res2 := "{" +
		"\"source\": {" +
		"\"netbox-url\": \"https://netbox.global.cloud.sap\"," +
		"\"netbox-token\": \"" + os.Getenv("NETBOX_TOKEN") + "\"," +
		"\"apod\": \"AP001\"" +
		"}," +
		"\"version\": {}" +
		"}"
	test.AutoTestCheck(t, func() fr.Resource { return NewNetboxResource() }, map[string]test.Case{
		"valid input": {
			Input:  res2,
			Output: `{ "hash": 5.626454937305665e+18 }`,
			Validation: func(t *testing.T, assertions *assert.Assertions, res interface{}) {
				r := res.(*NetboxResource)
				assert.Equal(t, r.source.NetboxUrl, "https://netbox.global.cloud.sap")
			},
		},
	})
}
