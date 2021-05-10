FROM golang:1.16.4-buster@sha256:cdc0ce37b610ff73209377bf3a8ba9b3710bf853876863d4b3b6c1f778edade5 AS build

WORKDIR $GOPATH/src/kurtschwarz/home/services/cloudflare-ddns
COPY ./ .
RUN CGO_ENABLED=0 go install -ldflags '-extldflags "-static"' -tags timetzdata

FROM scratch AS runtime

COPY --from=build /go/bin/cloudflare-ddns /cloudflare-ddns
COPY --from=alpine:3.13.5@sha256:69e70a79f2d41ab5d637de98c1e0b055206ba40a8145e7bddb55ccc04e13cf8f /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT [ "/cloudflare-ddns" ]
