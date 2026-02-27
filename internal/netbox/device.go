package netbox

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/netbox-community/go-netbox/v4"
	"github.com/sapcc/concourse-netbox-resource/internal/concourse"
	"github.com/sapcc/concourse-netbox-resource/internal/filter"
)

// runDeviceQueriesByRegion runs device queries for each region with limited parallelism
func runDeviceQueriesByRegion(client *netbox.APIClient, netboxFilter filter.NetboxObject, lastUpdatedGte *time.Time, regions []string, parallelQueries int, ctx context.Context) ([]netbox.DeviceWithConfigContext, error) {
	type regionResult struct {
		devices []netbox.DeviceWithConfigContext
		err     error
		region  string
	}

	semaphore := make(chan struct{}, parallelQueries)
	resultsChan := make(chan regionResult, len(regions))

	var wg sync.WaitGroup
	for _, region := range regions {
		wg.Add(1)
		go func(region string) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			// Create a copy of the filter with the specific region
			regionFilter := netboxFilter
			regionFilter.RegionName = []string{region}

			devices, err := runPagedDeviceQuery(client, regionFilter, lastUpdatedGte, ctx)
			resultsChan <- regionResult{devices: devices, err: err, region: region}
		}(region)
	}

	// Wait for all goroutines to complete and close the results channel
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	allDevices := make([]netbox.DeviceWithConfigContext, 0, 512)
	for result := range resultsChan {
		if result.err != nil {
			return nil, fmt.Errorf("error querying devices for region %s: %w", result.region, result.err)
		}
		allDevices = append(allDevices, result.devices...)
	}

	return allDevices, nil
}

func createDeviceQuery(client *netbox.APIClient, netboxFilter filter.NetboxObject, lastUpdatedGte *time.Time, ctx context.Context) netbox.ApiDcimDevicesListRequest {
	query := client.DcimAPI.DcimDevicesList(ctx)
	if len(netboxFilter.SiteName) > 0 {
		query = query.Site(netboxFilter.SiteName)
	}
	if len(netboxFilter.RegionName) > 0 {
		query = query.Region(netboxFilter.RegionName)
	}
	if len(netboxFilter.Tenant) > 0 {
		query = query.Tenant(netboxFilter.Tenant)
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
	if lastUpdatedGte != nil {
		query = query.LastUpdatedGte([]time.Time{*lastUpdatedGte})
	}
	return query
}

func runPagedDeviceQuery(client *netbox.APIClient, netboxFilter filter.NetboxObject, lastUpdatedGte *time.Time, ctx context.Context) ([]netbox.DeviceWithConfigContext, error) {
	deviceList := make([]netbox.DeviceWithConfigContext, 0, 512)
	limit := int32(512)
	offset := int32(0)
	for {
		pagedQuery := createDeviceQuery(client, netboxFilter, lastUpdatedGte, ctx).Limit(limit).Offset(offset)
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

func fetchDetailsFromDeviceList(input concourse.Input, deviceList []netbox.DeviceWithConfigContext, lastUpdatedGte *time.Time, parallelQueries int, ctx context.Context) ([]concourse.Version, error) {
	output = make([]concourse.Version, 0, len(deviceList))

	// Collect device IDs that need interface queries
	deviceIdsNeedingInterfaces := make([]int32, 0)
	for _, d := range deviceList {
		if serverInterfaceOptionIsSet(d, netboxFilter) {
			deviceIdsNeedingInterfaces = append(deviceIdsNeedingInterfaces, d.Id)
		}
	}

	// Bulk fetch all interfaces at once
	interfacesByDevice, err := runBulkInterfaceQuery(client, netboxFilter, deviceIdsNeedingInterfaces, lastUpdatedGte, parallelQueries, ctx)
	if err != nil {
		return nil, fmt.Errorf("error during bulk interface query: %w", err)
	}

	for _, d := range deviceList {
		name := ""
		if d.Name.IsSet() && d.Name.Get() != nil {
			name = *d.Name.Get()
		}
		if serverInterfaceOptionIsSet(d, netboxFilter) {
			interfaceList, hasInterfaces := interfacesByDevice[d.Id]
			// Skip devices with no changed interfaces
			if !hasInterfaces || len(interfaceList) == 0 {
				continue
			}
			output, err = populateInterfaceDetails(name, input, d, interfaceList, lastUpdatedGte, ctx)
			if err != nil {
				return nil, fmt.Errorf("error during server interface details query: %w", err)
			}
		} else {
			output, err = populateDeviceDetails(name, input, d, lastUpdatedGte, ctx)
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

func populateDeviceDetails(name string, input concourse.Input, device netbox.DeviceWithConfigContext, fallbackReferenceTime *time.Time, ctx context.Context) ([]concourse.Version, error) {
	lastUpdatedTime, referenceTime, err = getTimestamps(device, input, fallbackReferenceTime)
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

		deviceRegion := getSiteRegion(device.Site.GetId())
		deviceTags := getDeviceTags(device)

		output = append(output, concourse.Version{
			Id:               fmt.Sprintf("%d", device.Id),
			LastUpdated:      lastUpdatedTime.Format(time.RFC3339),
			ObjectType:       "devices",
			DeviceName:       name,
			DeviceRole:       device.Role.GetSlug(),
			DeviceSite:       device.Site.GetSlug(),
			DeviceRegion:     deviceRegion,
			DeviceTags:       deviceTags,
			DeviceApiUrl:     device.Url,
			DeviceDisplayUrl: displayUrl,
			ConfigContext:    configContext,
		})
	}
	return output, nil
}

func serverInterfaceOptionIsSet(device netbox.DeviceWithConfigContext, netboxFilter filter.NetboxObject) bool {
	sIf := netboxFilter.ServerInterface
	// Check if interface filters are set
	interfaceFiltersSet := len(sIf.InterfaceId) > 0 ||
		len(sIf.InterfaceName) > 0 ||
		sIf.Enabled != nil ||
		sIf.MgmtOnly != nil ||
		sIf.Connected != nil ||
		sIf.Cabled != nil ||
		len(sIf.Type) > 0

	if !interfaceFiltersSet {
		return false
	}

	// If role filter is specified, check if device role matches any of them
	// If no role filter, apply interface query to all devices
	if len(netboxFilter.Role) > 0 {
		deviceRole := device.Role.GetSlug()
		return slices.Contains(netboxFilter.Role, deviceRole)
	}

	return true
}

func getDeviceTags(device netbox.DeviceWithConfigContext) string {
	tags := device.GetTags()
	if len(tags) == 0 {
		return ""
	}
	tagSlugs := make([]string, 0, len(tags))
	for _, tag := range tags {
		tagSlugs = append(tagSlugs, tag.GetSlug())
	}
	return strings.Join(tagSlugs, ",")
}
