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

### Run operator locally with Kind
In order to run operator locally create cluster with config `kind_test_cluster.yml`
```
kind create cluster --config kind_test_cluster.yml
```
This config exposes node port 80 to port 8000 on your local setup.

Afterwards, deploy operator as described in Deployment section.

Wait till all ports will be created.
```
shell$ kubectl get pods

NAME                                                  READY   STATUS    RESTARTS   AGE
example-guestbook-frontend-57949bd655-pm6sl           1/1     Running   0          4m54s
example-guestbook-frontend-57949bd655-tn45x           1/1     Running   0          4m54s
example-guestbook-frontend-57949bd655-vpnxh           1/1     Running   0          4m54s
example-guestbook-redis-redis-master                  1/1     Running   0          6m55s
example-guestbook-redis-redis-slave-5cddd644c-9srcn   1/1     Running   0          5m59s
example-guestbook-redis-redis-slave-5cddd644c-h5qbd   1/1     Running   0          5m59s
guestbook-operator-55457f87f-28p87                    1/1     Running   0          7m24s

```

With operator properly applied, open your favorite web browser and enter `http://127.0.0.1:8000`.
You'll see guestbook aplication working on your local cluster.
