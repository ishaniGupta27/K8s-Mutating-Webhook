WEBHOOK_SERVICE?=test-webhook-service
NAMESPACE?=default
CONTAINER_REPO?=ishani2727/k8s-mutating-webhook
CONTAINER_VERSION?=1.1
CONTAINER_IMAGE=$(CONTAINER_REPO):$(CONTAINER_VERSION)

.PHONY: docker-build
docker-build:
	docker build -t $(CONTAINER_IMAGE) .

.PHONY: docker-push
docker-push:
	docker push $(CONTAINER_IMAGE)