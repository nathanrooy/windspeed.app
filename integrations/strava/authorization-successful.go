package main

import (
	"bytes"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
    "fmt"
	"html/template"
    "io"
	"log"
	"net/http"
	"net/url"
	"os"
    "time"
	
	"windspeed/helpers/strava"

	"github.com/aws/aws-lambda-go/events"
  	"github.com/aws/aws-lambda-go/lambda"
    "github.com/gorilla/securecookie"
)

//go:embed *.html
var templates embed.FS

type StravaResponse struct {
	AccessToken		string 	`json:"access_token"`
	RefreshToken 	string 	`json:"refresh_token"`
	ExpiresAt		int64	`json:"expires_at"`
	Athlete struct {
		ID 			int64	`json:"id"`
		FirstName 	string	`json:"firstname"`
		LastName 	string	`json:"lastname"`
	} 						`json:"athlete"`
}

func getTokensFromCode(code string) StravaResponse {
    // get user tokens from strava
    params := url.Values{}
    params.Add("client_id", os.Getenv("STRAVA_CLIENT_ID"))
    params.Add("client_secret", os.Getenv("STRAVA_CLIENT_SECRET"))
    params.Add("code", code)
    params.Add("grant_type",  "authorization_code")
    resp, err := http.PostForm("https://www.strava.com/oauth/token", params)
    if err != nil {
        log.Printf("error getting user tokens from strava")
    }
    defer resp.Body.Close()

    // parse response from strava
    body, err := io.ReadAll(resp.Body)
    if err != nil {
      // handle error
    }
    var stravaResponse StravaResponse
    err = json.Unmarshal(body, &stravaResponse)
    if err != nil {
        // handle error
    }
    return stravaResponse
}

func authenticatedResponse(stravaResponse StravaResponse) (*events.APIGatewayProxyResponse, error) {
    
    // create csrf token
    csrf := make([]byte, 64)
    rand.Read(csrf)
    csrfToken := hex.EncodeToString(csrf)

    // prepare to render template
    tmpl := template.Must(template.ParseFS(templates, "*.html"))
    buf := new(bytes.Buffer)
    data := map[string]string{
        "csrfToken": csrfToken,
        "firstName": stravaResponse.Athlete.FirstName,
    }
    err := tmpl.Execute(buf, data)
    if err != nil {
        return nil, err
    }

    // create secure cookie
    var s = securecookie.New([]byte(os.Getenv("HASH_KEY")), []byte(os.Getenv("BLOCK_KEY")))
    v := map[string]string{
        "id": fmt.Sprintf("%v", stravaResponse.Athlete.ID),
        "exp": fmt.Sprintf("%v", time.Now().Add(5 * time.Minute).Unix()),
        "fn": stravaResponse.Athlete.FirstName,
        "csrf": csrfToken,
    }
    encoded, _ := s.Encode("windspeed", v);
    cookie := &http.Cookie{
        Name:  "windspeed",
        Value: encoded,
        SameSite: http.SameSiteStrictMode,
        Path: "/",
        Secure: true,
        HttpOnly: true,
    }
    
    // ask the user which units they prefer
    return &events.APIGatewayProxyResponse{
        StatusCode: 200,
        Body: buf.String(),
        Headers: map[string]string{
            "Set-Cookie": cookie.String(),
        },
    }, nil
}

func handler(r events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
    if r.HTTPMethod == "GET" {
        code := r.QueryStringParameters["code"]
        if code == "" {
            return &events.APIGatewayProxyResponse{
                StatusCode: 500,
                Body: "authorization code not found in strava response",
            }, nil
        } else {
            // get tokens from Strava
            stravaResponse := getTokensFromCode(code)
            
            // persist new user credentials
            strava.AddNewUser(stravaResponse.Athlete.ID, stravaResponse.AccessToken, stravaResponse.RefreshToken, stravaResponse.ExpiresAt)
        
            return authenticatedResponse(stravaResponse)
        }
    }
    return nil, nil
}

func main() {
	lambda.Start(handler)
}
