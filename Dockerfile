#########################################################
# Build the sources and provide the result in a multi stage
# docker container. The alpine build image has to match
# the alpine image in the referencing runtime container.
#########################################################
FROM golang:1.11.2-alpine3.8 AS builder

RUN apk add  \
		bash \
		gcc \
		musl-dev \
		openssl

# Directory in workspace
WORKDIR "/go/src/github.com/Peripli/service-manager-broker-proxy-istio-plugin"

COPY . ./
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -v -a -buildmode=plugin  -o /service-manager-istio-plugin.so  .

FROM  gcr.io/sap-se-gcp-istio-dev/sb-proxy-k8s

WORKDIR /app

COPY --from=builder /service-manager-istio-plugin.so /app/

RUN apk add --no-cache bash gawk sed grep bc coreutils

ENTRYPOINT [ "./main","--plugin","service-manager-istio-plugin.so"]
