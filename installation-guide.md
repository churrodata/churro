# churro Installation Guide

- [Environment Requirements](Development Host Setup)
- [Download churro](churro-download)
- [Deploy the churro operator](Deploy churro)
- [Deploy the churro Web Console](Deploy churro Web Console)
- [Check the Install](Validate Installation)
- [sftp Setup](sftp-setup)

## Development Host Setup

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

## churro-download

TODO

## Deploy churro

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

#### Deploy cockroachdb Operator
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


## Deploy churro Web Console
In the current version of churro, only a single churro web console is deployed.  This web console serves up a HTML interface so users and admins can deploy and manage churro pipelines.

### Deploy
To deploy the web console application:
```
make deploy-churro-web-console
```
This will deploy a single Deployment with a single Pod.  By default the web console will 
be configured to look for a cockroach database instance to write to.

### Verify
You can see the running web console by the following command:
```
kubectl -n churro get pod --selector=name=churro-ui
NAME                         READY   STATUS    RESTARTS   AGE
churro-ui-66f8df8687-h5fmv   1/1     Running   0          3m58s
```

Once the web console is running, it will create some database tables for churro to write
state into, you can verify this by looking at the tables it creates:
```
make run-web-console-db-client
root@:26257/defaultdb> show tables from defaultdb;
  schema_name |     table_name     | type  | estimated_row_count
--------------+--------------------+-------+----------------------
  public      | authenticateduser  | table |                   0
  public      | userpipelineaccess | table |                   0
  public      | userprofile        | table |                   1
(4 rows)

Time: 30ms total (execution 22ms / network 9ms)
root@:26257/defaultdb> quit
```

### Port Forward
By default, the web console has a ClusterIP Service type that requires you to
port forward to gain access on your local development host.

The following make target will create a forwarded port (8080) on your local dev
box:
```
make port-forward
```
You can then access it via the url https://<your hostname>:8080

The web console is bootstrapped with an administrator user ID of `admin@admin.org` and
a password of `admin`.

## Validate Installation

After you complete the churro installation, you should be able to create a Pipeline to further validate your installation as being complete.

### Create Pipeline
Here are the steps you should be able to perform:
 * Browse to the https://<your host>:8080 web console application page
log in using the bootstrap admin credentials (admin@admin.org/admin)
 * Click the Create Pipeline button
 * Fill out the form choosing a pipeline name of pipeline1, the name has to be all lowercase and shorter than 20 characters with only a-z and 0-9 characters.
 * Wait till the status of the pipeline turns green

Once the status goes green it means you have a running pipeline.  You can verify
this by looking on your kube cluster at the namespace created for the pipeline:
```
kubectl -n pipeline1 get pod
NAME                        READY   STATUS    RESTARTS   AGE
churro-ctl                  1/1     Running   0          3m8s
churro-extractsource        1/1     Running   0          3m8s
cockroachdb-0               1/1     Running   0          3m25s
cockroachdb-1               1/1     Running   0          3m25s
cockroachdb-2               1/1     Running   0          3m25s
cockroachdb-client-secure   1/1     Running   0          3m8s
```

## sftp-setup

### What is sftp?
sftp is a fairly mainstream means to upload or transfer files to and from different hosts.  In churro, we have included an sftp service within the churro-watch Pod to enable an easy means for users to upload data files directly into churro Data Source file folders.

### How it works
Within the churro-watch Pod, we include a new container, this container will run the sftp server from [text](https://github.com/drakkan/sftpgo).  We have configured this sftp server to mount the same file folder that crunchy-watch uses to process uploaded data source files (e.g. CSV, JSON, Excel, XML).

### sftp Service
By default, churro doesn't create a Service that exposes the sftp service.  You can
create one by running the following Makefile target:
```
make create-sftp-service
```

This service exposes the sftp port (2022) as well as the sftp web console port (8080).  You can now port-forward to this service or modify it to be a LoadBalancer service depending on your environment.

### Port Forward to the sftp Web Console
There is a Makefile target that you can use to port-forward to the sftp web console:
```
make port-forward-sftp-web
```

### sftp User Configuration Example
The sftp service includes a web console that allows you to define sftp users.  We use this console to add a churro user account that we will use to upload data files.

First, generate a unique key as follows:
```
ssh-keygen
```

This will create a public and private key such as /home/yourname/.ssh/[id_rsa, id_rsa.pub]

Within the sftp web console, we create a user that specifies the public key contents you generated.  You will also want to set the data directory to be */data*, this is where the churro-watch data source directory is mounted within the sftp container.  This mounted volume is shared with the churro-watch container.

### sftp Client Configuration

Next, we create a sftp config file (e.g. config.sftp) that can be used on the client host that a user would use to upload files to churro.  The config file looks like this:
````
Host localhost
User test2
IdentityFile /home/jeffmc/.ssh/id_rsa
````
You can upload files using sftp by using a command as follows:
```
sftp -F /home/yourname/config.sftp -P 2022 test2@yourhost
```

After connecting, you can look around the file directory that churro-watch uses to find
data files to process:
```
sftp> cd csvfiles
sftp> ls
ready                                   source1.csv.churro-processed
sftp> put source1.csv
```

The *put* sftp command uploads a file *source1.csv* from your current local client directory to the remote server directory.  Effectively this will cause churro-watch to process the uploaded file.


