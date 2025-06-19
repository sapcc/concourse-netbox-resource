package netbox

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/netbox-community/go-netbox/v4"
	"github.com/sapcc/concourse-netbox-resource/internal/concourse"
	"github.com/sapcc/concourse-netbox-resource/internal/filter"
)

var (
	client          *netbox.APIClient
	netboxFilter    filter.NetboxObject
	lastUpdatedTime *time.Time
	referenceTime   time.Time
	output          []concourse.Version
	err             error
)

func Query(input concourse.Input, ctx context.Context) ([]concourse.Version, error) {
	var (
		deviceList []netbox.DeviceWithConfigContext
	)

	netboxFilter = input.Source.Filter
	client = netbox.NewAPIClientFor(input.Source.Url, input.Source.Token)

	deviceList, err = runPagedDeviceQuery(client, netboxFilter, ctx)
	if err != nil {
		return nil, fmt.Errorf("error during device query: %w", err)
	}

	output, err = fetchDetailsFromDeviceList(input, deviceList, ctx)
	if err != nil {
		return nil, fmt.Errorf("error during device details query: %w", err)
	}
	return output, nil
}

func createDeviceQuery(client *netbox.APIClient, netboxFilter filter.NetboxObject, ctx context.Context) netbox.ApiDcimDevicesListRequest {
	query := client.DcimAPI.DcimDevicesList(ctx)
	if len(netboxFilter.SiteName) > 0 {
		query = query.Site(netboxFilter.SiteName)
	}
	if len(netboxFilter.Tag) > 0 {
		query = query.Tag(netboxFilter.Tag)
	}
	if len(netboxFilter.Role) > 0 {
		query = query.Role(netboxFilter.Role)
	}
	if len(netboxFilter.DeviceId) > 0 {
		query = query.Id(netboxFilter.DeviceId)
	}
	if len(netboxFilter.DeviceName) > 0 {
		query = query.NameIc(netboxFilter.DeviceName)
	}
	if len(netboxFilter.DeviceType) > 0 {
		query = query.DeviceType(netboxFilter.DeviceType)
	}
	if len(netboxFilter.DeviceStatus) > 0 {
		query = query.Status(netboxFilter.DeviceStatus)
	}
	return query
}

func createInterfaceQuery(client *netbox.APIClient, netboxFilter filter.NetboxObject, ctx context.Context) netbox.ApiDcimInterfacesListRequest {
	query := client.DcimAPI.DcimInterfacesList(ctx)
	if len(netboxFilter.ServerInterface.InterfaceId) > 0 {
		query = query.Id(netboxFilter.ServerInterface.InterfaceId)
	}
	if len(netboxFilter.ServerInterface.InterfaceName) > 0 {
		query = query.NameIc(netboxFilter.ServerInterface.InterfaceName)
	}
	if netboxFilter.ServerInterface.Enabled != nil {
		query = query.Enabled(*netboxFilter.ServerInterface.Enabled)
	}
	if netboxFilter.ServerInterface.MgmtOnly != nil {
		query = query.MgmtOnly(*netboxFilter.ServerInterface.MgmtOnly)
	}
	if netboxFilter.ServerInterface.Connected != nil {
		query = query.Connected(*netboxFilter.ServerInterface.Connected)
	}
	if netboxFilter.ServerInterface.Cabled != nil {
		query = query.Cabled(*netboxFilter.ServerInterface.Cabled)
	}
	if len(netboxFilter.ServerInterface.Type) > 0 {
		query = query.TypeIc(netboxFilter.ServerInterface.Type)
	}
	return query
}

