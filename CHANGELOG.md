# Changelog

## 0.2.0

* `region_name`, `tenant` and `tag` properties added to the query filter
* `device_site`, `device_region` and `device_tags` properties added to the version output
* `use_changelog` option added to use the NetBox changelog. See the documentation for details and performance considerations
* `parallel_queries` option added to speed up the check process by running multiple queries in parallel
* golang version updated to `1.25.7`


## 0.1.0

* official netbox library used
* check command with filter support
* in,out command currently only noop
* (unit) tests rewritten
* Container image creation updated and documented
* Golang image for the build stage
* Distroless base without libc and shell for the final image
* Concourse config documented
* Github actions workflows created
