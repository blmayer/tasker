FROM golang:1.16 as build

ADD . /root/

RUN cd /root && CGO_ENABLED=0 go build -v

FROM scratch

COPY --from=build /root/tasker /
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

EXPOSE 8080

CMD ["/tasker"]
