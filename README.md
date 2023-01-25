# garbage-collection-example
The project help to understand the garbage collector of kubernetes.


## Getting Started
You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

### Running on the cluster
1.compile
```sh
make build
```

2. package
```sh
make docker-build && make docker-push
```
3. Install operator

```sh
kubectl apply -f config/bases
kubectl apply -f config/rbac
kubectl apply -f config/manager/manager.yaml
```
4.edit and deploy by the example_v1_garbage.yaml to verify
```sh
kubectl apply -f config/samples
```


