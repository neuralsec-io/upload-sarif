FROM alpine@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6 AS certificates
RUN apk add -U --no-cache ca-certificates

FROM gcr.io/distroless/static@sha256:58133991db06659feaabe0f4e97a35cebf15ef4ea08f8a4c6d2ee5f75e4aa6a0
ARG TARGETPLATFORM
COPY --from=certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY $TARGETPLATFORM/upload-sarif /usr/bin/upload-sarif
ENTRYPOINT [ "/usr/bin/upload-sarif" ]
