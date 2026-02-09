package netbox

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/netbox-community/go-netbox/v4"
	"github.com/sapcc/concourse-netbox-resource/internal/filter"
)

// queryDevicesFromChangelog queries devices based on changelog events.
// If queryInterfaces is true, it queries dcim.interface events and extracts device IDs.
// If queryInterfaces is false, it queries dcim.device events directly.
func queryDevicesFromChangelog(client *netbox.APIClient, netboxFilter filter.NetboxObject, lastUpdatedGte *time.Time, queryInterfaces bool, parallelQueries int, ctx context.Context) ([]netbox.DeviceWithConfigContext, error) {
	if lastUpdatedGte == nil {
		return nil, fmt.Errorf("lastUpdatedGte is required for changelog-based queries")
	}

	var deviceIds []int32
	var err error

	if queryInterfaces {
		// Get device IDs from interface changelog
		deviceIds, err = getDeviceIdsFromInterfaceChangelog(client, *lastUpdatedGte, ctx)
		if err != nil {
			return nil, fmt.Errorf("error getting device IDs from interface changelog: %w", err)
		}
	} else {
		// Get device IDs from device changelog
		deviceIds, err = getDeviceIdsFromDeviceChangelog(client, *lastUpdatedGte, ctx)
		if err != nil {
			return nil, fmt.Errorf("error getting device IDs from device changelog: %w", err)
		}
	}

	if len(deviceIds) == 0 {
		return []netbox.DeviceWithConfigContext{}, nil
	}

	// Fetch devices by IDs in batches
	devices, err := fetchDevicesByIds(client, netboxFilter, deviceIds, parallelQueries, ctx)
	if err != nil {
		return nil, fmt.Errorf("error fetching devices by IDs: %w", err)
	}

	return devices, nil
}

// fetchDevicesByIds fetches devices by their IDs, applying additional filters and batching for performance.
func fetchDevicesByIds(client *netbox.APIClient, netboxFilter filter.NetboxObject, deviceIds []int32, parallelQueries int, ctx context.Context) ([]netbox.DeviceWithConfigContext, error) {
	if len(deviceIds) == 0 {
		return []netbox.DeviceWithConfigContext{}, nil
	}

	// Batch device IDs into chunks to avoid overwhelming the server
	batchSize := 512
	var batches [][]int32
	for i := 0; i < len(deviceIds); i += batchSize {
		end := min(i+batchSize, len(deviceIds))
		batches = append(batches, deviceIds[i:end])
	}

	type batchResult struct {
		devices  []netbox.DeviceWithConfigContext
		err      error
		batchIdx int
	}

	semaphore := make(chan struct{}, parallelQueries)
	resultsChan := make(chan batchResult, len(batches))

	var wg sync.WaitGroup
	for idx, batchIds := range batches {
		wg.Add(1)
		go func(idx int, batchIds []int32) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			var batchDevices []netbox.DeviceWithConfigContext
			limit := int32(512)
			offset := int32(0)

			for {
				// Create query with device IDs and apply other filters
				query := client.DcimAPI.DcimDevicesList(ctx).Id(batchIds).Limit(limit).Offset(offset)

				// Apply additional filters
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
				if len(netboxFilter.DeviceName) > 0 {
					query = query.NameIc(netboxFilter.DeviceName)
				}
				if len(netboxFilter.DeviceType) > 0 {
					query = query.DeviceType(netboxFilter.DeviceType)
				}
				if len(netboxFilter.DeviceStatus) > 0 {
					query = query.Status(netboxFilter.DeviceStatus)
				}

				response, _, err := query.Execute()
				if err != nil {
					resultsChan <- batchResult{err: err, batchIdx: idx}
					return
				}

				batchDevices = append(batchDevices, response.Results...)

				if !response.Next.IsSet() || response.Next.Get() == nil || *response.Next.Get() == "" || len(response.Results) == 0 {
					break
				}
				offset += limit
			}

			resultsChan <- batchResult{devices: batchDevices, batchIdx: idx}
		}(idx, batchIds)
	}

	// Wait for all goroutines to complete and close the results channel
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	allDevices := make([]netbox.DeviceWithConfigContext, 0, len(deviceIds))
	for result := range resultsChan {
		if result.err != nil {
			return nil, fmt.Errorf("error fetching devices (batch %d): %w", result.batchIdx, result.err)
		}
		allDevices = append(allDevices, result.devices...)
	}

	return allDevices, nil
}

