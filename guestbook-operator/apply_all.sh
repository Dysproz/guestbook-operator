kubectl $1 -f ./deploy/role.yaml
kubectl $1 -f ./deploy/service_account.yaml
kubectl $1 -f ./deploy/role_binding.yaml
kubectl $1 -f ./deploy/crds/dysproz.io_redis_crd.yaml
kubectl $1 -f ./deploy/crds/dysproz.io_guestbooks_crd.yaml
kubectl $1 -f ./deploy/operator.yaml
kubectl $1 -f ./deploy/crds/dysproz.io_v1alpha1_guestbook_cr.yaml

