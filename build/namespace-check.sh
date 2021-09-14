#!/bin/bash

NS=$1
kubectl get ns $NS

if [ $? -eq 1 ]; then 
	kubectl create namespace $NS
fi
