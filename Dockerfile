FROM alpine@sha256:71e9b7bae160ba167fa17414401b94cbcdaf3c32feeab9298bd794a2a0614c35 AS certificates
RUN apk add -U --no-cache ca-certificates

FROM gcr.io/distroless/static@sha256:034df5c1737e8241f6ed6423a12e720ee0f6d6f088609ee172f98f6f2166ef5d
ARG TARGETPLATFORM
COPY --from=certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY $TARGETPLATFORM/upload-sarif /usr/bin/upload-sarif
ENTRYPOINT [ "/usr/bin/upload-sarif" ]
