FROM node:14-alpine

RUN echo "\
const http = require('http');\
const server = http.createServer((req, res) => {res.end();});\
server.listen(9003);\
"  > /bin/health.js

# The program files will be mounted at /home/app
WORKDIR /home/app

STOPSIGNAL SIGINT

COPY ./docker-entrypoint.sh /usr/local/bin/
ENTRYPOINT ["docker-entrypoint.sh"]

EXPOSE 8080
EXPOSE 9003
