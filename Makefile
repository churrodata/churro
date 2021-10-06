SAMPLE_WATCH_DIRS=/churro
GRPC_CERTS_DIR=certs/grpc
DB_CERTS_DIR=certs/db
BUILDDIR=./build
PIPELINE=single
CHURRO_NS=churro
TAG=0.0.2
PLATFORMS="linux/amd64,linux/arm64"
images=churro-ui churro-operator churro-sftp churro-ctl churro-extract churro-extractsource

## pick the database you want to use for the web console
#DATABASE=singlestore
#DATABASE=mysql
DATABASE=cockroachdb

.DEFAULT_GOAL := all

setup-env:
	echo 'downloading build dependencies...'
	echo be sure to add $(HOME)/go/bin to your PATH
	go get -u github.com/golang/protobuf/protoc-gen-go
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc
	wget https://github.com/protocolbuffers/protobuf/releases/download/v3.12.4/protoc-3.12.4-linux-x86_64.zip
	unzip protoc-3.12.4-linux-x86_64.zip 'bin/protoc' -d $(HOME)/go
	go get -u go101.org/gold
	which protoc-gen-go

## Create certificates for a pipeline
cert:
	$(BUILDDIR)/gen-certs.sh certs $(PIPELINE)
uicert:
	openssl genrsa -out certs/ui/https-server.key 2048
	openssl ecparam -genkey -name secp384r1 -out certs/ui/https-server.key
	openssl req -new -x509 -sha256 -key certs/ui/https-server.key -out certs/ui/https-server.crt -days 3650

gen-crd:
	controller-gen "crd:trivialVersions=true,crdVersions=v1" paths="./..." output:crd:artifacts:config=deploy/operator
gen-docs:
	echo 'generating docs...'
	gold -gen -dir=/tmp ./...
unit-test:
	echo 'unit tests...'
	go test ./cmd/... ./internal... -v
bounce-ui:
	kubectl -n churro delete deploy/churro-ui
bounce-operator:
	kubectl -n churro delete deploy/churro-operator --ignore-not-found=true
	kubectl -n churro create -f  deploy/operator/churro-operator-deployment.yaml
undeploy-churro-web-console:
	kubectl -n $(CHURRO_NS) delete --ignore-not-found=true \
		deploy/churro-ui \
		service/churro-ui \
		clusterrole/churro-ui \
		clusterrolebinding/churro-ui \
		sa/churro-ui
deploy-testdb:
	#deploy/mysql/secret.sh
	kubectl -n testdb create -f deploy/mysql/sample-cluster.yaml
deploy-churro-web-console:
	kubectl -n $(CHURRO_NS) create -f deploy/ui/service-account.yaml
	kubectl -n $(CHURRO_NS) create -f deploy/ui/cluster-role.yaml
	kubectl -n $(CHURRO_NS) create -f deploy/ui/cluster-role-binding.yaml
	kubectl -n $(CHURRO_NS) create -f deploy/ui/service.yaml
	kubectl -n $(CHURRO_NS) create -f deploy/ui/$(DATABASE)/churro-ui.yaml
update-templates:
	kubectl -n $(CHURRO_NS) delete configmap churro-templates --ignore-not-found=true
	kubectl -n $(CHURRO_NS) create configmap churro-templates --from-file=deploy/templates
undeploy-cockroach-operator:
	kubectl -n $(CHURRO_NS) delete -f deploy/cockroachdb/cr.yaml --ignore-not-found=true
	kubectl -n $(CHURRO_NS) delete -f deploy/cockroachdb/operator.yaml --ignore-not-found=true
	kubectl delete -f deploy/cockroachdb/crd.yaml --ignore-not-found=true
	kubectl -n $(CHURRO_NS) delete -f deploy/cockroachdb/rbac.yaml --ignore-not-found=true
	kubectl -n $(CHURRO_NS) delete pvc --all
undeploy-db-operators: undeploy-cockroach-operator undeploy-singlestore-operator
deploy-db-operators: deploy-cockroach-operator deploy-singlestore-operator
deploy-cockroach-operator:
	kubectl create -f deploy/cockroachdb/crd.yaml
	kubectl -n $(CHURRO_NS) create -f deploy/cockroachdb/rbac.yaml
	kubectl -n $(CHURRO_NS) create -f deploy/cockroachdb/operator.yaml
	kubectl -n $(CHURRO_NS) create -f deploy/cockroachdb/cr.yaml
deploy-singlestore-operator:
	kubectl create -f deploy/singlestore/crd.yaml
	kubectl -n $(CHURRO_NS) create -f deploy/singlestore/rbac.yaml
	kubectl -n $(CHURRO_NS) create -f deploy/singlestore/operator.yaml
	kubectl -n $(CHURRO_NS) create -f deploy/singlestore/cr.yaml
