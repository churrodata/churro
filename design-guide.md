# [churro](https://churrodata.github.com/churro) Design Guide

churro is made of various software components which are deployed to a k8s cluster.

## churro-ui
The web user interface is called churro-ui, it is deployed as a k8s Deployment and is fronted by a k8s Service which enables end user access via a browser.

![alt text](./images/pipeline-users.png)

## churro-operator
There is a k8s operator which runs as a k8s Pod with the k8s namespace called churro.  This application component acts to deploy and manage various churro pipelines and insures they are kept running.  You run a single churro-operator on your system.

When churro creates a pipeline, it creates a k8s namespace to hold the pipeline's churro application components which include the following:

## churro-ctl
The brain or controller of a pipeline is handled by the churro-ctl k8s Pod in a given pipeline's k8s namespace.  The churro-ui application communicates to the churro-ctl microservice within each pipeline's namespace.  The churro-ctl Pod is fronted by the churro-ui k8s Service.

## churro-extractsource
Ingested files are watched for by the churro-extractsource application component.  There is a single churro-extractsource Pod created in each pipeline's namespace.  The churro-extractsource components looks for watched directories and new files to process.  When a new input file is found, churro-extractsource creates a churro-extract k8s Pod to process the input file.

In the case of a JSON API data source, churro-extractsoure will create a churro-extract Pod to read that JSON API.  churro web console lets you start and stop JSON API extract pods.

There is a k8s Service, churro-extractsource, that fronts the deployed churro-extractsource component.  Currently only churro-ctl interacts with churro-extractsource.

## churro-extract
As new files are found to be processed, a churro-extract Pod is created to process that new file.  The churro-extract component performs the following functions:
 * reads the input file
 * applies any transformations to that file's data
 * loads the file data into the target database (e.g. cockroachdb)

Once the file is processed, the churro-extract Pod stops.  These churro-extract Pods are named using a unique naming suffix so that extracted files can be processed concurrently.

In the case of a JSON API data source, churro-extract Pods do not stop unless you delete them, so they will run and capture data on some configured polling interval (e.g. 5 minutes).  The churro web console lets you 

## cockroachdb
The churro target database for a pipeline is a dedicated cockroachdb database which consists of a StatefulSet which causes n-number of cockroachdb nodes to run within a pipeline's namespace.  Currently the default set of nodes is defined to 3 which is the minimum number of nodes for running cockroachdb.

There is a k8s Service named cockroachdb-public that fronts the deployed database.

## mysql
churro also supports the creation of churro pipeline backend databases using mysql.

## singlestore (memsql)
churro also supports the creation of churro pipeline backend databases using singlestore's memsql database.
