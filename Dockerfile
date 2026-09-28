FROM alpine@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6 AS certificates
RUN apk add -U --no-cache ca-certificates

FROM gcr.io/distroless/static@sha256:87bce11be0af225e4ca761c40babb06d6d559f5767fbf7dc3c47f0f1a466b92c
ARG TARGETPLATFORM
COPY --from=certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY $TARGETPLATFORM/upload-sarif /usr/bin/upload-sarif
ENTRYPOINT [ "/usr/bin/upload-sarif" ]
