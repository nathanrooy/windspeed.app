go run scripts/build.go

mkdir functions

mkdir functions/api_strava_webhook
cp -rf integrations/strava/webhook.go functions/api_strava_webhook/main.go

mkdir functions/integrations_strava_authorization-successful
cp -rf integrations/strava/authorization-successful.go functions/integrations_strava_authorization-successful/main.go
cp -rf templates/authorized.html functions/integrations_strava_authorization-successful/authorized.html
cp -rf templates/header.html functions/integrations_strava_authorization-successful/header.html

mkdir functions/integrations_strava_final
cp -rf integrations/strava/final.go functions/integrations_strava_final/main.go
cp -rf templates/final.html functions/integrations_strava_final/final.html
cp -rf templates/header.html functions/integrations_strava_final/header.html
