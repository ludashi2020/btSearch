FROM debian:stable

RUN apt update
RUN apt install curl inetutils-ping net-tools dnsutils wget -y

WORKDIR /dist
COPY ./worker ./worker
COPY ./worker.conf ./worker.conf
RUN chmod +x /dist/worker



CMD ["/dist/worker"]
