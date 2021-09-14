#!/bin/bash
kubectl -n testdb create secret generic  mypwds \
        --from-literal=rootUser=root \
        --from-literal=rootHost=% \
        --from-literal=rootPassword="secret"
