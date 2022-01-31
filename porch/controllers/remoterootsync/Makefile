# GCP project to use for development
GCP_PROJECT_ID ?= $(shell gcloud config get-value project)

.PHONY: run-local
run-local:
	GCP_PROJECT_ID=${GCP_PROJECT_ID} HACK_ENABLE_LOOPBACK=1 go run .