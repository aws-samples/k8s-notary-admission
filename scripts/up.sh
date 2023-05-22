#!/usr/bin/env bash

trap 'catch $? $LINENO' ERR
catch() {
  echo "Error $1 occurred on $2"
}

ECR_POLICY=arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly
SIGNER_POLICY=arn:aws:iam::<AWS_ACCOUNT_ID>:policy/<IAM_POLICY_NAME>

service_account () {
  eksctl create iamserviceaccount \
    --name notary-admission \
    --namespace notary-admission \
    --cluster $1 \
    --attach-policy-arn $ECR_POLICY \
    --attach-policy-arn $SIGNER_POLICY \
    --approve \
    --override-existing-serviceaccounts
}

CLUSTER=${1:-notary-admission}

KUBECTL="kubectl"

read -p "Do you wish to install/update the namespace? " yn
case $yn in
    [Yy]* ) ${KUBECTL} apply -f k8s/0-ns.yaml
esac

read -p "Do you wish to install/update the service account? " yn
case $yn in
    [Yy]* ) service_account ${CLUSTER}
esac

helm install notary-admission --atomic --namespace notary-admission charts/notary-admission/ \
--set server.tls.secrets.cabundle="$(cat controller/tls/secrets/notary-admission-ca.crt | base64 | tr -d '\n\r')" \
--set server.tls.secrets.key="$(cat controller/tls/secrets/notary-admission-server.key | base64 | tr -d '\n\r')" \
--set server.tls.secrets.crt="$(cat controller/tls/secrets/notary-admission-server.crt | base64 | tr -d '\n\r')"

echo ""

LABEL=$(${KUBECTL} get ns kube-system -oyaml | { grep notary-admission-ignore || true; })
if [[ "$LABEL" == "" ]]
then
  ${KUBECTL} label ns kube-system notary-admission-ignore=ignore
  ${KUBECTL} get ns kube-system -oyaml | grep notary-admission-ignore
fi

echo ""

read -p "Do you wish to run post install test? " yn
case $yn in
    [Yy]* ) helm test notary-admission -n notary-admission
esac