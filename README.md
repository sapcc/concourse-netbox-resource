# concourse-netbox-resource
A concourse resource to trigger from netbox

## Container Image build

This resource is built using a Dockerfile that uses a multi-stage build process. The first stage builds the Go application, and the second stage creates a minimal container image using distroless. The following optional build arguments are available:

- `BUILDER_NAME`: The name of the Go builder image (default: `golang`)
- `BUILDER_VERSION`: The version of the Go builder image (default: `1.25.0-bookworm`)
- `BASE_NAME`: The base image for the final container (default: `gcr.io/distroless/static-debian13`)
- `BASE_VERSION`: The version of the base image (default: `latest`) (`nonroot` is not possible because Concourse [requires root permissions to run the resource](https://github.com/concourse/concourse/issues/403))

The following build arguments are mandatory:

- `GIT_COMMIT`: the current Git commit id
- `GIT_TAG`: the current Git tag (if any)
- `BUILD_DATE`: the current timestamp

```shell=bash
GIT_COMMIT="$(git rev-parse HEAD)"
GIT_TAG="$(git name-rev --tags --name-only ${GIT_COMMIT})"
BUILD_DATE="$(date -u +'%Y%m%dT%H%M%SZ')"
GO_VERSION="$(go list -f {{.GoVersion}} -m)-bookworm"
export GIT_COMMIT GIT_TAG BUILD_DATE GO_VERSION
docker build --build-arg GIT_TAG="${GIT_TAG}" --build-arg BUILD_DATE="${BUILD_DATE}" --build-arg GIT_COMMIT="${GIT_COMMIT}" --build-arg BUILDER_VERSION="${GO_VERSION}" --tag concourse-netbox-resource:"${GIT_TAG}"-"${BUILD_DATE}" ./
unset GIT_COMMIT GIT_TAG BUILD_DATE GO_VERSION
```

## Usage

This resource is designed to be used in a Concourse CI pipeline. It can be configured to trigger jobs based on events from NetBox, such as changes to devices or other objects.

### Configuration

#### Source Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `url` | *(none)* | The URL of the NetBox instance (mandatory) |
| `token` | *(none)* | The API token for authentication |
| `parallel_queries` | `1` | Number of parallel queries to NetBox. Can speed up the check process, but be careful not to overload your NetBox instance |
| `use_changelog` | `false` | Use the NetBox changelog to determine which devices have interface changes. More efficient than querying all devices, but it may miss changes without a changelog entry |
| `initial_lookback` | *(epoch 0)* | Initial lookback timeframe for the first check (when no version exists) using Go duration format (e.g., `168h` for 7 days, `720h30m` for 30 days 30 minutes, `8760h` for 1 year). Supports: `h` (hours), `m` (minutes), `s` (seconds), and combinations like `1h30m45s`. If not specified, defaults to Unix epoch (1970-01-01) returning all matching objects |

> **Warning:** When `use_changelog` is enabled with filters like `tenant`, `site_name`, or `region_name`, performance may be significantly slower than the classic query. The changelog API fetches all change events globally and filters afterward, while the classic query applies filters directly at the API level. Use `use_changelog: false` (default) for better performance when filtering by tenant or other device attributes.

#### Filter Parameters

All filter parameters are optional and can be used in combination to narrow down the devices and interfaces that trigger the resource. If no filters are provided, all devices and their interfaces will be considered.

| Parameter | Default | Description |
|-----------|---------|-------------|
| `filter.site_name` | `[]` | Filter devices by site name(s) |
| `filter.region_name` | `[]` | Filter devices by region name(s) |
| `filter.tenant` | `[]` | Filter devices by tenant slug(s) (e.g., `my-tenant`) |
| `filter.tag` | `[]` | Filter devices by tag slug(s) |
| `filter.role` | `[]` | Filter devices by role slug(s) |
| `filter.device_id` | `[]` | Filter devices by device ID(s) |
| `filter.device_name` | `[]` | Filter devices by name (case-insensitive contains) |
| `filter.device_type` | `[]` | Filter devices by device type(s) |
| `filter.device_status` | `[]` | Filter devices by status (e.g., `active`, `planned`) |
| `filter.get_config_context` | `false` | Include the device's config context in the output |

#### Server Interface Filter Parameters

Additional filters can be applied to the server interfaces of the devices. These filters will be applied after filtering the devices, so they will only affect the interfaces of the already filtered devices.

| Parameter | Default | Description |
|-----------|---------|-------------|
| `filter.server_interface.interface_id` | `[]` | Filter interfaces by interface ID(s) |
| `filter.server_interface.interface_name` | `[]` | Filter interfaces by name (case-insensitive contains) |
| `filter.server_interface.enabled` | *(none)* | Filter interfaces by enabled state |
| `filter.server_interface.mgmt_only` | *(none)* | Filter interfaces by management-only flag |
| `filter.server_interface.connected` | *(none)* | Filter interfaces by connected state |
| `filter.server_interface.cabled` | *(none)* | Filter interfaces by cabled state |
| `filter.server_interface.type` | `[]` | Filter interfaces by type(s) (e.g., `virtual`, `1000base-t`) |

#### Example Configuration

This is an example of how to configure the resource in a Concourse pipeline:

```yaml
resource_types:
  - name: netbox-resource
    type: registry-image
    check_every: never
    source:
      repository: registry.fqdn/org/concourse-netbox-resource
      tag: 0.1.0

resources:
  - name: example.netbox
    type: netbox-resource
    icon: netbox
    check_every: 15m
    source:
      url: "https://netbox.example.local"
      token: "your-api-token",
      parallel_queries: 1,
      use_changelog: false,
      initial_lookback: "168h30m",
      filter:
        site_name: ["site 1"]
        region_name: ["region 1"]
        tenant: ["my-tenant"]
        tag: ["tag1"]
        role: ["server"]
        device_id: [123]
        device_name: ["server1"]
        device_type: ["vendor model"]
        device_status: ["active"]
        get_config_context: true
        server_interface:
          interface_id: [456]
          interface_name: ["eth0"]
          enabled: true
          mgmt_only: false
          connected: true
          cabled: true
          type: ["virtual"]
```
