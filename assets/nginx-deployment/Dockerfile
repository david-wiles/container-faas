FROM vlab-server AS builder
FROM nginx:alpine

COPY --from=builder /go/bin/faas-server /go/bin/faas-server
COPY ./nginx.conf /etc/nginx/nginx.conf

RUN mkdir /etc/nginx/apps/
RUN mkdir /etc/nginx/logs/

COPY ./docker-entrypoint.sh /
EXPOSE 1025-65535
ENTRYPOINT ["/docker-entrypoint.sh"]
