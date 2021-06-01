FROM keppel.eu-de-1.cloud.sap/ccloud-dockerhub-mirror/library/golang:alpine AS builder

RUN apk update && apk add --no-cache git ca-certificates && update-ca-certificates

ADD . /app/
WORKDIR /app
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o resource .
RUN mkdir -p /target/opt/resource/
RUN cp resource /target/opt/resource/
RUN ln -s resource /target/opt/resource/in
RUN ln -s resource /target/opt/resource/out
RUN ln -s resource /target/opt/resource/check


FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /target/opt /opt
