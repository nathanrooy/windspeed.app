package main

import (
    "bytes"
    "embed"
    "html/template"
    "log"
    "net/http"
    "net/url"
    "os"
    "strconv"
	"strings"
	"time"

    "windspeed/helpers/strava"
    
	"github.com/aws/aws-lambda-go/events"
  	"github.com/aws/aws-lambda-go/lambda"
	"github.com/gorilla/securecookie"
)

//go:embed *.html
var templates embed.FS

func handler(r events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
    // Parse out user units selection and CSRF token.
    formValues, _ := url.ParseQuery(r.Body)
    units := formValues.Get("units")
    csrfForm := formValues.Get("csrf_token")

    // Get cookie and decode.
    var s = securecookie.New([]byte(os.Getenv("HASH_KEY")), []byte(os.Getenv("BLOCK_KEY")))
    cookie := make(map[string]string)
    for headerName, headerValue := range r.Headers {
        if strings.ToLower(headerName) == "cookie" {
            cookieParts := strings.SplitN(headerValue, "=", 2)
            if cookieParts[0] == "windspeed" {
                err := s.Decode("windspeed", cookieParts[1], &cookie)
                if err != nil {
                    log.Printf("> failed to decode secure cookie: %v\n", err)
                }
            }
        }
    }

    // Validate cookie contents.
    cookieExp, _ := strconv.ParseInt(cookie["exp"], 10, 64)
    var cookieIsValid bool = false
    if cookie["csrf"] == csrfForm &&  cookieExp > time.Now().Unix() && cookieExp > 0 {
        cookieIsValid = true
    }

    // Persist new user settings.
    athleteId, _ := strconv.ParseInt(cookie["id"],  10, 64)
    if cookieIsValid {
        switch units {
        case "imperial": 
            strava.AddUserSettings(athleteId, "imperial")
        case "metric":
            strava.AddUserSettings(athleteId, "metric")
        }

        // Prepare HTML templates for rendering.
		data := map[string]string{
			"firstName": cookie["fn"],
		}
        tmpl := template.Must(template.ParseFS(templates, "*.html"))
		buf := new(bytes.Buffer)
		err := tmpl.Execute(buf, data)
		if err != nil {
			return nil, err
		}

        // Nullify secure cookie session.
        v := map[string]string{
            "exp": "0",
            "csrf": "",
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
        
        // Send new users to final confirmation page.
        return &events.APIGatewayProxyResponse{
            StatusCode: 200,
            Body: buf.String(),
            Headers: map[string]string{
                "Set-Cookie": cookie.String(),
            },
        }, nil
    } else {
        return &events.APIGatewayProxyResponse{
            StatusCode: 200,
            Body: "expired session",
        }, nil
    }
    return nil, nil
}

func main() {
	lambda.Start(handler)
}
