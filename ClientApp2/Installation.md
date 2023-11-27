# ClientApp Operator Documentation

This documentation provides instructions on deploying and using the ClientApp operator. The operator is designed to manage instances of the `ClientApp` custom resource, representing a simple HTTP API server.

## Table of Contents
1. [Set-up](#1-set-up)
   - [Prerequisites](#prerequisites)
2. [Deploying the Operator](#2-deploying-the-operator)
   - [Operator Deployment](#operator-deployment)
   - [Creating a ClientApp](#creating-a-clientapp)
   - [Updating a ClientApp](#updating-a-clientapp)
   - [Deleting a ClientApp](#deleting-a-clientapp)
3. [Cleanup](#4-cleanup)

## 1. Set-up

### Prerequisites
Ensure you have the following prerequisites installed on your system:
- [Go 1.19](https://golang.org/doc/install)
- [Minikube](https://minikube.sigs.k8s.io/docs/start/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- [Operator SDK 1.30](https://sdk.operatorframework.io/)

### Minikube Installation
Start a Minikube cluster by running the following command:
```sh
minikube start
```
## 2. Deploying the Operator 
### Operator Deployment
a. Clone this repository:
   ```sh
   git clone https://github.com/YOUR_USERNAME/clientapp-operator.git
  
   ```
b. Change into the operator directory:
    ``` sh 
    cd clientapp-operator 
    ``` 

c. Build and deploy the operator:
   ```sh 
    make install
    make run
   ```
  Note: Ensure that your kubeconfig is correctly set up to point to the target Kubernetes cluster. If you're not using the default kubeconfig, specify the path with the KUBECONFIG environment variable

Build and push your image to the location specified by `IMG`:

```sh
make docker-build docker-push IMG=<some-registry>/clientapp:tag
```

Deploy the controller to the cluster with the image specified by `IMG`:

```sh
make deploy IMG=<some-registry>/clientapp:tag
```
d. verify that the operator is running successfully. You should see logs indicating that the operator has started and is watching for ClientApp resources.
  ```sh
  kubectl logs deployment/clientapp-controller-manager -n system
  ```

### Creating a ClientApp
To create a ClientApp instance, create a YAML file (e.g., clientapp-instance.yaml) with the following content:
```yaml
apiVersion: custom.clinia.com/v1alpha1
kind: Clientapp
metadata:
  labels:
    app.kubernetes.io/name: clientapp
    app.kubernetes.io/instance: clientapp-sample
    app.kubernetes.io/part-of: clientapp2
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/created-by: clientapp2
  name: clientapp-sample
spec:
   # TODO(user): Add fields here
  name : nginx-deployment
  replicas : 5
  resources : 
    cpu: "1000m"  # 500 milliCPU
    memory: "512Mi"  # 512 Megabytes
  env: 
  - name : DEMO_ENV
    value : "Hello world"
  image : nginx:1.25
  host: nginx.example.com
  port : 80

  
```
### Updating a ClientApp
Update the fields of the ClientApp custom resource as needed. For example, to change the container image:
```yaml
apiVersion: custom.clinia.com/v1alpha1
kind: Clientapp
metadata:
  labels:
    app.kubernetes.io/name: clientapp
    app.kubernetes.io/instance: clientapp-sample
    app.kubernetes.io/part-of: clientapp2
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/created-by: clientapp2
  name: clientapp-sample
spec:
   # TODO(user): Add fields here
  name : nginx-deployment
  replicas : 6
  resources : 
    cpu: "1000m"  # 500 milliCPU
    memory: "512Mi"  # 512 Megabytes
  env: 
  - name : DEMO_ENV
    value : "Hello world"
  image : nginx:1.25
  host: nginx.example.com
  port : 80

```
Apply the changes:
```sh
kubectl apply -f clientapp-instance.yaml
```

### Deleting a ClientApp
To delete a ClientApp instance, use the following command:
``` sh
kubectl delete clientapp example-clientapp
```
### Modifying the API definitions
If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```
## 3. Cleanup
To clean up the resources created by the operator, run the following commands:
``` sh
kubectl delete -f ClientApp/config/samples/clinia_v1alpha1_clientapp.yaml
make uninstall
```