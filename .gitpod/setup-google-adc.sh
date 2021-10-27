#!/bin/bash

GCLOUD_ADC_PATH="/home/gitpod/.config/gcloud/application_default_credentials.json"

if [ ! -f "$GCLOUD_ADC_PATH" ]; then
    if [ -z "$GCP_ADC_FILE" ]; then
        echo "GCP_ADC_FILE not set, doing nothing."
        return;
    fi
    echo "$GCP_ADC_FILE" > "$GCLOUD_ADC_PATH"
    echo "Set GOOGLE_APPLICATION_CREDENTIALS value based on contents from GCP_ADC_FILE"

    gcloud auth activate-service-account --key-file $GCLOUD_ADC_PATH
    gcloud config set project $GOOGLE_PROJECT_ID

    export TF_VAR_project=$GOOGLE_PROJECT_ID
fi

export GOOGLE_APPLICATION_CREDENTIALS="$GCLOUD_ADC_PATH"