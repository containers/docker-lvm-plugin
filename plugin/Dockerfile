FROM alpine
RUN apk update && apk add lvm2 xfsprogs cryptsetup thin-provisioning-tools
RUN mkdir -p /var/lib/docker-lvm-plugin
COPY docker-lvm-plugin /
