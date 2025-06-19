package netbox

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"testing"

	"github.com/netbox-community/go-netbox/v4"
	"github.com/sapcc/concourse-netbox-resource/internal/concourse"
	"github.com/sapcc/concourse-netbox-resource/internal/helper"
)

var (
	ctx                         context.Context
	ConcourseSourceConfigObject concourse.Input
	fieldInFilter               reflect.Value
	fiFilterString              string
	fiFilterKind                reflect.Kind
	fieldInQuery                reflect.Value
	fiQueryString               string
	fiQueryKind                 reflect.Kind
)

func TestCreateQuery(t *testing.T) {
	tests := []struct {
		name         string
		sourceConfig string
		filterName   string
		fieldName    string
		queryType    string
		wantErr      bool
	}{
		{"validateDeviceSite", helper.ConcourseSourceConfig, "SiteName", "site", "dcimDevice", false},
		{"validateDeviceTag", helper.ConcourseSourceConfig, "Tag", "tag", "dcimDevice", false},
		{"validateDeviceRole", helper.ConcourseSourceConfig, "Role", "role", "dcimDevice", false},
		{"validateDeviceId", helper.ConcourseSourceConfig, "DeviceId", "id", "dcimDevice", false},
		{"validateDeviceName", helper.ConcourseSourceConfig, "DeviceName", "nameIc", "dcimDevice", false},
		{"validateDeviceType", helper.ConcourseSourceConfig, "DeviceType", "deviceType", "dcimDevice", false},
		{"validateDeviceStatus", helper.ConcourseSourceConfig, "DeviceStatus", "status", "dcimDevice", false},
		{"validateInterfaceId", helper.ConcourseSourceConfig, "InterfaceId", "id", "dcimInterface", false},
		{"validateInterfaceName", helper.ConcourseSourceConfig, "InterfaceName", "nameIc", "dcimInterface", false},
		{"validateInterfaceEnabled", helper.ConcourseSourceConfig, "Enabled", "enabled", "dcimInterface", false},
		{"validateInterfaceMgmtOnly", helper.ConcourseSourceConfig, "MgmtOnly", "mgmtOnly", "dcimInterface", false},
		{"validateInterfaceConnected", helper.ConcourseSourceConfig, "Connected", "connected", "dcimInterface", false},
		{"validateInterfaceCabled", helper.ConcourseSourceConfig, "Cabled", "cabled", "dcimInterface", false},
		{"validateInterfaceType", helper.ConcourseSourceConfig, "Type", "typeIc", "dcimInterface", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err = json.Unmarshal([]byte(test.sourceConfig), &ConcourseSourceConfigObject)
			if err != nil && err != io.EOF {
				t.Errorf("failed to decode test.sourceConfig: %v", err)
			}
			client = netbox.NewAPIClientFor(ConcourseSourceConfigObject.Source.Url, ConcourseSourceConfigObject.Source.Token)

			switch test.queryType {
			case "dcimDevice":
				query := createDeviceQuery(client, ConcourseSourceConfigObject.Source.Filter, ctx)
				fieldInFilter = reflect.ValueOf(ConcourseSourceConfigObject.Source.Filter).FieldByName(test.filterName)
				fieldInQuery = reflect.ValueOf(query).FieldByName(test.fieldName)
			case "dcimInterface":
				query := createInterfaceQuery(client, ConcourseSourceConfigObject.Source.Filter, ctx)
				fieldInFilter = reflect.ValueOf(ConcourseSourceConfigObject.Source.Filter.ServerInterface).FieldByName(test.filterName)
				fieldInQuery = reflect.ValueOf(query).FieldByName(test.fieldName)
			}

			switch fieldInFilter.Kind() {
			case reflect.Pointer:
				fiFilterString = fmt.Sprintf("%v", fieldInFilter.Elem())
				fiFilterKind = fieldInFilter.Elem().Kind()
			default:
				fiFilterString = fmt.Sprintf("%v", fieldInFilter)
				fiFilterKind = fieldInFilter.Kind()
			}

			fiQueryString = fmt.Sprintf("%v", fieldInQuery.Elem())
			fiQueryKind = fieldInQuery.Kind()

			if !fieldInQuery.IsValid() {
				t.Fatalf("expected %s filter %v, got empty field in query", test.filterName, fieldInFilter)
			}

			if fiFilterString != fiQueryString {
				t.Errorf("expected filter %v kind %v, got %v kind %v in query", fiFilterString, fiFilterKind, fiQueryString, fiQueryKind)
			}
		})
	}
}
