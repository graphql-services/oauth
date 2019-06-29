package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"

	trace "cloud.google.com/go/trace/apiv1"
	gcloudtracer "github.com/lovoo/gcloud-opentracing"
	basictracer "github.com/opentracing/basictracer-go"
	opentracing "github.com/opentracing/opentracing-go"
	"google.golang.org/api/option"
)

type Tracer struct {
	recorder *gcloudtracer.Recorder
}

func (t *Tracer) Initialize() error {
	filename := os.Getenv("GOOGLE_CREDENTIALS_FILE")
	if filename == "" {
		filename = "credentials.json"
	}
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil
	}

	client, err := trace.NewClient(context.Background(), option.WithCredentialsFile(filename))
	if err != nil {
		log.Fatalf("error creating a tracing client: %v", err)
	}

	creds, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	var credsObj struct {
		ProjectID string `json:"project_id"`
	}
	err = json.Unmarshal(creds, &credsObj)
	if err != nil {
		return err
	}

	recorder, err := gcloudtracer.NewRecorder(context.Background(), credsObj.ProjectID, client)
	if err != nil {
		log.Fatalf("error creating a recorder: %v", err)
	}
	t.recorder = recorder

	opentracing.SetGlobalTracer(basictracer.NewWithOptions(basictracer.Options{
		Recorder: recorder,
		ShouldSample: func(traceID uint64) bool {
			return true
		},
	}))

	return nil
}

func (t *Tracer) Close() error {
	if t.recorder == nil {
		return nil
	}
	return t.recorder.Close()
}
