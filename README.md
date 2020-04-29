# Guestbook Operator

## Idea
This is a sample project of Kubernetes Operator for deployment of Guestbook PHP application with Redis in backend.

Implementation of guestbook is based on [this](https://kubernetes.io/docs/tutorials/stateless-application/guestbook/) Kubernetes tutorial, but it's implemented as operator.

## Setup

### Building the operator
Operator may be build from guestbook-operator root directory with:
```
operator-sdk build <name of container>
```
Afterwards container should be pushed to the internet.
e.g.:
```
docker push <name of container>
```

Operator is also available from Docker Hub as `dysproz/guestbook-operator`

### Deployment
Operator should be deployed with applying YAMLs uder *deploy/* directory.
```
kubectl apply -f deploy/service_account.yaml
kubectl apply -f deploy/role.yaml
kubectl apply -f deploy/role_binding.yaml
kubectl apply -f deploy/crds/dysproz_guestbooks_crd.yaml
kubectl apply -f deploy/operator.yaml
```
This should setup operator in your cluster. You may check it by running `kubectl get pods`.

To apply example Guestbook resource run: `kubectl apply -f deploy/crds/dysproz_v1alpha1_guestbook_cr.yaml`.

### Run operator locally
Alternative way would be to run operator locally:
```
operator-sdk run --local --with-namespace=default
```
This command will run operator in local cluster (for example Kind).
