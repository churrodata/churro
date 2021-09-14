#/bin/bash

wget -qO- https://binaries.cockroachdb.com/cockroach-v19.2.4.linux-amd64.tgz | tar  xvz

mv cockroach-v19.2.4.linux-amd64* /tmp

sudo mv /tmp/cockroach*/cockroach /usr/local/bin/
rm -rf /tmp/cockroach-v19*
