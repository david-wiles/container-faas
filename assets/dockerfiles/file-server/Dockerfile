FROM httpd:alpine

# The program files will be mounted at /home/app
WORKDIR /home/app

STOPSIGNAL SIGINT

COPY ./httpd.conf /usr/local/apache2/conf/httpd.conf
COPY ./docker-entrypoint.sh /usr/local/bin/
ENTRYPOINT ["docker-entrypoint.sh"]

EXPOSE 8080
EXPOSE 8081
