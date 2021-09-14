#!/bin/bash

CREDS=/home/jeffmc/churro-grpc/certs

echo "running extract job...config file is $1"
churro-extract  -debug=true \
	-servicecrt $CREDS/grpc/service.crt \
	-servicekey $CREDS/grpc/service.key \
	-dbsslrootcert $CREDS/db/grpc/ca.crt \
	-dbsslkey $CREDS/db/client.pipeline1user.key \
	-dbsslcert $CREDS/db/client.pipeline1user.crt \
	-config $1  2>&1 /tmp/churro-extract.log