undeploy-singlestore-operator:
	kubectl -n $(CHURRO_NS) delete -f deploy/singlestore/cr.yaml --ignore-not-found=true
	kubectl -n $(CHURRO_NS) delete memsqlclusters --all --ignore-not-found=true
	kubectl delete -f deploy/singlestore/rbac.yaml --ignore-not-found=true
	kubectl -n $(CHURRO_NS) delete role/memsql-operator --ignore-not-found=true
	kubectl -n $(CHURRO_NS) delete rolebinding/memsql-operator --ignore-not-found=true
	kubectl -n $(CHURRO_NS) delete sa/memsql-operator --ignore-not-found=true
	kubectl -n $(CHURRO_NS) delete secret --selector=app.kubernetes.io/name=memsql-cluster --ignore-not-found=true
	kubectl -n $(CHURRO_NS) delete deploy/memsql-operator
	kubectl delete crd/memsqlclusters.memsql.com --ignore-not-found=true
	kubectl -n $(CHURRO_NS) delete pvc --selector=app.kubernetes.io/name=memsql-cluster --ignore-not-found=true
undeploy-mysql-operator:
	kubectl -n $(CHURRO_NS) delete secret/churro-ui-mysql-secret
	kubectl -n $(CHURRO_NS) delete mysqlcluster/churro-ui-mysql
	helm uninstall --namespace $(CHURRO_NS) mysql-operator
deploy-mysql-operator:
	#helm repo add presslabs https://presslabs.github.io/charts
	helm install --namespace $(CHURRO_NS) mysql-operator presslabs/mysql-operator
	kubectl -n $(CHURRO_NS) create -f deploy/ui/mysql/churro-ui-mysql-secret.yaml
	kubectl -n $(CHURRO_NS) create -f deploy/ui/mysql/churro-ui-mysql.yaml
regcred:
	kubectl -n churro create secret generic regcred \
		--from-file=.dockerconfigjson=/home/jeffmc/.docker/config.json \
		--type=kubernetes.io/dockerconfigjson
bounce-templates:
	kubectl -n $(CHURRO_NS) delete configmap churro-templates --ignore-not-found=true
	kubectl -n $(CHURRO_NS) create configmap churro-templates --from-file=deploy/templates
undeploy-churro-operator:
	kubectl -n $(CHURRO_NS) delete deploy churro-operator --ignore-not-found=true
	kubectl delete clusterrole churro-operator --ignore-not-found=true
	kubectl delete clusterrolebinding churro-operator --ignore-not-found=true
	kubectl -n $(CHURRO_NS) delete serviceaccount churro-operator --ignore-not-found=true
	kubectl -n $(CHURRO_NS) delete configmap churro-templates --ignore-not-found=true
	kubectl delete crd pipelines.churro.project.io --ignore-not-found=true
	kubectl delete crd mysqlclusters.mysql.presslabs.org --ignore-not-found=true
deploy-churro-operator:
	build/namespace-check.sh $(CHURRO_NS)
#	kubectl -n $(CHURRO_NS) create configmap churro-templates --from-file=deploy/templates
#	kubectl create -f deploy/operator/churro.project.io_pipelines.yaml
#	kubectl create -f deploy/ui/mysql/mysql.presslabs.org_mysqlclusters.yaml
#	kubectl create -f deploy/operator/churro-ui-crd.yaml
#	kubectl create -f deploy/operator/cluster-role.yaml
#	kubectl create -f deploy/operator/cluster-role-binding.yaml
	kubectl -n $(CHURRO_NS) create -f deploy/operator/service-account.yaml
	kubectl -n $(CHURRO_NS) create -f deploy/operator/churro-operator-deployment.yaml
push:
	for i in $(images); do \
		echo $$i; \
	    docker push docker.io/churrodata/$$i:latest; \
	done
	#docker push docker.io/churrodata/churro/memsql-studio:latest


build-memsql-studio: 
	docker build -f ./images/Dockerfile.memsql-studio -t docker.io/churrodata/memsql-studio .

compile-ui:
	go build -o build/churro-ui ui/main.go
build-ui-image-local: compile-ui
	docker build -f ./images/Dockerfile.churro-ui.local -t docker.io/churrodata/churro-ui .
build-ui-image:
	docker buildx build --push --platform $(PLATFORMS) -f ./images/Dockerfile.churro-ui -t docker.io/churrodata/churro-ui:$(TAG) .


compile-extract:
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=require_unimplemented_servers=false:. --go-grpc_opt=paths=source_relative rpc/extract/extract.proto
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=require_unimplemented_servers=false:. --go-grpc_opt=paths=source_relative rpc/extension/extension.proto
	go build -o build/churro-extract cmd/churro-extract/churro-extract.go

build-extract-image-local: compile-extract
	docker build  -f ./images/Dockerfile.churro-extract.local -t docker.io/churrodata/churro-extract .
