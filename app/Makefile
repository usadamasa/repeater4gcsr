.PHONY: \
	submit \
	deploy \

build:
	go build cmd/main.go

submit:
	gcloud builds submit --tag gcr.io/${GCP_PROJECT}/repeater4gcsr

deploy:
	gcloud beta run deploy \
		repeater4gcsr \
		--image gcr.io/${GCP_PROJECT}/repeater4gcsr:latest \
		--platform managed \
		--project=${GCP_PROJECT} \
		--region=asia-northeast1 \
		--set-env-vars GCSR_SSH_KEY_USER=${GCSR_SSH_KEY_USER} \

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
