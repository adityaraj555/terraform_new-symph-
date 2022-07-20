#==================================================================================================
# COMMON PARAMETERS
#==================================================================================================

SERVICE ?= symphony-service
VERSION      ?= $(shell cat version | head -n 1)
REVISION     ?= $(shell git rev-parse --short HEAD)
BUILD_NUMBER ?= none
DOCKER_TAG ?= latest
BRANCH_NAME      ?= develop
#DOCKER_REPO_NAME ?= 906858053434.dkr.ecr.us-east-2.amazonaws.com/symphony/symphony-service/$(LAMBDA) 
COVERAGE_REPORT_SERVER_PORT ?= 3001
COVERPROFILE=.cover.out
COVERDIR=.cover
#==================================================================================================
# DEVELOPMENT TASKS
#==================================================================================================
export PROJECT_DIR = $(shell pwd)

dep:
	@go get ./...
	
# run: 
# 	@go run ./lib/main.go serve

test: 
	@go test -coverprofile=$(COVERPROFILE) ./...

local-cover: test
	@mkdir -p $(COVERDIR)
	@go tool cover -html=$(COVERPROFILE) -o $(COVERDIR)/index.html
	@cd $(COVERDIR) && python -m SimpleHTTPServer $(COVERAGE_REPORT_SERVER_PORT)

cover: test
	@mkdir -p $(COVERDIR)
	@go tool cover -html=$(COVERPROFILE) -o $(COVERDIR)/index.html

clean:
	@rm -rf bin
 
.PHONY: dep run test cover clean build image docker-push tag-image ecr-login

generate-mocks:
	mockery --all --output ./commons/mocks

#==================================================================================================
# BUILDING RELEASE EXECUTABLES
#==================================================================================================

# Lambda
build-lambda: clean
	GOOS=linux go build -o ./bin/main ./lambdas/$(LAMBDA)/main.go

#==================================================================================================
# BUILDING DOCKER IMAGES
#==================================================================================================
ecr-login:
	@`aws ecr get-login --registry-ids 906858053434 --no-include-email --region us-east-2`
	
image:
	docker build -t $(SERVICE)/$(LAMBDA):$(DOCKER_TAG) -f build/docker/lambda.Dockerfile .

docker-push: image tag-image ecr-login
	docker push $(DOCKER_REPO_NAME):$(BRANCH_NAME)-latest

tag-image:
	@echo 'create tag latest'
	docker tag $(SERVICE):$(DOCKER_TAG) $(DOCKER_REPO_NAME):$(BRANCH_NAME)-latest