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
		ID   int    `json:"id"`
		Main string `json:"main"`
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
		Key:    aws.String(S3BucketKey + ".png"),
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

	apiMethod = "chat.postMessage"
	channel := "DDZFYNW95"
	attatchmentsColor := "#2eb886"
	attatchmentsText := "Icon updated successfully according to the current weather! :" + S3BucketKey + ":"
	iconEmoji := ":bokurainy:"
	username := "bokuweather"

	if respJSON.Ok {
		jsonStr := `{"channel":"` + channel + `","as_user":false,"attachments":[{"color":"` + attatchmentsColor + `","text":"` + attatchmentsText + `"}],"icon_emoji":"` + iconEmoji + `","username":"` + username + `"}`
		log.Printf("Request2 JSON: %s", jsonStr)

		req2, err := http.NewRequest("POST", rootURL+apiMethod, bytes.NewBuffer([]byte(jsonStr)))
		if err != nil {
			return "[Error] Fail to generate new Reqest.", nil
		}

		req2.Header.Set("Authorization", "Bearer "+token)
		req2.Header.Set("Content-Type", "application/json")

		resp2, err := client.Do(req2)
		if err != nil {
			return "[Error] Request2 failed.", nil
		}
		log.Printf("Request 2 response states: %s", resp2.Status)

		defer resp2.Body.Close()
		body2, err := ioutil.ReadAll(resp2.Body)
		if err != nil {
			return "[Error] Failed to read response 2.", nil
		}

		var resp2JSON slackAPIResponse
		if err = json.Unmarshal(body2, &resp2JSON); err != nil {
			return "[Error] Failed to unmarshal response json.", nil
		}

		if resp2JSON.Ok {
			return "Successfully Updated", nil
		}
		return "Fail to send image update outcome to slack." + resp2JSON.Error, nil
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
		imageName = "bokuthunder"
	case currentWeatherID < 600:
		imageName = "bokurainy"
	case currentWeatherID < 700:
		imageName = "bokusnowy"
	default:
		imageName = "bokusunny"
	}

	// 夜は天気に関係なくbokumoonに上書き
	location, _ := time.LoadLocation("Asia/Tokyo")
	if h := time.Now().In(location).Hour(); h <= 5 || 22 <= h {
		imageName = "bokumoon"
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
	if err != nil {
		return 0, err
	}

	var respJSON weatherAPIResponse
	if err = json.Unmarshal(body, &respJSON); err != nil {
		return 0, err
	}

	log.Printf("Current weather: %s", respJSON.Weather[0].Main)
	return respJSON.Weather[0].ID, nil
}

func main() {
	lambda.Start(setSlackPhotoHandler)
}
