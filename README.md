# concourse-netbox-resource
A concourse resource to trigger from netbox

## Container Image build

This resource is built using a Dockerfile that uses a multi-stage build process. The first stage builds the Go application, and the second stage creates a minimal container image using distroless. The following optional build arguments are available:

- `BUILDER_NAME`: The name of the Go builder image (default: `golang`)
- `BUILDER_VERSION`: The version of the Go builder image (default: `1.24.4-bookworm`)
- `BASE_NAME`: The base image for the final container (default: `gcr.io/distroless/static-debian12`)
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

The `source.url` parameter is mandatory. All fields in the `source.filter` section and the `source.token` are optional. Fields with brackets `[]` can contain multiple values. `source.filter.device_name` and `source.filter.interface_name` are using a `case-insensitive contains` filter. The `source.filter.get_config_context` parameter can be set to `true` to include the device's config context in the output. Because of the current limitation in the go-netbox library the config context is gathered with every query, but only included in the output if this parameter is set to `true`.

This is an example of how to configure the resource in a Concourse pipeline:

```yaml
resource_types:
  - name: netbox-resource
    type: registry-image
    check_every: never
    source:
      repository: registry.fqdn/org/concourse-netbox-resource
      tag: 0.1.0-20250619163905

resources:
  - name: example.netbox
    type: netbox-resource
    icon: netbox
    check_every: 15m
    source:
      url: "https://netbox.example.local"
      token: "your-api-token"
      filter:
        site_name: ["site 1"]
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
