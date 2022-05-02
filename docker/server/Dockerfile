FROM debian:stable
#FROM bmixo/images:torrent-es1

#RUN apt update
#RUN apt install curl inetutils-ping net-tools dnsutils wget -y

WORKDIR /dist
COPY ./server ./server
COPY ./server.conf ./server.conf
ADD ./config  ./config
RUN chmod +x /dist/server



CMD ["/dist/server"]
