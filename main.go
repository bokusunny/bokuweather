package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/dghubble/oauth1"
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

type apiResult struct {
	Type       string `json:"type"`
	StatusCode int    `json:"status_code"`
}

func getImageName() string {
	currentWeatherID := fetchCurrentWeatherID()

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
	localTime := time.Now().In(location)
	if h := localTime.Hour(); h <= 5 || 22 <= h {
		imageName = "bokumoon"
	}

	// イベントがある日は天気、時刻に関係なく上書き
	if _, m, d := localTime.Date(); m == 10 && d == 31 {
		imageName = "halloween_pumpkin"
	} else if m == 12 && d == 25 {
		imageName = "christmas_mark6_tonakai"
	} else if m == 2 && d == 3 {
		imageName = "setsubun_akaoni"
	}

	return imageName
}

func fetchCurrentWeatherID() int {
	log.Println("[INFO] Start fetching current weather info from weather api")

	city := "Tokyo"
	token := os.Getenv("WEATHER_API_TOKEN")
	apiURL := "https://api.openweathermap.org/data/2.5/weather?q=" + city + "&appid=" + token

	log.Printf("[INFO] Weather api token: %s", token)

	resp, _ := http.Get(apiURL)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		notifyAPIResultToSlack(false)
		log.Fatalf("[ERROR] Fail to read weather api response body: %s", err.Error())
	}
	defer resp.Body.Close()

	var respJSON weatherAPIResponse
	if err = json.Unmarshal(body, &respJSON); err != nil {
		notifyAPIResultToSlack(false)
		log.Fatalf("[ERROR] Fail to unmarshal weather api response json: %s", err.Error())
	}

	log.Printf("[INFO] Current weather: %s", respJSON.Weather[0].Main)
	return respJSON.Weather[0].ID
}

func fetchS3ImageObjByName(imageName string) *s3.GetObjectOutput {
	AWSSessionID := os.Getenv("AWS_SESSION_ID")
	AWSSecretAccessKey := os.Getenv("AWS_SECRET")

	log.Println("[INFO] Start fetching image obj from S3.")
	log.Printf("[INFO] AWS session id: %s", AWSSessionID)
	log.Printf("[INFO] AWS secret access key: %s", AWSSecretAccessKey)

	sess := session.Must(session.NewSession())
	creds := credentials.NewStaticCredentials(AWSSessionID, AWSSecretAccessKey, "")

	svc := s3.New(sess, &aws.Config{
		Region:      aws.String(endpoints.ApNortheast1RegionID),
		Credentials: creds,
	})

	obj, err := svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String("bokuweather"),
		Key:    aws.String(imageName + ".png"),
	})
	if err != nil {
		notifyAPIResultToSlack(false)
		log.Fatalf("[ERROR] Fail to get image object: %s", err.Error())
	}

	return obj
}

func updateSlackIcon(imgByte []byte, c chan apiResult) {
	imgBuffer := bytes.NewBuffer(imgByte)

	reqBody := &bytes.Buffer{}
	w := multipart.NewWriter(reqBody)
	part, err := w.CreateFormFile("image", "main.go")
	if _, err := io.Copy(part, imgBuffer); err != nil {
		notifyAPIResultToSlack(false)
		log.Fatalf("[Error] Fail to copy the image: %s", err.Error())
	}
	w.Close()

	req, _ := http.NewRequest(
		"POST",
		// "https://httpbin.org/post", // httpテスト用
		"https://slack.com/api/users.setPhoto",
		reqBody,
	)

	token := os.Getenv("SLACK_TOKEN")
	log.Printf("[INFO] Slack token: %s", token)

	req.Header.Set("Content-type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

	log.Println("[INFO] Send request to update slack icon!")
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		notifyAPIResultToSlack(false)
		log.Fatalf("[Error] Something went wrong with the slack setPhoto request : %s", err.Error())
	}
	log.Printf("[INFO] SetPhoto response status: %s", resp.Status)

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		notifyAPIResultToSlack(false)
		log.Fatalf("[Error] Fail to read the response: %s", err.Error())
	}

	var respJSON slackAPIResponse
	if err = json.Unmarshal(body, &respJSON); err != nil {
		notifyAPIResultToSlack(false)
		log.Fatalf("[Error] Fail to unmarshal the response: %s", err.Error())
	}

	if !respJSON.Ok {
		notifyAPIResultToSlack(false)
		log.Fatalf("[ERROR] Something went wrong with setPhoto request: %s", respJSON.Error)
	}

	c <- apiResult{"slack", resp.StatusCode}
}

