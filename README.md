# component-controller
controller for create middleware running in k3s/k8s

## Getting Started
1. Install Controller In kube-system Namespace:

```sh
kubectl apply -f deploy/deploy-all-in-one.yaml
```

2. Running Middleware example:
	
```sh
kubectl apply -f deploy/sample-comp.yaml
```
