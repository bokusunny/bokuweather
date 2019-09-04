package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
)

type response struct {
	Ok      bool `json:"ok"`
	Profile struct {
		DisplayName string `json:"display_name"`
	} `json:"profile"`
}

func hello() (string, error) {
	rootURL := "https://slack.com/api/"
	apiMethod := "users.profile.get"
	token := os.Getenv("SLACK_TOKEN")

	resp, err := http.Get(rootURL + apiMethod + "?token=" + token)

	if err != nil {
		return err.Error(), nil
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err.Error(), nil
	}

	var respJSON response
	if err = json.Unmarshal(body, &respJSON); err != nil {
		return err.Error(), nil
	}

	userName := respJSON.Profile.DisplayName

	return userName, nil
}

func main() {
	lambda.Start(hello)
}
