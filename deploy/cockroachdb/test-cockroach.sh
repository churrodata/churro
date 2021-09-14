#!/bin/bash

set -x
# this script tests the cockroach cluster
# used only for testing
# reference:  https://www.cockroachlabs.com/docs/stable/start-a-local-cluster.html

# storage is used within /tmp so it is temporary only
STORE_ROOT=/tmp

# cockroach http ports 
NODE1_HTTP=10000
NODE2_HTTP=10001
NODE3_HTTP=10002

# cockroach db ports
NODE1_DB=11000
NODE2_DB=11001
NODE3_DB=11002

curl http://$HOSTNAME:$NODE1_HTTP
curl http://$HOSTNAME:$NODE2_HTTP
curl http://$HOSTNAME:$NODE3_HTTP
cockroach sql --certs-dir=certs/db \
	--host=$HOSTNAME:$NODE1_DB --execute="create table foo (id int);"
cockroach sql --certs-dir=certs/db \
	--host=$HOSTNAME:$NODE2_DB --execute="select * from foo;"
cockroach sql --certs-dir=certs/db \
	--host=$HOSTNAME:$NODE3_DB --execute="drop table foo;"
