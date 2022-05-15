package main

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/http"
	"os"

	"github.com/google/uuid"
	s3presign "github.com/myaut/go-s3presigned-post"
)

const (
	region = "ru-central1"
	bucket = "sgm-uploads"
)

var (
	formText = template.Must(template.New("form").Parse(`
	<form action="{{ .Action }}" id="upload-form" method="post" enctype="multipart/form-data">
       <input type="hidden" name="acl" value="private" />
       <input type="hidden" name="key" value="{{ .Key }}" />
       <input type="hidden" name="x-amz-algorithm" value="AWS4-HMAC-SHA256" />
       <input type="hidden" name="x-amz-date" value="{{ .Date }}" />
       <input type="hidden" name="x-amz-credential" value="{{ .Credential }}" />
       <input type="hidden" name="x-amz-signature" value="{{ .Signature }}" />
       <input type="hidden" name="policy" value="{{ .Policy }}" />
       <input type="hidden" name="success_action_redirect" value="https://stellaris-galaxy-map.website.yandexcloud.net/map.html?{{ .Key }}" />
       
       <input type="file" name="file" id="upload-file" />
       <input type="submit" name="submit" id="upload-submit" value="Create Map" />
     </form>
`))
)

type Request struct {
}

type Response struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
}

func Form(ctx context.Context, request *Request) (*Response, error) {
	key := fmt.Sprintf("%s.sav", uuid.New().String())
	post, err := s3presign.NewPresignedPOST(key, &s3presign.Credentials{
		Region:          region,
		Bucket:          bucket,
		AccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
	}, &s3presign.PolicyOptions{
		ExpiryMinutes:   30,
		MaxFileSize:     16777216,
		ACL:             "private",
		RedirectBaseURL: "https://stellaris-galaxy-map.website.yandexcloud.net",
	})
	if err != nil {
		return nil, err
	}

	output := bytes.NewBuffer(nil)
	err = formText.Execute(output, post)
	if err != nil {
		return nil, err
	}

	return &Response{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type": "text/html",
		},
		Body: output.String(),
	}, nil
}
