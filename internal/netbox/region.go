package netbox

import (
	"context"
	"fmt"

	"github.com/netbox-community/go-netbox/v4"
	"github.com/sapcc/concourse-netbox-resource/internal/filter"
)

// getUniqueRegions extracts unique region slugs from the site cache
// If filter has specific regions, use those; otherwise extract from sites
func getUniqueRegions(cache map[int32]string, netboxFilter filter.NetboxObject) []string {
	if len(netboxFilter.RegionName) > 0 {
		return netboxFilter.RegionName
	}

	regionSet := make(map[string]bool)
	for _, region := range cache {
		if region != "" {
			regionSet[region] = true
		}
	}

	regions := make([]string, 0, len(regionSet))
	for region := range regionSet {
		regions = append(regions, region)
	}
	return regions
}

func fetchSiteRegions(client *netbox.APIClient, netboxFilter filter.NetboxObject, ctx context.Context) (map[int32]string, error) {
	cache := make(map[int32]string)
	if client == nil {
		return cache, nil
	}

	limit := int32(100)
	offset := int32(0)
	for {
		query := client.DcimAPI.DcimSitesList(ctx).Limit(limit).Offset(offset)
		if len(netboxFilter.RegionName) > 0 {
			query = query.Region(netboxFilter.RegionName)
		}
		siteResponse, _, err := query.Execute()
		if err != nil {
			return nil, fmt.Errorf("error during DcimSitesList query: %w", err)
		}
		for _, site := range siteResponse.Results {
			region := ""
			if site.Region.IsSet() && site.Region.Get() != nil {
				region = site.Region.Get().GetSlug()
			}
			cache[site.Id] = region
		}
		if !siteResponse.Next.IsSet() || siteResponse.Next.Get() == nil || *siteResponse.Next.Get() == "" || len(siteResponse.Results) == 0 {
			break
		}
		offset += limit
	}
	return cache, nil
}

func getSiteRegion(siteId int32) string {
	if siteRegionCache == nil {
		return ""
	}
	if region, ok := siteRegionCache[siteId]; ok {
		return region
	}
	return ""
}
