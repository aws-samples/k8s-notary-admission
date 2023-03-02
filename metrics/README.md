# Metrics Setup

The K8s Notary Admission controller has been instrumented with Prometheus HTTP metrics.

## Installing Prometheus

Prometheus requires Kubernetes [Persist Volumes (PV)](https://kubernetes.io/docs/concepts/storage/persistent-volumes/) and Persistent Volume Claims (PVC). In Amazon EKS, this requires the a default [Storage Class](https://kubernetes.io/docs/concepts/storage/storage-classes/) and the [aws-ebs-csi-driver](https://docs.aws.amazon.com/eks/latest/userguide/ebs-csi.html) installed in the cluster.

Verifying the default cluster Storage Classes can be done with the following _kubectl_ command.

```
kubectl get storageclass
NAME            PROVISIONER             RECLAIMPOLICY   VOLUMEBINDINGMODE      ALLOWVOLUMEEXPANSION   AGE
gp2 (default)   kubernetes.io/aws-ebs   Delete          WaitForFirstConsumer   false                  24d
```





```
{
  "Version": "2012-10-17",
  "Statement": [
      {
          "Effect": "Allow",
          "Action": [
              "kms:CreateGrant",
              "kms:ListGrants",
              "kms:RevokeGrant"
          ],
          "Resource": [
              "<KMS_KEY_ARN>"
          ],
          "Condition": {
              "Bool": {
                  "kms:GrantIsForAWSResource": "true"
              }
          }
      },
      {
          "Effect": "Allow",
          "Action": [
              "kms:Encrypt",
              "kms:Decrypt",
              "kms:ReEncrypt*",
              "kms:GenerateDataKey*",
              "kms:DescribeKey"
          ],
          "Resource": [
              "<KMS_KEY_ARN>"
          ]
      }
  ]
}
```