#!/bin/bash

# this script starts a test cockroachdb cluster on $HOSTNAME
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

# start node 1
#cockroach start --insecure --store=$STORE_ROOT/cockroach-store --listen-addr=$HOSTNAME:$NODE1_DB --http-addr=$HOSTNAME:$NODE1_HTTP --join=$HOSTNAME:$NODE1_DB,$HOSTNAME:$NODE2_DB,$HOSTNAME:$NODE3_DB --background

# start node 2
#cockroach start --insecure --store=$STORE_ROOT/cockroach-store2 --listen-addr=$HOSTNAME:$NODE2_DB --http-addr=$HOSTNAME:$NODE2_HTTP --join=$HOSTNAME:$NODE1_DB,$HOSTNAME:$NODE2_DB,$HOSTNAME:$NODE3_DB --background

# start node 3
#cockroach start --insecure --store=$STORE_ROOT/cockroach-store3 --listen-addr=$HOSTNAME:$NODE3_DB --http-addr=$HOSTNAME:$NODE3_HTTP --join=$HOSTNAME:$NODE1_DB,$HOSTNAME:$NODE2_DB,$HOSTNAME:$NODE3_DB --background

# start node 1
cockroach start --store=$STORE_ROOT/cockroach-store \
	--certs-dir=certs/db \
	--listen-addr=$HOSTNAME:$NODE1_DB \
	--http-addr=$HOSTNAME:$NODE1_HTTP \
	--join=$HOSTNAME:$NODE1_DB,$HOSTNAME:$NODE2_DB,$HOSTNAME:$NODE3_DB \
	--background

# start node 2
cockroach start --store=$STORE_ROOT/cockroach-store2 \
	--certs-dir=certs/db \
	--listen-addr=$HOSTNAME:$NODE2_DB \
	--http-addr=$HOSTNAME:$NODE2_HTTP \
	--join=$HOSTNAME:$NODE1_DB,$HOSTNAME:$NODE2_DB,$HOSTNAME:$NODE3_DB \
	--background

# start node 3
cockroach start --store=$STORE_ROOT/cockroach-store3 \
	--certs-dir=certs/db \
	--listen-addr=$HOSTNAME:$NODE3_DB \
	--http-addr=$HOSTNAME:$NODE3_HTTP \
	--join=$HOSTNAME:$NODE1_DB,$HOSTNAME:$NODE2_DB,$HOSTNAME:$NODE3_DB \
	--background

# initialize the cluster
cockroach init --host=$HOSTNAME:$NODE1_DB \
	--certs-dir=certs/db 
