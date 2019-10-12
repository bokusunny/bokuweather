# bokuweather
<img src="https://img.shields.io/badge/go-v1.12-blue.svg"/> [![Go Report Card](https://goreportcard.com/badge/github.com/bokusunny/bokuweather)](https://goreportcard.com/report/github.com/bokusunny/bokuweather)

**Update slack/twitter icon according to the current weather☀️**
<img width="710" alt="スクリーンショット 2019-10-05 10 23 22" src="https://user-images.githubusercontent.com/39128496/66697353-c8c29b00-ed0f-11e9-9c22-4506495e5aa4.png">

## How this app works
※ You can check the detail on [Qiita](https://qiita.com/bokusunny/items/af73ed04c304e9efeba2) 

1. Fetch current weather information from [OpenWeatherMap API](https://openweathermap.org/current)
2. Get the image(Uploaded to S3 beforehand) from S3 according to the weather info
3. Send update-icon request with the image to [slack API](https://api.slack.com/methods/users.setPhoto)/[Twitter API](https://developer.twitter.com/en/docs/accounts-and-users/manage-account-settings/api-reference/post-account-update_profile_image) asynchronously
4. Post the results of the requests to my slack DM
5. Build and zip this script, upload to AWS Lambda and execute regularly

## Requirement
- Go
- AWS(Lambda, S3, IAM)
- AWS SDK forGO
- Slack API
- Twitter API
