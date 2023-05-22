#!/usr/bin/env bash

trap 'catch $? $LINENO' ERR
catch() {
  echo "Error $1 occurred on $2"
}

for ((i=1;i<=1000;i++)); 
do 
   kubectl apply -f 3-test-pod-bad.yaml
   echo $i
done