func runPagedDeviceQuery(client *netbox.APIClient, netboxFilter filter.NetboxObject, ctx context.Context) ([]netbox.DeviceWithConfigContext, error) {
	deviceList := make([]netbox.DeviceWithConfigContext, 0, 25)
	limit := int32(25)
	offset := int32(0)
	for {
		pagedQuery := createDeviceQuery(client, netboxFilter, ctx).Limit(limit).Offset(offset)
		deviceQueryResponse, _, err := pagedQuery.Execute()
		if err != nil {
			return nil, fmt.Errorf("error during DcimDevicesList query: %w", err)
		}
		deviceList = append(deviceList, deviceQueryResponse.Results...)
		if !deviceQueryResponse.Next.IsSet() || deviceQueryResponse.Next.Get() == nil || *deviceQueryResponse.Next.Get() == "" || len(deviceQueryResponse.Results) == 0 {
			break
		}
		offset += limit
	}
	return deviceList, nil
}

func runPagedInterfaceQuery(client *netbox.APIClient, netboxFilter filter.NetboxObject, deviceId int32, ctx context.Context) ([]netbox.Interface, error) {
	interfaceList := make([]netbox.Interface, 0, 25)
	limit := int32(25)
	offset := int32(0)
	for {
		pagedQuery := createInterfaceQuery(client, netboxFilter, ctx).DeviceId([]int32{deviceId}).Limit(limit).Offset(offset)
		interfaceQueryResponse, _, err := pagedQuery.Execute()
		if err != nil {
			return nil, fmt.Errorf("error during DcimInterfacesList query: %w", err)
		}
		interfaceList = append(interfaceList, interfaceQueryResponse.Results...)
		if !interfaceQueryResponse.Next.IsSet() || interfaceQueryResponse.Next.Get() == nil || *interfaceQueryResponse.Next.Get() == "" || len(interfaceQueryResponse.Results) == 0 {
			break
		}
		offset += limit
	}
	return interfaceList, nil
}

func fetchDetailsFromDeviceList(input concourse.Input, deviceList []netbox.DeviceWithConfigContext, ctx context.Context) ([]concourse.Version, error) {
	var (
		interfaceList []netbox.Interface
	)
	output = make([]concourse.Version, 0, len(deviceList))

	for _, d := range deviceList {
		name := ""
		if d.Name.IsSet() && d.Name.Get() != nil {
			name = *d.Name.Get()
		}
		if serverInterfaceOptionIsSet(d, netboxFilter) {
			interfaceList, err = runPagedInterfaceQuery(client, netboxFilter, d.Id, ctx)
			if err != nil {
				return nil, fmt.Errorf("error during server interface query: %w", err)
			}

			output, err = populateInterfaceDetails(name, input, d, interfaceList)
			if err != nil {
				return nil, fmt.Errorf("error during server interface details query: %w", err)
			}
		} else {
			output, err = populateDeviceDetails(name, input, d)
			if err != nil {
				return nil, fmt.Errorf("error during device details query: %w", err)
			}
		}
	}
	// Sort the output by LastUpdated in ascending order
	slices.SortStableFunc(output, func(a, b concourse.Version) int {
		return strings.Compare(a.LastUpdated, b.LastUpdated)
	})
	return output, nil
}

func populateInterfaceDetails(name string, input concourse.Input, device netbox.DeviceWithConfigContext, interfaceList []netbox.Interface) ([]concourse.Version, error) {
	for _, iface := range interfaceList {
		lastUpdatedTime, referenceTime, err = getTimestamps(device, input)
		if err != nil {
			return nil, fmt.Errorf("error parsing netbox timestamps because of: %w", err)
		}

		if lastUpdatedTime.UTC().After(referenceTime) {
			configContext := ""
			if input.Source.Filter.GetConfigContext != nil && *input.Source.Filter.GetConfigContext {
				if device.HasConfigContext() {
					configContextData := device.GetConfigContext()
					if configContextBytes, err := json.Marshal(configContextData); err == nil {
						configContext = string(configContextBytes)
					}
				}
			}

			deviceDisplayUrl := ""
			if device.DisplayUrl != nil {
				deviceDisplayUrl = *device.DisplayUrl
			}

			interfaceDisplayUrl := ""
			if iface.DisplayUrl != nil {
				interfaceDisplayUrl = *iface.DisplayUrl
			}

			output = append(output, concourse.Version{
				Id:                  fmt.Sprintf("%d", iface.Id),
				LastUpdated:         lastUpdatedTime.Format(time.RFC3339),
				ObjectType:          "interfaces",
				DeviceId:            fmt.Sprintf("%d", device.Id),
				DeviceName:          name,
				DeviceRole:          device.Role.GetSlug(),
				DeviceApiUrl:        device.Url,
				DeviceDisplayUrl:    deviceDisplayUrl,
				ConfigContext:       configContext,
				InterfaceName:       iface.Name,
				InterfaceType:       string(iface.Type.GetLabel()),
				InterfaceApiUrl:     iface.Url,
				InterfaceDisplayUrl: interfaceDisplayUrl,
			})
		}
	}
	return output, nil
}

