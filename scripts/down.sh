#!/usr/bin/env bash

trap 'catch $? $LINENO' ERR
catch() {
  echo "Error $1 occurred on $2"
}

service_account () {
  eksctl delete iamserviceaccount \
    --name notary-admission \
    --namespace notary-admission \
    --cluster $1
}

CLUSTER=${1:-notary-admission}

KUBECTL="kubectl"

helm uninstall notary-admission --namespace notary-admission

${KUBECTL} label ns kube-system notary-admission-ignore-

${KUBECTL} -n notary-admission delete pod notary-admission-test-connection

read -p "Do you wish to delete the service account? " yn
case $yn in
    [Yy]* ) service_account ${CLUSTER}
esac

read -p "Do you wish to delete the namespace? " yn
case $yn in
    [Yy]* ) ${KUBECTL} delete -f k8s/0-ns.yaml
esac