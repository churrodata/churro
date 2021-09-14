#!/bin/bash

CERTS_DIR=$1
PIPELINE=$2
TARGET=$CERTS_DIR/$PIPELINE
echo "gen certs for pipeline " $PIPELINE " into " $TARGET
DB_CERTS_DIR=$TARGET/db
GRPC_CERTS_DIR=$TARGET/grpc

### Check if a directory does not exist ###
if [ ! -d $DB_CERTS_DIR ] 
then
    echo "Directory " $DB_CERTS_DIR " DOES NOT exists. Will create..." 
    mkdir -p $DB_CERTS_DIR 
fi
if [ ! -d $GRPC_CERTS_DIR ] 
then
    echo "Directory " $GRPC_CERTS_DIR " DOES NOT exists. Will create..." 
    mkdir -p $GRPC_CERTS_DIR 
fi

## create db certs
if [ -f $DB_CERTS_DIR/ca.key ]; then 
	rm $DB_CERTS_DIR/ca.key
fi
if [ -f $DB_CERTS_DIR/ca.crt ]; then 
	rm $DB_CERTS_DIR/ca.crt
fi
if [ -f $DB_CERTS_DIR/client.$PIPELINE.crt ]; then 
	rm $DB_CERTS_DIR/client.$PIPELINE.crt
fi
if [ -f $DB_CERTS_DIR/client.$PIPELINE.key ]; then 
	rm $DB_CERTS_DIR/client.$PIPELINE.key
fi
if [ -f $DB_CERTS_DIR/client.root.key ]; then 
	rm $DB_CERTS_DIR/client.root.key
fi
if [ -f $DB_CERTS_DIR/client.root.crt ]; then 
	rm $DB_CERTS_DIR/client.root.crt
fi
if [ -f $DB_CERTS_DIR/node.crt ]; then 
	rm $DB_CERTS_DIR/node.crt
fi
if [ -f $DB_CERTS_DIR/node.key ]; then 
	rm $DB_CERTS_DIR/node.key
fi
cockroach cert create-ca \
        --certs-dir=$DB_CERTS_DIR/ \
        --ca-key=$DB_CERTS_DIR/ca.key
cockroach cert create-node \
        localhost 127.0.0.1 cockroachdb-public cockroachdb-public.$PIPELINE cockroachdb-public.$PIPELINE.svc.cluster.local *.cockroachdb *.cockroachdb.$PIPELINE *.cockroachdb.$PIPELINE.svc.cluster.local \
        --certs-dir=$DB_CERTS_DIR \
        --ca-key=$DB_CERTS_DIR/ca.key
cockroach cert create-client \
        root \
        --certs-dir=$DB_CERTS_DIR \
        --ca-key=$DB_CERTS_DIR/ca.key
cockroach cert create-client \
        $PIPELINE \
        --certs-dir=$DB_CERTS_DIR \
        --ca-key=$DB_CERTS_DIR/ca.key

## generate grpc certs
if [ -f $GRPC_CERTS_DIR/ca.key ]; then 
	rm $GRPC_CERTS_DIR/ca.key
fi
if [ -f $GRPC_CERTS_DIR/ca.cert ]; then 
	rm $GRPC_CERTS_DIR/ca.cert
fi
if [ -f $GRPC_CERTS_DIR/service.key ]; then 
	rm $GRPC_CERTS_DIR/service.key
fi
if [ -f $GRPC_CERTS_DIR/service.csr ]; then 
	rm $GRPC_CERTS_DIR/service.csr
fi
if [ -f $GRPC_CERTS_DIR/service.crt ]; then 
	rm $GRPC_CERTS_DIR/service.crt
fi


openssl genrsa -out $GRPC_CERTS_DIR/ca.key 4096

openssl req -new -x509 -key $GRPC_CERTS_DIR/ca.key -sha256 -subj "/C=US/ST=TX/O=CA, Inc." -days 365 -out $GRPC_CERTS_DIR/ca.cert

openssl genrsa -out $GRPC_CERTS_DIR/service.key 4096

openssl req -new -key $GRPC_CERTS_DIR/service.key -out $GRPC_CERTS_DIR/service.csr -config $CERTS_DIR/grpc/certificate.conf

openssl x509 -req -in $GRPC_CERTS_DIR/service.csr -CA $GRPC_CERTS_DIR/ca.cert \
-CAkey $GRPC_CERTS_DIR/ca.key -CAcreateserial \
-out $GRPC_CERTS_DIR/service.crt -days 365 -sha256 -extfile $CERTS_DIR/grpc/certificate.conf -extensions req_ext

