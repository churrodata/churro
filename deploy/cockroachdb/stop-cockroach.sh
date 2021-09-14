#!/bin/bash

# this script starts a test cockroachdb cluster on $HOSTNAME
# used only for testing
# reference:  https://www.cockroachlabs.com/docs/stable/start-a-local-cluster.html

# storage is used within /tmp so it is temporary only
STORE_ROOT=/tmp

# cockroach db ports
NODE1_DB=11000
NODE2_DB=11001
NODE3_DB=11002

cockroach quit --insecure --host=$HOSTNAME:$NODE1_DB
cockroach quit --insecure --host=$HOSTNAME:$NODE2_DB
cockroach quit --insecure --host=$HOSTNAME:$NODE3_DB
