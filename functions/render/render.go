package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/myaut/stellaris-galaxy-map/pkg/sgm"
	"github.com/myaut/stellaris-galaxy-map/pkg/sgmrender"
)

const (
	bucketMaps  = "sgm-maps"
	bucketSaves = "sgm-uploads"
)

var internalError = errors.New("internal error in SGM renderer")

type Request struct {
	Query map[string]string `json:"queryStringParameters"`
}

type ResponseBody struct {
	PermanentURL string `json:"permanent_url"`
}

type Response struct {
	StatusCode int          `json:"statusCode"`
	Body       ResponseBody `json:"body"`
}

func Render(ctx context.Context, request *Request) (*Response, error) {
	key := request.Query["key"]
	if len(key) == 0 {
		return &Response{StatusCode: http.StatusBadRequest}, nil
	}

	client, err := newS3Client(ctx)
	if err != nil {
		log.Printf("error initializing s3: %s", err.Error())
		return nil, internalError
	}

	mapKey := fmt.Sprintf(strings.Split(key, ".")[0] + ".svg")
	objects, err := client.ListObjects(ctx, &s3.ListObjectsInput{
		Bucket: aws.String(bucketMaps),
	})
	if err != nil {
		log.Printf("error checking existing map: %s", err.Error())
		return nil, internalError
	}
	if len(objects.CommonPrefixes) == 0 {
		saveFile, err := client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(bucketSaves),
			Key:    aws.String(key),
		})
		if err != nil {
			log.Printf("error loading save file: %s", err.Error())
			return nil, internalError
		}

		tmpFile, err := ioutil.TempFile(os.TempDir(), "sgm-render-")
		if err != nil {
			log.Printf("error opening temp file: %s", err.Error())
			return nil, internalError
		}
		defer tmpFile.Close()

		_, err = io.Copy(tmpFile, saveFile.Body)
		if err != nil {
			log.Printf("error writing temp save file: %s", err.Error())
			return nil, internalError
		}

		state, err := sgm.LoadGameState(tmpFile.Name())
		if err != nil {
			log.Printf("error loading game state: %s", err.Error())
			return nil, internalError
		}

		r := sgmrender.NewRenderer(state)
		r.Render()
		buf, err := r.WriteToBytes()
		if err != nil {
			log.Printf("error generating SVG: %s", err.Error())
			return nil, internalError
		}

		_, err = client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:      aws.String(bucketMaps),
			Key:         aws.String(mapKey),
			ContentType: aws.String("image/svg+xml"),
			Body:        bytes.NewBuffer(buf),
		})
		if err != nil {
			log.Printf("error writing SVG to S3: %s", err.Error())
			return nil, internalError
		}

		_, err = client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(bucketSaves),
			Key:    aws.String(key),
		})
		if err != nil {
			log.Printf("error deleting save file from S3: %s", err.Error())
		}
	}

	return &Response{
		StatusCode: http.StatusCreated,
		Body: ResponseBody{
			PermanentURL: fmt.Sprintf("https://storage.yandexcloud.net/%s/%s", bucketMaps, mapKey),
		},
	}, nil
}

func newS3Client(ctx context.Context) (*s3.Client, error) {
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if service == s3.ServiceID && region == "ru-central1" {
			return aws.Endpoint{
				PartitionID:   "yc",
				URL:           "https://storage.yandexcloud.net",
				SigningRegion: "ru-central1",
			}, nil
		}
		return aws.Endpoint{}, fmt.Errorf("unknown endpoint requested")
	})

	cfg, err := config.LoadDefaultConfig(ctx, config.WithEndpointResolverWithOptions(customResolver))
	if err != nil {
		return nil, err
	}

	return s3.NewFromConfig(cfg), nil
}
