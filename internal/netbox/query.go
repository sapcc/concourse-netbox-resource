package netbox

import (
	"context"
	"fmt"
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
	siteRegionCache map[int32]string
)

func Query(input concourse.Input, ctx context.Context) ([]concourse.Version, error) {
	var (
		deviceList     []netbox.DeviceWithConfigContext
		err            error
		lastUpdatedGte *time.Time
	)

	netboxFilter = input.Source.Filter
	client = netbox.NewAPIClientFor(input.Source.Url, input.Source.Token)

	// Parse lastUpdated timestamp for filtering
	// If not provided, use initial_lookback or epoch 0
	parsedTime, err := getReferenceTime(input)
	if err != nil {
		return nil, fmt.Errorf("error parsing reference time: %w", err)
	}
	lastUpdatedGte = &parsedTime

	// Determine parallelism - default to 1 if not specified
	parallelQueries := getParallelQueries(input.Source.ParallelQueries)

	// First fetch site regions - this is fast and gives us region list
	siteRegionCache, err = fetchSiteRegions(client, netboxFilter, ctx)
	if err != nil {
		return nil, fmt.Errorf("error during site region query: %w", err)
	}

	// Check if changelog-based query mode is enabled
	if useChangelog(input) {
		// Changelog-based query: find devices from changelog events
		// If interface filters are set, query interface change events
		// Otherwise, query device change events
		queryInterfaces := interfaceFilterIsSet(netboxFilter)
		deviceList, err = queryDevicesFromChangelog(client, netboxFilter, lastUpdatedGte, queryInterfaces, parallelQueries, ctx)
		if err != nil {
			return nil, fmt.Errorf("error during changelog-based device query: %w", err)
		}
	} else {
		// Classic query mode
		// Determine if we're querying interfaces
		// If so, we need all devices (no lastUpdatedGte) but filter interfaces by lastUpdatedGte
		// If not, we filter devices by lastUpdatedGte
		interfaceQueryMode := interfaceFilterIsSet(netboxFilter)
		var deviceLastUpdatedGte *time.Time
		if !interfaceQueryMode {
			deviceLastUpdatedGte = lastUpdatedGte
		}

		// Get unique regions from the site cache
		regions := getUniqueRegions(siteRegionCache, netboxFilter)

		// Run device queries per region with configured parallelism
		deviceList, err = runDeviceQueriesByRegion(client, netboxFilter, deviceLastUpdatedGte, regions, parallelQueries, ctx)
		if err != nil {
			return nil, fmt.Errorf("error during device query: %w", err)
		}
	}

	output, err = fetchDetailsFromDeviceList(input, deviceList, lastUpdatedGte, parallelQueries, ctx)
	if err != nil {
		return nil, fmt.Errorf("error during device details query: %w", err)
	}
	return output, nil
}

// getParallelQueries returns the number of parallel queries to use.
// Defaults to 1 if not specified or invalid.
func getParallelQueries(configured int) int {
	if configured <= 0 {
		return 1
	}
	return configured
}

func useChangelog(input concourse.Input) bool {
	useChangelog := input.Source.UseChangelog != nil && *input.Source.UseChangelog
	return useChangelog
}
