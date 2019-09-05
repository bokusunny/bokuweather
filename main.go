package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
)

type slackAPIRequest struct {
	Profile struct {
		DisplayName string `json:"display_name"`
	} `json:"profile"`
}

type slackAPIResponse struct {
	Ok      bool `json:"ok"`
	Profile struct {
		DisplayName string `json:"display_name"`
	} `json:"profile"`
	Error string `json:"error"`
}

func handler() (string, error) {
	rootURL := "https://slack.com/api/"
	apiMethod := "users.profile.set"
	token := os.Getenv("SLACK_TOKEN")

	reqBody := slackAPIRequest{}
	reqBody.Profile.DisplayName = "bokusunny"
	reqBodyByte, err := json.Marshal(reqBody)

	log.Printf("JSON: %s", string(reqBodyByte))

	if err != nil {
		log.Println("[Error] Failed to marshal response json.")
		return err.Error(), nil
	}

	req, err := http.NewRequest(
		"POST",
		rootURL+apiMethod,
		bytes.NewBuffer(reqBodyByte),
	)

	if err != nil {
		log.Println("[Error] Fail to generate new Reqest.")
		return err.Error(), nil
	}

	req.Header.Set("Content-type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		log.Println("[Error] Request failed.")
		return err.Error(), nil
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("[Error] Failed to read response.")
		return err.Error(), nil
	}

	var respJSON slackAPIResponse
	if err = json.Unmarshal(body, &respJSON); err != nil {
		log.Println("[Error] Failed to unmarshal response json.")
		return err.Error(), nil
	}

	if respJSON.Ok == false {
		return respJSON.Error, nil
	}

	userName := respJSON.Profile.DisplayName

	return "Successfully Updated: " + userName, nil
}

func main() {
	lambda.Start(handler)
}
