FROM ubuntu
RUN apt-get update
RUN apt-get -y install bridge-utils wget git golang build-essential python
RUN wget https://www.kernel.org/pub/linux/utils/util-linux/v2.24/util-linux-2.24.tar.bz2
RUN bzip2 -d -c util-linux-2.24.tar.bz2 | tar xvf -
RUN cd util-linux-2.24 && ./configure --without-ncurses && make nsenter && cp nsenter /usr/local/bin
ADD https://github.com/sonchang/nme-controller/releases/download/v0.2/nme-controller /usr/bin/nme-controller
COPY ./setup.py /setup.py
COPY ./bootstrap.sh /bootstrap.sh
COPY ./main.go /main.go
RUN chmod +x /setup.py
RUN chmod +x /bootstrap.sh
RUN chmod +x /usr/bin/nme-controller
CMD /bootstrap.sh

