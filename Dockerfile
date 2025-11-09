FROM alpine@sha256:4b7ce07002c69e8f3d704a9c5d6fd3053be500b7f1c69fc0d80990c2ad8dd412 AS certificates
RUN apk add -U --no-cache ca-certificates

FROM gcr.io/distroless/static@sha256:87bce11be0af225e4ca761c40babb06d6d559f5767fbf7dc3c47f0f1a466b92c
COPY --from=certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY upload-sarif /usr/bin/upload-sarif
ENTRYPOINT [ "/usr/bin/upload-sarif" ]
