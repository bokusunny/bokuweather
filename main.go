package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type slackAPIResponse struct {
	Ok    bool   `json:"ok"`
	Error string `json:"error"`
}

func setSlackPhotoHandler() (string, error) {
	rootURL := "https://slack.com/api/"
	apiMethod := "users.setPhoto"
	token := os.Getenv("SLACK_TOKEN")
	AWSSessionID := os.Getenv("AWS_SESSION_ID")
	AWSSecretAccessKey := os.Getenv("AWS_SECRET")
	AWSBucket := os.Getenv("BUCKET")
	AWSBucketKey := os.Getenv("BUCKET_KEY")

	sess := session.Must(session.NewSession())
	creds := credentials.NewStaticCredentials(AWSSessionID, AWSSecretAccessKey, "")

	svc := s3.New(sess, &aws.Config{
		Region:      aws.String(endpoints.ApNortheast1RegionID),
		Credentials: creds,
	})

	obj, err := svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(AWSBucket),
		Key:    aws.String(AWSBucketKey),
	})

	if err != nil {
		return err.Error(), nil
	}

	defer obj.Body.Close()

	bodyByte, err := ioutil.ReadAll(obj.Body)
	bodyBuffer := bytes.NewBuffer(bodyByte)

	reqBody := &bytes.Buffer{}
	w := multipart.NewWriter(reqBody)
	part, err := w.CreateFormFile("image", "main.go")
	if _, err := io.Copy(part, bodyBuffer); err != nil {
		return "[Error] Fail to copy the file.", nil
	}
	w.Close()

	req, err := http.NewRequest(
		"POST",
		// "https://httpbin.org/post", // httpテスト用
		rootURL+apiMethod,
		reqBody,
	)

	if err != nil {
		return "[Error] Fail to generate new Reqest.", nil
	}

	req.Header.Set("Content-type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return "[Error] Request failed.", nil
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "[Error] Failed to read response.", nil
	}

	var respJSON slackAPIResponse
	if err = json.Unmarshal(body, &respJSON); err != nil {
		return "[Error] Failed to unmarshal response json.", nil
	}

	// log.Println(string(body)) // httpbinでのresp確認用

	if respJSON.Ok {
		// TODO: Slackに成功or失敗を通知する
		return "Successfully Updated", nil
	}
	return "Fail to Update: " + respJSON.Error, nil
}

func main() {
	lambda.Start(setSlackPhotoHandler)
}
