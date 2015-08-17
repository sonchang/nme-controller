FROM ubuntu
RUN apt-get update
RUN apt-get -y install bridge-utils wget git golang build-essential python
RUN wget https://www.kernel.org/pub/linux/utils/util-linux/v2.24/util-linux-2.24.tar.bz2
RUN bzip2 -d -c util-linux-2.24.tar.bz2 | tar xvf -
RUN cd util-linux-2.24 && ./configure --without-ncurses && make nsenter && cp nsenter /usr/local/bin
COPY ./setup.py /setup.py
COPY ./bootstrap.sh /bootstrap.sh
COPY ./main.go /main.go
COPY ./nme_controller.sh /nme_controller.sh
RUN chmod +x /setup.py
RUN chmod +x /bootstrap.sh
RUN chmod +x /nme_controller.sh
CMD /bootstrap.sh

