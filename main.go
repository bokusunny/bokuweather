package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"time"

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

type weatherAPIResponse struct {
	Weather []struct {
		ID int `json:"id"`
	}
}

func setSlackPhotoHandler() (string, error) {
	rootURL := "https://slack.com/api/"
	apiMethod := "users.setPhoto"
	token := os.Getenv("SLACK_TOKEN")
	AWSSessionID := os.Getenv("AWS_SESSION_ID")
	AWSSecretAccessKey := os.Getenv("AWS_SECRET")
	S3Bucket := os.Getenv("BUCKET")
	S3BucketKey, err := getImageName()

	if err != nil {
		return err.Error(), nil
	}

	log.Printf("Slack token: %s", token)
	log.Printf("AWS session id: %s", AWSSessionID)
	log.Printf("AWS secret access key: %s", AWSSecretAccessKey)
	log.Printf("S3 bucket: %s", S3Bucket)
	log.Printf("S3 bucket key: %s", S3BucketKey)

	sess := session.Must(session.NewSession())
	creds := credentials.NewStaticCredentials(AWSSessionID, AWSSecretAccessKey, "")

	svc := s3.New(sess, &aws.Config{
		Region:      aws.String(endpoints.ApNortheast1RegionID),
		Credentials: creds,
	})

	obj, err := svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(S3Bucket),
		Key:    aws.String(S3BucketKey),
	})

	if err != nil {
		return "[Error] Fail to get object from s3: " + err.Error(), nil
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

func getImageName() (string, error) {
	currentWeatherID, err := fetchCurrentWeatherID()
	if err != nil {
		return "", err
	}

	var imageName string
	// ref: https://openweathermap.org/weather-conditions
	switch {
	case 200 <= currentWeatherID && currentWeatherID < 300:
		imageName = "bokuthunder.png"
	case currentWeatherID < 600:
		imageName = "bokurainy.png"
	case currentWeatherID < 700:
		imageName = "bokusnowy.png"
	default:
		imageName = "bokusunny.png"
	}

	// 夜は天気に関係なくbokumoon.pngに上書き
	if h := time.Now().Hour(); h <= 5 || 22 <= h {
		imageName = "bokumoon.png"
	}

	return imageName, nil
}

func fetchCurrentWeatherID() (int, error) {
	city := "Tokyo"
	token := os.Getenv("WEATHER_API_TOKEN")
	apiURL := "https://api.openweathermap.org/data/2.5/weather?q=" + city + "&appid=" + token

	resp, err := http.Get(apiURL)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	log.Println(string(body))
	if err != nil {
		return 0, err
	}

	var respJSON weatherAPIResponse
	if err = json.Unmarshal(body, &respJSON); err != nil {
		return 0, err
	}

	return respJSON.Weather[0].ID, nil
}

func main() {
	lambda.Start(setSlackPhotoHandler)
}
