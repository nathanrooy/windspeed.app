package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

    "windspeed/helpers/strava"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type StravaPost struct {
	AspectType       string    `json:"aspect_type"`
	EventTime        int64     `json:"event_time"`
	ObjectId         int64     `json:"object_id"`
	ObjectType       string    `json:"object_type"`
	OwnerId          int64     `json:"owner_id"`
	SubscriptionId   int64     `json:"subscription_id"`
	Updates struct {
	    Authorized   string    `json:"authorized"`
	}                          `json:"updates"`
}

func webhook(r events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	switch r.HTTPMethod {
	case "POST":
		return process_webhook_post(r)
	case "GET":
		return process_webhook_get(r)
    }
    return nil, nil
}

func process_webhook_post(r events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
    var stravaPost StravaPost
    err := json.Unmarshal([]byte(r.Body), &stravaPost)
    if err != nil {
        log.Printf("> json error: %v\n", err)
    }
    if stravaPost.AspectType == "create" && stravaPost.ObjectType == "activity" {
        defer strava.AddWeatherDetails(stravaPost.OwnerId, stravaPost.ObjectId)
    } else if stravaPost.Updates.Authorized == "false" {
        defer strava.DeleteUser(stravaPost.OwnerId)        
    }
    log.Printf("> returning 200 to strava")
    return &events.APIGatewayProxyResponse{
        StatusCode: 200,
        Body: "webhook ok",
    }, nil
}

func process_webhook_get(r events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
    log.Printf("verifying webhook subscription with strava\n")
    switch is_app_subscribed() {
    case true:
        log.Printf("windspeed.app is already subscribed\n")
        return &events.APIGatewayProxyResponse{
            StatusCode: 200,
            Body: "windspeed.app is already subscribed!",
        }, nil
    case false:
        log.Printf("performing hub challenge with strava...\n")
        hub_mode  := r.QueryStringParameters["hub.mode"]
        hub_token := r.QueryStringParameters["hub.verify_token"]
        if hub_mode == "subscribe" && hub_token == os.Getenv("STRAVA_VERIFY_TOKEN") {
            log.Printf("hub challenge passed.\n")
            return &events.APIGatewayProxyResponse{
                StatusCode: 200,
                Body: fmt.Sprintf("{ \"hub.challenge\":\"%s\" }", r.QueryStringParameters["hub.challenge"]),
                Headers: map[string]string{"Content-Type": "application/json"},
            }, nil
        } else {
            log.Printf("hub challenge failed, verification tokens do not match.\n")
            return &events.APIGatewayProxyResponse{
                StatusCode: 200,
                Body: "app is not subscribed...",
            }, nil
        }
    }
    log.Printf("finished\n")
    return nil, nil
}

func is_app_subscribed() bool {
	params := url.Values{}
	params.Add("client_id", os.Getenv("STRAVA_CLIENT_ID"))
	params.Add("client_secret", os.Getenv("STRAVA_CLIENT_SECRET"))
	resp, err := http.Get("https://www.strava.com/api/v3/push_subscriptions?" + params.Encode())
	if err != nil {
		log.Printf("request failed: %s\n", err)
	}
	defer resp.Body.Close()

	body_str, _ := io.ReadAll(resp.Body)
	body_map := []map[string]interface{}{}
	_ = json.Unmarshal(body_str, &body_map)
	if len(body_map) == 0 {
		return false
	} else {
		if body_map[0]["id"] != nil {
			return true
		} else {
			return false
		}
	}
}

func main() {
	lambda.Start(webhook)
}
