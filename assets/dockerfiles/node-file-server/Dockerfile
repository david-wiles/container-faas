FROM node:14-alpine

# The program files will be mounted at /home/app
WORKDIR /home/app

RUN npm install http-server -g

STOPSIGNAL SIGINT

COPY ./docker-entrypoint.sh /usr/local/bin/
ENTRYPOINT ["docker-entrypoint.sh"]

EXPOSE 8080
