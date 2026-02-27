package netbox

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/netbox-community/go-netbox/v4"
	"github.com/sapcc/concourse-netbox-resource/internal/concourse"
	"github.com/sapcc/concourse-netbox-resource/internal/filter"
)

func createInterfaceQuery(client *netbox.APIClient, netboxFilter filter.NetboxObject, lastUpdatedGte *time.Time, ctx context.Context) netbox.ApiDcimInterfacesListRequest {
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
	if lastUpdatedGte != nil {
		query = query.LastUpdatedGte([]time.Time{*lastUpdatedGte})
	}
	return query
}

func runBulkInterfaceQuery(client *netbox.APIClient, netboxFilter filter.NetboxObject, deviceIds []int32, lastUpdatedGte *time.Time, parallelQueries int, ctx context.Context) (map[int32][]netbox.Interface, error) {
	interfacesByDevice := make(map[int32][]netbox.Interface)
	if len(deviceIds) == 0 {
		return interfacesByDevice, nil
	}

	// Batch device IDs into chunks of 512 to avoid overwhelming the server
	batchSize := 512
	var batches [][]int32
	for i := 0; i < len(deviceIds); i += batchSize {
		end := min(i+batchSize, len(deviceIds))
		batches = append(batches, deviceIds[i:end])
	}

	type batchResult struct {
		interfaces []netbox.Interface
		err        error
		batchIdx   int
	}

	semaphore := make(chan struct{}, parallelQueries)
	resultsChan := make(chan batchResult, len(batches))

	var wg sync.WaitGroup
	for idx, batchDeviceIds := range batches {
		wg.Add(1)
		go func(idx int, batchDeviceIds []int32) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			var batchInterfaces []netbox.Interface
			limit := int32(512)
			offset := int32(0)
			for {
				pagedQuery := createInterfaceQuery(client, netboxFilter, lastUpdatedGte, ctx).DeviceId(batchDeviceIds).Limit(limit).Offset(offset)
				interfaceQueryResponse, _, err := pagedQuery.Execute()
				if err != nil {
					resultsChan <- batchResult{err: err, batchIdx: idx}
					return
				}
				batchInterfaces = append(batchInterfaces, interfaceQueryResponse.Results...)
				if !interfaceQueryResponse.Next.IsSet() || interfaceQueryResponse.Next.Get() == nil || *interfaceQueryResponse.Next.Get() == "" || len(interfaceQueryResponse.Results) == 0 {
					break
				}
				offset += limit
			}
			resultsChan <- batchResult{interfaces: batchInterfaces, batchIdx: idx}
		}(idx, batchDeviceIds)
	}

	// Wait for all goroutines to complete and close the results channel
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	for result := range resultsChan {
		if result.err != nil {
			return nil, fmt.Errorf("error during DcimInterfacesList query (batch %d): %w", result.batchIdx, result.err)
		}
		for _, iface := range result.interfaces {
			deviceId := iface.Device.GetId()
			interfacesByDevice[deviceId] = append(interfacesByDevice[deviceId], iface)
		}
	}
	return interfacesByDevice, nil
}

func populateInterfaceDetails(name string, input concourse.Input, device netbox.DeviceWithConfigContext, interfaceList []netbox.Interface, fallbackReferenceTime *time.Time, ctx context.Context) ([]concourse.Version, error) {
	for _, iface := range interfaceList {
		// Use the interface's last_updated timestamp, not the device's
		lastUpdatedTime, referenceTime, err = getTimestamps(iface, input, fallbackReferenceTime)
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

			deviceRegion := getSiteRegion(device.Site.GetId())
			deviceTags := getDeviceTags(device)

			output = append(output, concourse.Version{
				Id:                  fmt.Sprintf("%d", iface.Id),
				LastUpdated:         lastUpdatedTime.Format(time.RFC3339),
				ObjectType:          "interfaces",
				DeviceId:            fmt.Sprintf("%d", device.Id),
				DeviceName:          name,
				DeviceRole:          device.Role.GetSlug(),
				DeviceSite:          device.Site.GetSlug(),
				DeviceRegion:        deviceRegion,
				DeviceTags:          deviceTags,
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

// interfaceFilterIsSet checks if any interface filter options are configured
// Used to determine if we should query interfaces (and skip lastUpdatedGte on devices)
func interfaceFilterIsSet(netboxFilter filter.NetboxObject) bool {
	sIf := netboxFilter.ServerInterface
	return len(sIf.InterfaceId) > 0 ||
		len(sIf.InterfaceName) > 0 ||
		sIf.Enabled != nil ||
		sIf.MgmtOnly != nil ||
		sIf.Connected != nil ||
		sIf.Cabled != nil ||
		len(sIf.Type) > 0
}