build-extract-image:
	docker buildx build --push --platform $(PLATFORMS) -f ./images/Dockerfile.churro-extract -t docker.io/churrodata/churro-extract:$(TAG) .


compile-extractsource:
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=require_unimplemented_servers=false:. --go-grpc_opt=paths=source_relative rpc/extractsource/extractsource.proto
	go build -o build/churro-extractsource cmd/churro-extractsource/churro-extractsource.go

build-extractsource-image-local: 

	docker build -f ./images/Dockerfile.churro-extractsource -t docker.io/churrodata/churro-extractsource .

build-extractsource-image: 
	docker buildx build --push --platform $(PLATFORMS) -f ./images/Dockerfile.churro-extractsource -t docker.io/churrodata/churro-extractsource:$(TAG) .


compile-ctl:
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=require_unimplemented_servers=false:. --go-grpc_opt=paths=source_relative rpc/ctl/ctl.proto
	go build -o build/churro-ctl cmd/churro-ctl/churro-ctl.go

build-ctl-image-local: 
	docker build -f ./images/Dockerfile.churro-ctl -t docker.io/churrodata/churro-ctl .
build-ctl-image: 
	docker buildx build --push --platform $(PLATFORMS) -f ./images/Dockerfile.churro-ctl -t docker.io/churrodata/churro-ctl:$(TAG) .


build-sftp-image-local: 
	docker build -f ./images/Dockerfile.churro-sftp -t docker.io/churrodata/churro-sftp:latest .
build-sftp-image: 

	docker buildx build --push --platform $(PLATFORMS) -f ./images/Dockerfile.churro-sftp -t docker.io/churrodata/churro-sftp:$(TAG) .


compile-operator:
	go build -o build/churro-operator cmd/churro-operator/churro-operator.go
build-operator-image-local:  compile-operator
	docker build -f ./images/Dockerfile.churro-operator.local -t docker.io/churrodata/churro-operator:latest .
build-operator-image: 
	docker buildx build --push --platform $(PLATFORMS) -f ./images/Dockerfile.churro-operator -t docker.io/churrodata/churro-operator:$(TAG) .


port-forward-httppost:
	kubectl -n $(PIPELINE) port-forward svc/my-httppost-files --address `hostname --ip-address` 10000:10000 
port-forward:
	kubectl -n churro port-forward svc/churro-ui --address `hostname --ip-address` 8080:8080 
create-sftp-service:
	kubectl -n $(PIPELINE) create -f deploy/templates/churro-sftp-svc.yaml
port-forward-sftp:
	kubectl -n $(PIPELINE)  port-forward svc/churro-extractsource-sftp --address `hostname --ip-address` 2022:2022 
port-forward-sftp-web:
	kubectl -n $(PIPELINE)  port-forward svc/churro-extractsource-sftp --address `hostname --ip-address` 8080:8080 
port-forward-db-console:
	kubectl -n $(PIPELINE) port-forward svc/cockroachdb-public --address `hostname --ip-address` 26257:26257 
port-forward-db-console-singlestore:
	kubectl -n $(PIPELINE) port-forward --address `hostname --ip-address`  pod/memsql-studio 10000:8080
port-forward-ui-db:
	kubectl -n churro port-forward svc/cockroachdb-public --address `hostname --ip-address` 26257:26257 

compile: compile-operator compile-ctl compile-extractsource compile-extract

all: build-sftp-image-local build-extract-image-local build-extractsource-image-local build-ctl-image-local build-operator-image-local build-ui-image-local

release: build-sftp-image build-extract-image build-extractsource-image build-ctl-image build-operator-image build-ui-image

pipeline-certs:
	$(BUILDDIR)/gen-certs.sh certs $(PIPELINE)

gen-godoc-site:
	godoc-static \
		-site-name="churro Documentation" \
		-destination=./doc/godoc \
		gitlab.com/churrodata/churro/internal/backpressure 
start-prometheus:
	./deploy/prometheus/start-prometheus.sh
run-db-client:
	kubectl -n $(PIPELINE) exec -it cockroachdb-client-secure -- ./cockroach sql --certs-dir=/cockroach-certs --host=cockroachdb-public
run-singlestoredb-client:
	mysql -u admin -h 10.109.21.97 -P 3306 -pxxxxxxx
run-web-console-db-client:
	kubectl -n $(CHURRO_NS) exec -it cockroachdb-2 -- ./cockroach sql --certs-dir=/cockroach/cockroach-certs
backup-ui-db:
	kubectl -n $(CHURRO_NS) exec -it cockroachdb-2 -- ./cockroach dump defaultdb --certs-dir=/cockroach/cockroach-certs
backup-db:
	kubectl -n $(PIPELINE) exec -it cockroachdb-2 -- ./cockroach dump pipeline1 --certs-dir=/cockroach/cockroach-certs
.PHONY: clean

clean:
	rm $(BUILDDIR)/churro*
	rm /tmp/churro*.log