func populateDeviceDetails(name string, input concourse.Input, device netbox.DeviceWithConfigContext) ([]concourse.Version, error) {
	lastUpdatedTime, referenceTime, err = getTimestamps(device, input)
	if err != nil {
		return nil, fmt.Errorf("error parsing netbox timestamps because of: %w", err)
	}

	if lastUpdatedTime.UTC().After(referenceTime) {
		configContext := ""
		if input.Source.Filter.GetConfigContext != nil && *input.Source.Filter.GetConfigContext {
			if device.HasConfigContext() {
				configContextData := device.GetConfigContext()
				if configContextBytes, err := json.Marshal(configContextData); err == nil {
					configContext = string(configContextBytes)
				}
			}
		}

		displayUrl := ""
		if device.DisplayUrl != nil {
			displayUrl = *device.DisplayUrl
		}

		output = append(output, concourse.Version{
			Id:               fmt.Sprintf("%d", device.Id),
			LastUpdated:      lastUpdatedTime.Format(time.RFC3339),
			ObjectType:       "devices",
			DeviceName:       name,
			DeviceRole:       device.Role.GetSlug(),
			DeviceApiUrl:     device.Url,
			DeviceDisplayUrl: displayUrl,
			ConfigContext:    configContext,
		})
	}
	return output, nil
}

func serverInterfaceOptionIsSet(device netbox.DeviceWithConfigContext, netboxFilter filter.NetboxObject) bool {
	sIf := netboxFilter.ServerInterface
	if device.Role.GetSlug() == "server" && (len(sIf.InterfaceId) > 0 ||
		len(sIf.InterfaceName) > 0 ||
		sIf.Enabled != nil ||
		sIf.MgmtOnly != nil ||
		sIf.Connected != nil ||
		sIf.Cabled != nil ||
		len(sIf.Type) > 0) {
		return true
	}
	return false
}

func getReferenceTime(lastUpdated string) (time.Time, error) {
	var (
		referenceTime time.Time
		err           error
	)

	if lastUpdated == "" {
		return time.Time{}, nil
	}

	referenceTime, err = time.Parse(time.RFC3339, lastUpdated)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid reference time format found in 'version.last_updated': %w", err)
	} else {
		return referenceTime.UTC(), nil
	}
}

func getTimestamps(device any, input concourse.Input) (*time.Time, time.Time, error) {
	switch device := device.(type) {
	case netbox.DeviceWithConfigContext:
		lastUpdatedTime = device.LastUpdated.Get()
	case netbox.Interface:
		lastUpdatedTime = device.LastUpdated.Get()
	default:
		return &time.Time{}, time.Time{}, fmt.Errorf("unexpected device type in getTimestamps: %T", device)
	}
	referenceTime, err = getReferenceTime(input.Version.LastUpdated)
	if err != nil {
		return &time.Time{}, time.Time{}, fmt.Errorf("error parsing netbox lastupdated timestamp because of: %w", err)
	}
	return lastUpdatedTime, referenceTime, nil
}
