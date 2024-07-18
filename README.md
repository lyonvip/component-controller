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

3. View Running Resources:

```sh
$ kubectl get comp -n component
NAME                  TYPE       ISVALIDATE
aic-mysql-server      mysql      true
aic-rabbitmq-server   rabbitmq   true
aic-redis-server      redis      true

$ kubectl get deploy -n component
NAME              READY   UP-TO-DATE   AVAILABLE   AGE
mysql-server      1/1     1            1           20h
rabbitmq-server   1/1     1            1           20h
redis-server      1/1     1            1           20h

$ kubectl get po -n component
NAME                               READY   STATUS    RESTARTS   AGE
mysql-server-798bd45cff-zcbcq      2/2     Running   0          20h
rabbitmq-server-857944cb89-bqqcg   1/1     Running   0          20h
redis-server-6957c56bf7-wgfjk      2/2     Running   0          20h
```