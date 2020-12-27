TEST_PROJECT_NAME:=example_test

.PHONY: \
	build \
	deploy \
	submit \
	test \
	up \

build:
	go build cmd/main.go

test:
	GCP_PROJECT=$(TEST_PROJECT_NAME) \
	GCSR_SSH_KEY_USER=user@example.com \
	go test

up:
	docker-compose up --build

submit:
	gcloud builds submit --config cloudbuild.yaml .

deploy:
	gcloud beta run deploy \
		repeater4gcsr \
		--concurrency=1 \
		--image gcr.io/${GCP_PROJECT}/repeater4gcsr:latest \
		--max-instances=1 \
		--min-instances=0 \
		--platform managed \
		--project=${GCP_PROJECT} \
		--region=asia-northeast1 \
		--timeout=15m \

deploy-functions:
	gcloud functions deploy repeater4gcsr \
		--egress-settings all \
		--entry-point Webhook \
		--ingress-settings internal-and-gclb \
		--region=asia-northeast1 \
		--runtime go113 \
		--service-account repeater4gcsr@${GCP_PROJECT}.iam.gserviceaccount.com \
		--set-env-vars GCSR_SSH_KEY_USER=${GCSR_SSH_KEY_USER} \
		--trigger-http \
		--vpc-connector projects/${GCP_PROJECT}/locations/asia-northeast1/connectors/repeater4gcsr-an1 \
