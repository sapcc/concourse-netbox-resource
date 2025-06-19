ARG BUILDER_NAME="golang"
ARG BUILDER_VERSION="1.25.0-bookworm"
ARG BASE_NAME="gcr.io/distroless/static-debian12"
ARG BASE_VERSION="latest"
ARG GIT_COMMIT="undefined"
ARG GIT_TAG="undefined"
ARG BUILD_DATE="undefined"
FROM $BUILDER_NAME:$BUILDER_VERSION AS buildstage
ARG GIT_COMMIT
ARG GIT_TAG
ARG BUILD_DATE

ENV CGO_ENABLED=0

COPY . /go/src
SHELL ["/bin/bash", "-euo", "pipefail", "-c"]

RUN mkdir -p /opt/resource
WORKDIR /go/src
RUN \
  export LDFLAGS="-X 'github.com/sapcc/concourse-netbox-resource/internal/helper.gitCommit=${GIT_COMMIT}' -X 'github.com/sapcc/concourse-netbox-resource/internal/helper.buildDate=${BUILD_DATE}' -X 'github.com/sapcc/concourse-netbox-resource/internal/helper.gitVersion=${GIT_TAG}'" \
  && go test -ldflags "${LDFLAGS}" -cover ./... \
  && go build -ldflags "${LDFLAGS}" -o /opt/resource/check main.go \
  && ln -s /opt/resource/check /opt/resource/in \
  && ln -s /opt/resource/check /opt/resource/out

RUN /opt/resource/check -v | grep -q "${GIT_TAG}"
RUN /opt/resource/in -v | grep -q "${GIT_TAG}"
RUN /opt/resource/out -v | grep -q "${GIT_TAG}"

ARG BASE_NAME
ARG BASE_VERSION
FROM $BASE_NAME:$BASE_VERSION
ARG BASE_NAME
ARG BASE_VERSION
ARG GIT_COMMIT
ARG GIT_TAG
ARG BUILD_DATE
COPY --from=buildstage /opt/resource /opt/resource

LABEL org.opencontainers.image.title="concourse-netbox-resource"
LABEL org.opencontainers.image.authors="businessbean, SchwarzM"
LABEL org.opencontainers.image.url="https://github.com/sapcc/concourse-netbox-resource/blob/master/Dockerfile"
LABEL org.opencontainers.image.revision="${GIT_COMMIT}"
LABEL org.opencontainers.image.version="${GIT_TAG}"
LABEL org.opencontainers.image.created="${BUILD_DATE}"
LABEL org.opencontainers.image.licenses="Apache-2.0"
LABEL org.opencontainers.image.base.name="${BASE_NAME}"
LABEL org.opencontainers.image.base.digest="${BASE_NAME}:${BASE_VERSION}"
LABEL source_repository="https://github.com/sapcc/concourse-netbox-resource/"
