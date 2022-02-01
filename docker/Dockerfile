FROM debian:stable
WORKDIR /dist
COPY ./torrent-web ./torrent-web
ADD ./misc ./misc

RUN apt update&&apt install  curl -y

RUN chmod +x /dist/torrent-web

CMD ["/dist/torrent-web"]
