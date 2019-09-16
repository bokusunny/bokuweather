package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"

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

// func handler() (string, error) {
// rootURL := "https://slack.com/api/"
// apiMethod := "users.setPhoto"
// token := os.Getenv("SLACK_TOKEN")

// file, err := os.Open("test.jpeg")
// if err != nil {
// 	log.Fatal("[Error] Fail to open the image.")
// }
// defer file.Close()

// reqBody := &bytes.Buffer{}
// w := multipart.NewWriter(reqBody)
// part, err := w.CreateFormFile("image", file.Name())
// log.Println(file.Name())
// if _, err := io.Copy(part, file); err != nil {
// 	log.Fatal("[Error] Fail to copy the file.")
// }
// w.Close()

// req, err := http.NewRequest(
// 	"POST",
// 	rootURL+apiMethod,
// 	reqBody,
// )

// if err != nil {
// 	log.Println("[Error] Fail to generate new Reqest.")
// 	return err.Error(), nil
// }

// req.Header.Set("Content-type", "multipart/form-data")
// req.Header.Set("Authorization", "Bearer "+token)

// client := &http.Client{}
// resp, err := client.Do(req)

// if err != nil {
// 	log.Println("[Error] Request failed.")
// 	return err.Error(), nil
// }

// defer resp.Body.Close()
// body, err := ioutil.ReadAll(resp.Body)
// if err != nil {
// 	log.Println("[Error] Failed to read response.")
// 	return err.Error(), nil
// }

// var respJSON slackAPIResponse
// if err = json.Unmarshal(body, &respJSON); err != nil {
// 	log.Println("[Error] Failed to unmarshal response json.")
// 	return err.Error(), nil
// }

// if respJSON.Ok {
// 	// TODO: Slackに成功or失敗を通知する
// 	return "Successfully Updated", nil
// }

// return respJSON.Error, nil
// }

func main() {
	// lambda.Start(handler)
	rootURL := "https://slack.com/api/"
	apiMethod := "users.setPhoto"
	token := os.Getenv("SLACK_TOKEN")
	AWSBucket := os.Getenv("BUCKET")
	AWSKey := os.Getenv("KEY")

	sess := session.Must(session.NewSession())
	// creds := stscreds.NewCredentials(sess, "arn:aws:iam::786206931755:user/sunny")
	creds := credentials.NewStaticCredentials(
		// TODO: 環境変数に
		"AKIA3ODMVXMVSMCC7ZPV",
		"u/Ocz2ehVk1Ia34ptCwp6ry0j1KGfkFOxOTa/dq/",
		"",
	)

	svc := s3.New(sess, &aws.Config{
		Region:      aws.String(endpoints.ApNortheast1RegionID),
		Credentials: creds,
	})

	obj, err := svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(AWSBucket),
		Key:    aws.String(AWSKey),
	})

	if err != nil {
		log.Fatal(err.Error())
	}

	defer obj.Body.Close()
	log.Println("Object Size:", aws.Int64Value(obj.ContentLength))

	bodyByte, err := ioutil.ReadAll(obj.Body)
	reqBody := bytes.NewBuffer(bodyByte)

	req, err := http.NewRequest(
		"POST",
		rootURL+apiMethod,
		reqBody,
	)

	if err != nil {
		log.Fatal("[Error] Fail to generate new Reqest.")
	}

	req.Header.Set("Content-type", "multipart/form-data")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		log.Fatal("[Error] Request failed.")
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("[Error] Failed to read response.")
	}

	var respJSON slackAPIResponse
	if err = json.Unmarshal(body, &respJSON); err != nil {
		log.Fatal("[Error] Failed to unmarshal response json.")
	}

	if respJSON.Ok {
		// TODO: Slackに成功or失敗を通知する
		log.Fatal("Successfully Updated")
	}
	log.Fatal("Fail to Update: " + respJSON.Error)
}
