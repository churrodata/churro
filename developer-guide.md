# churro Developer Guide

Here are some links useful for people wanting to develop on churro
or build churro extensions:

* [Development Environment](#Development Environment)
* [Building a churro extension](#Building a churro extension)
* [Build churro](#Build churro)
* [Deploying](#Deploying)

## Development Environment
Here is a typical developer environment that we use to develop and test churro:

### Operator System
Fedora 33

### Kubernetes 
Kubernetes 1.20

We tend to test locally with k8s installed via kubeadm. 

[kubeadm](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/install-kubeadm/)

For running on Fedora 33 you will need to disable swap as follows:
```bash=
sudo touch /etc/systemd/zram-generator.conf
sudo touch /etc/systemd/zram-generator.conf
```

I tend to also disable selinux and the firewalld for local development:
```bash=
systemctl disable firewalld
systemctl stop firewalld
vi /etc/sysconfig/selinux
```

Make sure your host can resolve to a single hostname and IP address that won't change.  Kubernetes needs to have your hostname resolve to an IP address that can be found in /etc/hosts for example.
```bash=
sudo hostnamectl set-hostname yourhostname
sudo vi /etc/hosts (add yourhostname to your /etc/hosts file)
```


### golang Compiler
go version 1.15.7

### Kubernetes Storage Class
You need at least a single StorageClass for churro to use.  We tend to use
the hostpath provisioner found here:
[rimusz hostpath provisioner](https://charts.rimusz.net)

```bash=
helm repo add rimusz https://charts.rimusz.net
helm install my-hostpath-provisioner rimusz/hostpath-provisioner --version 0.2.10
```

WARNING!!!  https://github.com/kubernetes-sigs/nfs-subdir-external-provisioner/issues/25#issuecomment-742616668
You need to apply that bugfix with k8s 1.20 in order to get hostpath storage to work.


Other combinations might work as well, we tend to use the latest and greatest of the building blocks (kube, golang)

### Source Code
Access to the churro source code is granted with a valid license which can be obtained by
request with churrodata directly.

You would access the source as follows:
```
git clone git@gitlab.com:churro-group/churro
```

### Install gRPC Tools

```bash
sudo dnf -y install protobuf-compiler
go get google.golang.org/protobuf/cmd/protoc-gen-go \
         google.golang.org/grpc/cmd/protoc-gen-go-grpc
```

### Configure Docker
For running churro with cockroachdb, you will need to configure Docker to support a larger than default set of file limits:
```bash=
sudo vi /etc/sysconfig/docker
Add OPTIONS=”--default-ulimit nofile=50000:50000” to the OPTIONS variable

sudo vi /etc/systemd/system.conf
Add DefaultLimitNOFILE=4096:524288
```

## Building a churro extension

churro ships a sample Extension, called the spacex-geo-extension, you can clone that code as follows to see how an extension works and have a starting point to write your own extension.

```
git clone git@gitlab.com:churro-public/spacex-geo-extension
```

In that example, you can build the extension using the included Makefile:
```
make
```

You will end up with the following container image:
```
cortado:[~/spacex-geo-extension] docker images | grep spacex
registry.gitlab.com/churro-public/spacex-geo-extension      latest               2ebef3de0aae        20 seconds ago      222MB
```

## Build churro

At this point you should have your development box setup and configured.  Also you should have the churro source code cloned.

churro development centers around targets within its Makefile.

## Compile churro
```
make
```

This command will compile all the churro binaries and also build the churro container images locally.

You can verify that your local container repository has the following images:
```
cortado:[~/churro] docker images | grep churro
registry.gitlab.com/churro-group/churro/churro-ui          latest               b72f6589b438        57 seconds ago       239MB
registry.gitlab.com/churro-group/churro/churro-operator    latest               0c14ab1ee9ea        About a minute ago   223MB
registry.gitlab.com/churro-group/churro/churro-ctl         latest               c62cb9c80a5b        About a minute ago   470MB
registry.gitlab.com/churro-group/churro/churro-extractsource       latest               4c468219155a        About a minute ago   222MB
registry.gitlab.com/churro-group/churro/churro-extract     latest               b094fa79549b        About a minute ago   241MB
registry.gitlab.com/churro-group/churro/churro-sftp            latest          0202897f6f31   6 days ago      193MB



## Deploying

### Deploy churro

This document describes the steps to set up churro on an Azure Kubernetes Cluster (AKS), the same steps will generally apply to other Kube cluster/environments.  Users will want to adjust
the various configuration steps below to match their particular environment.

#### Create Namespace

```
kubectl create namespace churro
```

#### Create Pull Secret

Since churro images are currently private, you will need to be able to authenticate to the churro container registry which will update your Docker config with credentials used for an image pull Secret, in the churro container manifests we'll refer to that image pull Secret.

```
kubectl -n churro create secret generic regcred \
    --from-file=.dockerconfigjson=<path/to/.docker/config.json> \
    --type=kubernetes.io/dockerconfigjson
```

## Deploy cockroachdb Operator
This operator is used only for running the cockroachdb instance used by the churro web console if you specify a databasetype value of cockroachdb in the churroui CR. 

```
kubectl create -f deploy/cockroachdb/crd.yaml
kubectl -n churro create -f deploy/cockroachdb/rbac.yaml
kubectl -n churro create -f deploy/cockroachdb/operator.yaml
```

Wait for the cockroachdb operator to be in running status.

```
kubectl -n churro get pod
```

#### Deploy cockroachdb Instance

```
kubectl -n churro create -f deploy/cockroachdb/cr.yaml
```

Wait for the 3 cockroach pods to be in running status.

```
kubectl -n churro get pod
NAME                                  READY   STATUS    RESTARTS   AGE
cockroach-operator-5864f4f767-w5hgx   1/1     Running   0          14m
cockroachdb-0                         1/1     Running   0          2m
cockroachdb-1                         1/1     Running   0          73s
cockroachdb-2                         1/1     Running   0          41s
```

#### Deploy churro operator
```
make deploy-churro-operator
```

#### Customize the churroui CR

You will want to customize the churroui CR before you create it to fit your installation environment.  Here is an example of settings that are possible:
```yaml
apiVersion: churro.project.io/uiv1alpha1
kind: Churroui
metadata:
  name: fuzzy
  labels:
    name: fuzzy
status:
  active: "true"
  standby:
    - "AAPL"
    - "AMZN"
spec:
  databasetype: cockroachdb
  servicetype: LoadBalancer
  storageclassname: default
  storagesize: 1G
  accessmode:  ReadWriteOnce
```

| setting  | meaning  | default  | possible values  |
|---|---|---|---|
|  databasetype  |  the web consoles database it will use | cockroachdb  |   |
|  servicetype |  the type of Service for the web console | ClusterIP  | ClusterIP or LoadBalancer  |
|  storageclassname |  the storage class name for the web console app | default storage class  |  must be an existing storage class name |
|  storagesize |  the size of volume to request for the web console app | default size for storage class  |  must be valid syntax for kube storage resource and storage class |  
|  accessmode | the storage access mode to use for the web console app  | ReadWriteMany  | ReadWriteMany or ReadWriteOnce  | 

#### Create the churroui CR

```
kubectl -n churro create -f deploy/operator/churro-ui-cr.yaml
```

Verify the churro web console is running:
```
kubectl -n churro get pod
NAME                                  READY   STATUS    RESTARTS   AGE
churro-operator-744b879c67-2bjqq      1/1     Running   5          5m57s
churro-ui-98fdcf565-5xrq5             1/1     Running   0          7s
cockroach-operator-5864f4f767-w5hgx   1/1     Running   0          50m
cockroachdb-0                         1/1     Running   0          38m
cockroachdb-1                         1/1     Running   0          37m
cockroachdb-2                         1/1     Running   0          36m
```

