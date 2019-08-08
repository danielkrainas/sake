FROM alpine:latest

ENV DIST_NAME sake
ENV APP_HOME /opt/sake

RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*

COPY ./dist $APP_HOME/
COPY ./docker/* $APP_HOME/
RUN chmod +x $APP_HOME/$DIST_NAME && chmod +x $APP_HOME/*.sh
RUN mv $APP_HOME/$DIST_NAME /bin

ENTRYPOINT ["/opt/sake/entrypoint.sh"]
