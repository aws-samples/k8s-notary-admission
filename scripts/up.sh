#!/usr/bin/env bash

trap 'catch $? $LINENO' ERR
catch() {
  echo "Error $1 occurred on $2"
}

service_account () {
  eksctl create iamserviceaccount \
    --name notary-admission \
    --namespace notary-admission \
    --cluster ${CLUSTER} \
    --attach-policy-arn arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly \
    --attach-policy-arn <AWS_SIGNER_POLICY_ARN> \
    --approve \
    --override-existing-serviceaccounts
}

CLUSTER=${1:-notary-admission}

# set -x

KUBECTL="kubectl"

read -p "Do you wish to install/update the namespace? " yn
case $yn in
    [Yy]* ) ${KUBECTL} apply -f k8s/0-ns.yaml
esac

read -p "Do you wish to install/update the service account? " yn
case $yn in
    [Yy]* ) service_account
esac

helm install notary-admission --atomic --namespace notary-admission charts/notary-admission/ \
--set server.tls.secrets.cabundle="$(cat controller/tls/secrets/notary-admission-ca.crt | base64 | tr -d '\n\r')" \
--set server.tls.secrets.key="$(cat controller/tls/secrets/notary-admission-server.key | base64 | tr -d '\n\r')" \
--set server.tls.secrets.crt="$(cat controller/tls/secrets/notary-admission-server.crt | base64 | tr -d '\n\r')"


LABEL=$(${KUBECTL} get ns kube-system -oyaml | { grep notary-admission-ignore || true; })
if [[ "$LABEL" == "" ]]
then
  ${KUBECTL} label ns kube-system notary-admission-ignore=ignore
  ${KUBECTL} get ns kube-system -oyaml | grep notary-admission-ignore
fi

read -p "Do you wish to run post install test? " yn
case $yn in
    [Yy]* ) helm test notary-admission
esac