// getDeviceIdsFromInterfaceChangelog queries the NetBox changelog for interface changes
// since a certain time and returns the unique device IDs affected by those changes.
func getDeviceIdsFromInterfaceChangelog(client *netbox.APIClient, sinceTime time.Time, ctx context.Context) ([]int32, error) {
	deviceIdSet := make(map[int32]struct{})

	limit := int32(512)
	offset := int32(0)

	for {
		query := client.CoreAPI.CoreObjectChangesList(ctx).
			ChangedObjectType("dcim.interface").
			TimeAfter(sinceTime).
			Limit(limit).
			Offset(offset)

		response, _, err := query.Execute()
		if err != nil {
			return nil, fmt.Errorf("error querying interface changelog: %w", err)
		}

		for _, change := range response.Results {
			// Extract device ID from PostchangeData or PrechangeData
			deviceId := extractDeviceIdFromInterfaceChange(change)
			if deviceId > 0 {
				deviceIdSet[deviceId] = struct{}{}
			}
		}

		// Check if there are more pages
		if !response.Next.IsSet() || response.Next.Get() == nil || *response.Next.Get() == "" || len(response.Results) == 0 {
			break
		}
		offset += limit
	}

	return slices.Collect(maps.Keys(deviceIdSet)), nil
}

// getDeviceIdsFromDeviceChangelog queries the NetBox changelog for device changes
// since a certain time and returns the unique device IDs.
func getDeviceIdsFromDeviceChangelog(client *netbox.APIClient, sinceTime time.Time, ctx context.Context) ([]int32, error) {
	deviceIdSet := make(map[int32]struct{})

	limit := int32(512)
	offset := int32(0)

	for {
		query := client.CoreAPI.CoreObjectChangesList(ctx).
			ChangedObjectType("dcim.device").
			TimeAfter(sinceTime).
			Limit(limit).
			Offset(offset)

		response, _, err := query.Execute()
		if err != nil {
			return nil, fmt.Errorf("error querying device changelog: %w", err)
		}

		for _, change := range response.Results {
			// For device changes, the ChangedObjectId is the device ID
			deviceId := int32(change.GetChangedObjectId())
			if deviceId > 0 {
				deviceIdSet[deviceId] = struct{}{}
			}
		}

		// Check if there are more pages
		if !response.Next.IsSet() || response.Next.Get() == nil || *response.Next.Get() == "" || len(response.Results) == 0 {
			break
		}
		offset += limit
	}

	return slices.Collect(maps.Keys(deviceIdSet)), nil
}

// extractDeviceIdFromInterfaceChange extracts the device ID from an ObjectChange's data.
// For interface changes, the device ID can be found in:
// 1. ChangedObject (the interface object itself with device reference)
// 2. PostchangeData (for create/update - contains the interface data)
// 3. PrechangeData (for delete - contains the interface data before deletion)
func extractDeviceIdFromInterfaceChange(change netbox.ObjectChange) int32 {
	// Try ChangedObject first - this is the actual interface object
	if change.HasChangedObject() {
		changedObj := change.GetChangedObject()
		if deviceId := extractDeviceIdFromMap(changedObj); deviceId > 0 {
			return deviceId
		}
	}

	// Try PostchangeData (for create/update)
	if change.HasPostchangeData() {
		postData := change.GetPostchangeData()
		if deviceId := extractDeviceIdFromMap(postData); deviceId > 0 {
			return deviceId
		}
	}

	// Try PrechangeData (for delete operations)
	if change.HasPrechangeData() {
		preData := change.GetPrechangeData()
		if deviceId := extractDeviceIdFromMap(preData); deviceId > 0 {
			return deviceId
		}
	}

	return 0
}

// extractDeviceIdFromMap extracts the device ID from a changelog data map.
// The interface data contains a "device" field with the device ID.
func extractDeviceIdFromMap(data any) int32 {
	dataMap, ok := data.(map[string]any)
	if !ok {
		return 0
	}

	// Interface changes have a "device" field containing the device ID
	deviceField, exists := dataMap["device"]
	if !exists {
		return 0
	}

	// Handle different possible formats
	switch v := deviceField.(type) {
	case float64:
		return int32(v)
	case int:
		return int32(v)
	case int32:
		return v
	case int64:
		return int32(v)
	case map[string]any:
		// If device is an object with an "id" field
		if id, ok := v["id"]; ok {
			switch idVal := id.(type) {
			case float64:
				return int32(idVal)
			case int:
				return int32(idVal)
			case int32:
				return idVal
			case int64:
				return int32(idVal)
			}
		}
	}

	return 0
}
