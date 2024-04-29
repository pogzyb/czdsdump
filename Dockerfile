FROM golang:alpine3.19 as stage

WORKDIR /build
COPY . .
RUN go build -o czdsdump .

FROM alpine:3.19

ARG VERSION
ARG CREATED
ARG REVISION

LABEL org.opencontainers.image.title="CZDSdump"
LABEL org.opencontainers.image.description="Tool for dumping the Centralized Zone Data System."
LABEL org.opencontainers.image.version=$VERSION
LABEL org.opencontainers.image.authors="pogzyb@umich.edu"
LABEL org.opencontainers.image.url="https://github.com/pogzyb/czdsdump"
LABEL org.opencontainers.image.source="https://github.com/pogzyb/czdsdump"
LABEL org.opencontainers.image.documentation="https://github.com/pogzyb/czdsdump"
LABEL org.opencontainers.image.created=$CREATED
LABEL org.opencontainers.image.revision=$REVISION
LABEL org.opencontainers.image.licenses="MIT"

COPY --from=stage /build/czdsdump /usr/local/bin/czdsdump
USER guest
ENTRYPOINT [ "czdsdump" ]
CMD [ "--help" ]