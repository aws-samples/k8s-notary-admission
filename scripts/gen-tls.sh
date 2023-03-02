#!/usr/bin/env bash

trap 'catch $? $LINENO' ERR
catch() {
  echo "Error $1 occurred on $2"
}

TLS_DIR=${1:-"controller/tls"}

pushd "${TLS_DIR}"

rm -rf secrets/*

openssl genrsa -out secrets/notary-admission-ca.key 2048
openssl req -x509 -new -nodes -sha256 -key secrets/notary-admission-ca.key -days 365 -out secrets/notary-admission-ca.crt -subj /CN=admission_ca 2>&1
openssl genrsa -out secrets/notary-admission-server.key 2048
openssl req -new -key secrets/notary-admission-server.key -sha256 -out secrets/notary-admission-server.csr -subj /CN=notary-admission.notary-admission.svc -config server.conf 2>&1
openssl x509 -req -days 365 -in secrets/notary-admission-server.csr -sha256 -CA secrets/notary-admission-ca.crt -CAkey secrets/notary-admission-ca.key -CAcreateserial -out secrets/notary-admission-server.crt -days 100000 -extensions v3_ext -extfile server.conf

base64 secrets/notary-admission-ca.crt > secrets/notary-admission-tls-ca-bundle.out
base64 secrets/notary-admission-server.crt > secrets/notary-admission-tls-crt.out
base64 secrets/notary-admission-server.key > secrets/notary-admission-tls-key.out

popd