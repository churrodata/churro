# Churro - ETL for Kubernetes

churro is a cloud-native Extract-Transform-Load (ETL) application designed to build, scale, and manage data pipeline applications.

* What is churro?
* Docs
* churro Cloud
* Starting with churro
* Need Help?
* Contributing
* Design


## What is churro?
churro is an application that processes input files and streams by extracting content and loading that content into a database of your choice, all running on Kubernetes.

## Docs
Detailed documentation is found on the [churrodata.com website](https://www.churrodata.com) or at the [churro github pages site](https://churrodata.github.io/churro/).

## churro Cloud
Inquires about churro Cloud can be directed to info@churrodata.com.  In the near future, users will be able to provision a churro instance on the cloud of their choice, with billing and management handled by churrodata.com

## Starting with churro
People generally start with churro by creating a kubernetes cluster, then running the churro Makefile and following the churro documentation to deploy churro to your running cluster.  Installation documentation is found on the [churro github pages website](https://churrodata.github.io/churro/churro-Installation-Guide).

## Contributing
Since churro is open source, you can view the source code and make contributions such as pull requests on our github.com site.

## Design
Some key aspects of the churro design:
* churro is designed from the start to run within a Kubernetes cluster.
* churro uses a micro-service architecture to scale ETL processing
* churro has extension points defined to allow for very customized processing to be performed per customer requirements.
* churro is written in golang
* churro currently supports persisting to cockroachdb, singlestore, and mysql databases
* churro implements a kubernetes operator to handle gitops style provisioning of churro pipeline resources

For more details on the churro design, checkout out the documentation at the [churro github pages website](https://churrodata.github.io/churro/churro-User-Guide).
