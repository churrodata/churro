#!/bin/bash

# this script starts a test prometheus cluster on $HOSTNAME
# used only for testing
# reference:  https://prometheus.io/docs/prometheus/latest/installation/

# storage is used within /tmp so it is temporary only

cp deploy/prometheus/prometheus.yml /tmp

# here we define port 9999 as the prometheus port, mapping to the internal
# port of 9090
#docker run \
#	-p 9999:9090  \
#	-v /tmp/prometheus.yml:/etc/prometheus/prometheus.yml \
#	prom/prometheus

prometheus --config.file=/tmp/prometheus.yml \
	--web.listen-address="0.0.0.0:9999"
