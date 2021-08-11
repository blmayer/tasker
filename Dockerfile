FROM golang:1.16 as build

ADD . /root/

RUN cd /root && go build -v

FROM scratch

COPY --from=build /root/tasker /tasker

CMD ["tasker"]