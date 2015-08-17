#!/bin/bash

export GOPATH=/go
export PATH=$PATH:/go/bin
go get github.com/Sirupsen/logrus
# next line is a hack until the metadata service is automatically added to DNS
#sudo echo "10.42.95.80 metadata" >> /etc/hosts
sudo python setup.py -r nme -k nsppe16 --nsipmask "172.17.0.200 255.255.0.0" --nssshport 11000 --nshttpport 11001 

