package resource

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	fr "github.com/tbe/resource-framework/resource"
	"github.com/tbe/resource-framework/test"
)

func TestHandler_In(t *testing.T) {
	res2 := "{" +
		"\"source\": {" +
		"\"netbox-url\": \"https://netbox.global.cloud.sap\"," +
		"\"netbox-token\": \"" + os.Getenv("NETBOX_TOKEN") + "\"," +
		"\"apod\": \"AP001\"" +
		"}," +
		"\"version\": { \"hash\": 123123123123 }," +
		"\"params\": {}" +
		"}"
	test.AutoTestIn(t, func() fr.Resource { return NewNetboxResource() }, map[string]test.Case{
		"valid input": {
			Input:  res2,
			Output: `{ "metadata": null, "version": {"hash": 1.23123123123e+11} }`,
			Validation: func(t *testing.T, assertions *assert.Assertions, res interface{}) {
				r := res.(*NetboxResource)
				assert.Equal(t, "https://netbox.global.cloud.sap", r.source.NetboxUrl)
			},
		},
	})
}
