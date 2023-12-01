package main

import (
	"html/template"
	"net/url"
	"os"
)

func main() {
	render_landing_page()
}

func render_landing_page() {
	tmpl, _ := template.ParseFiles("templates/index.html", "templates/header.html")
	html, _ := os.Create("public/index.html")
	_ = tmpl.Execute(html, map[string]string{"StravaAuthorizationLink": make_strava_link_to_get_code()})
	_ = html.Close()
}

func make_strava_link_to_get_code() string {
	params := url.Values{}
	params.Add("response_type", "code")
	params.Add("client_id", os.Getenv("STRAVA_CLIENT_ID"))
	params.Add("scope", "read,activity:write,activity:read_all")
	params.Add("approval_prompt", "auto")
	params.Add("redirect_uri", "https://windspeed.app/integrations/strava/authorization-successful")
	return "https://www.strava.com/oauth/authorize?" + params.Encode()
}
