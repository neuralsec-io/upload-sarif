FROM alpine@sha256:be2656061b536c1883c4034790e21d0a09c5aaaf51968b89368a3c0371ea2c9a AS certificates
RUN apk add -U --no-cache ca-certificates

FROM gcr.io/distroless/static@sha256:448810357cc8e67b7208988abdba0f5ec3d3aebee0a5b13eb624979792cffc0f
ARG TARGETPLATFORM
COPY --from=certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY $TARGETPLATFORM/upload-sarif /usr/bin/upload-sarif
ENTRYPOINT [ "/usr/bin/upload-sarif" ]
