all: modules build

AUTH_DOCKER_IMAGE_NAME=nailkabirov/arch-project-auth
AUTH_DOCKER_IMAGE_TAG=v0.0.2

USER_DOCKER_IMAGE_NAME=nailkabirov/arch-project-user
USER_DOCKER_IMAGE_TAG=v0.0.1

LOT_DOCKER_IMAGE_NAME=nailkabirov/arch-project-lot
LOT_DOCKER_IMAGE_TAG=v0.0.3

BILLING_DOCKER_IMAGE_NAME=nailkabirov/arch-project-billing
BILLING_DOCKER_IMAGE_TAG=v0.0.2

DELIVERY_DOCKER_IMAGE_NAME=nailkabirov/arch-project-delivery
DELIVERY_DOCKER_IMAGE_TAG=v0.0.1

NOTIFICATION_DOCKER_IMAGE_NAME=nailkabirov/arch-project-notification
NOTIFICATION_DOCKER_IMAGE_TAG=v0.0.1

APP_NAMES = \
	auth \
	billing \
	lot \
	user

.PHONY: modules
modules:
	go mod tidy

.PHONY: build
build: modules check test build_auth build_billing build_delivery build_lot build_notification build_user

.PHONY: build_auth
build_auth:
	docker build -t $(AUTH_DOCKER_IMAGE_NAME):$(AUTH_DOCKER_IMAGE_TAG) -f Auth.Dockerfile .

.PHONY: build_delivery
build_delivery:
	docker build -t $(DELIVERY_DOCKER_IMAGE_NAME):$(DELIVERY_DOCKER_IMAGE_TAG) -f Delivery.Dockerfile .

.PHONY: build_user
build_user:
	docker build -t $(USER_DOCKER_IMAGE_NAME):$(USER_DOCKER_IMAGE_TAG) -f User.Dockerfile .

.PHONY: build_lot
build_lot:
	docker build -t $(LOT_DOCKER_IMAGE_NAME):$(LOT_DOCKER_IMAGE_TAG) -f Lot.Dockerfile .

.PHONY: build_notification
build_notification:
	docker build -t $(NOTIFICATION_DOCKER_IMAGE_NAME):$(NOTIFICATION_DOCKER_IMAGE_TAG) -f Notification.Dockerfile .

.PHONY: build_billing
build_billing:
	docker build -t $(BILLING_DOCKER_IMAGE_NAME):$(BILLING_DOCKER_IMAGE_TAG) -f Billing.Dockerfile .

.PHONY: build_local
build_local: modules check test
	$(foreach name, $(APP_NAMES), CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o ./bin/$(name) ./cmd/$(name)/;)

.PHONY: test
test:
	go test ./...

.PHONY: clean
clean:
	rm -f $(foreach name,$(APP_NAMES), "bin/$(name)")

.PHONY: check
check:
	golangci-lint run