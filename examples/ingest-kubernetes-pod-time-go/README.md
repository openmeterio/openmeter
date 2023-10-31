# Metering Kubernetes Pod Execution Time

In this example, we will track the duration of a Pod running in Kubernetes with second-level accuracy.
This functionality is valuable for applications that require accurate compute time attribution to users for billing or analytics purposes.

## Our Example

In this example, we will develop a Go application that runs within a Kubernetes cluster. This application polls the running containers across all namespaces every second, targeting Pods with a specific `subject` label. The application then reports the collected data to OpenMeter for further analysis and usage tracking.

To facilitate this example, we will build upon an [official Kubernetes example]((https://github.com/kubernetes/client-go/tree/master/examples/in-cluster-client-configuration)) that enables in-cluster API access.

```sh
kubectl create clusterrolebinding default-view --clusterrole=view --serviceaccount=default:default
```

If you plan to utilize this example in a production environment, it is crucial to ensure that only one instance of this code is running and that overreporting is avoided.
Additionally, if your Kubernetes cluster runs a large scale of pods, consider transforming this code into a `DaemonSet` where each instance runs on a single Kubernetes Node and manages Pods exclusively from that node to distribute the load.

## Trying out locally

If you wish to try out this example locally, you can use [minikube](https://minikube.sigs.k8s.io/).

Follow these steps to install and start Minikube:

```sh
brew install minikube
minikube start
```

Run the following commands to deploy the required images:

```sh
minikube kubectl -- create clusterrolebinding default-view --clusterrole=view --serviceaccount=default:default
minikube image build  --file examples/ingest-kubernetes-pod-time-go/Dockerfile -t k8s-pod-time:latest ../..
minikube kubectl -- run hello --image=nginxdemos/hello --labels=subject=customer-1
minikube kubectl -- run k8s-pod-time-1 --image=k8s-pod-time:latest --image-pull-policy=Never
minikube kubectl -- logs -f k8s-pod-time-1
```

Check out the [quickstart guide](/quickstart) to see how to run OpenMeter.
