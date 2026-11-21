# Image for the Docker LAN test. The munna binary is copied in by run-docker.sh,
# which builds it for the container platform in a temporary build context.
FROM alpine:3.20

RUN apk add --no-cache curl python3 ca-certificates

COPY munna /usr/local/bin/munna

ENTRYPOINT ["/bin/sh", "-c"]