func updateTwitterIcon(imgByte []byte, c chan apiResult) {
	oauthAPIKey := os.Getenv("OAUTH_CONSUMER_API_KEY")
	oauthAPIKeySecret := os.Getenv("OAUTH_CONSUMER_SECRET_KEY")
	oauthAccessToken := os.Getenv("OAUTH_ACCESS_TOKEN")
	oauthAccessTokenSecret := os.Getenv("OAUTH_ACCESS_TOKEN_SECRET")

	log.Printf("[INFO] Twitter API Key: %s", oauthAPIKey)
	log.Printf("[INFO] Twitter API Secret Key: %s", oauthAPIKeySecret)
	log.Printf("[INFO] Twitter Access Token: %s", oauthAccessToken)
	log.Printf("[INFO] Twitter Secret Access Token: %s", oauthAccessTokenSecret)

	config := oauth1.NewConfig(oauthAPIKey, oauthAPIKeySecret)
	token := oauth1.NewToken(oauthAccessToken, oauthAccessTokenSecret)

	httpClient := config.Client(oauth1.NoContext, token)

	encodedImg := base64.StdEncoding.EncodeToString(imgByte)
	encodedImg = url.QueryEscape(encodedImg) // replace URL encoding reserved characters
	log.Printf("Encoded icon: %s", encodedImg)

	twitterAPIRootURL := "https://api.twitter.com"
	twitterAPIMethod := "/1.1/account/update_profile_image.json"
	URLParams := "?image=" + encodedImg

	req, _ := http.NewRequest(
		"POST",
		twitterAPIRootURL+twitterAPIMethod+URLParams,
		nil,
	)

	log.Println("[INFO] Send request to update twitter icon!")
	resp, err := httpClient.Do(req)
	if err != nil {
		notifyAPIResultToSlack(false)
		log.Fatalf("[Error] Something went wrong with the twitter request : %s", err.Error())
	}
	defer resp.Body.Close()

	log.Printf("[INFO] Twitter updateImage response status: %s", resp.Status)

	c <- apiResult{"twitter", resp.StatusCode}
}

func notifyAPIResultToSlack(isSuccess bool) slackAPIResponse {
	channel := os.Getenv("SLACK_NOTIFY_CHANNEL_ID")
	attachmentsColor := "good"
	imageName := getImageName()
	attachmentsText := "Icon updated successfully according to the current weather! :" + imageName + ":"
	iconEmoji := ":bokurainy:"
	username := "bokuweather"

	if !isSuccess {
		lambdaCloudWatchURL := "https://ap-northeast-1.console.aws.amazon.com/cloudwatch/home?region=ap-northeast-1#logStream:group=/aws/lambda/bokuweather;streamFilter=typeLogStreamPrefix"
		attachmentsColor = "danger"
		attachmentsText = "Bokuweather has some problems and needs your help!:bokuthunder:\nWatch logs: " + lambdaCloudWatchURL
	}

	jsonStr := `{"channel":"` + channel + `","as_user":false,"attachments":[{"color":"` + attachmentsColor + `","text":"` + attachmentsText + `"}],"icon_emoji":"` + iconEmoji + `","username":"` + username + `"}`
	req, _ := http.NewRequest("POST", "https://slack.com/api/chat.postMessage", bytes.NewBuffer([]byte(jsonStr)))

	req.Header.Set("Authorization", "Bearer "+os.Getenv("SLACK_TOKEN"))
	req.Header.Set("Content-Type", "application/json")

	log.Printf("[INFO] Send request to notify outcome. isSuccess?: %t", isSuccess)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("[Error] Something went wrong with the postMessage reqest : %s", err.Error())
	}
	log.Printf("[INFO] PostMessage response states: %s", resp.Status)

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("[Error] Fail to read the response: %s", err.Error())
	}

	var respJSON slackAPIResponse
	if err = json.Unmarshal(body, &respJSON); err != nil {
		log.Fatalf("[Error] Fail to unmarshal the response: %s", err.Error())
	}

	return respJSON
}

func handler() (string, error) {
	imageName := getImageName()
	log.Println("[INFO] Successfully got imageName according to weather api.")
	log.Printf("[INFO] Icon imageName: %s", imageName)

	obj := fetchS3ImageObjByName(imageName)
	log.Println("[INFO] Successfully fetch the image object from s3.")

	imgByte, err := ioutil.ReadAll(obj.Body)
	if err != nil {
		notifyAPIResultToSlack(false)
		log.Fatalf("[Error] Fail to read the image object: %s", err.Error())
	}
	defer obj.Body.Close()

	c := make(chan apiResult, 1)

	go updateSlackIcon(imgByte, c)
	go updateTwitterIcon(imgByte, c)

	result1, result2 := <-c, <-c
	if result1.StatusCode != 200 || result2.StatusCode != 200 {
		notifyAPIResultToSlack(false)
		log.Fatalf("[ERROR] Something went wrong with updateImage func.")
	}

	postMessageJSON := notifyAPIResultToSlack(true)
	if !postMessageJSON.Ok {
		log.Fatalf("[ERROR] Something went wrong with postMessage request: %s", postMessageJSON.Error)
	}

	return "Successfully Updated!", nil
}

func main() {
	lambda.Start(handler)
}
