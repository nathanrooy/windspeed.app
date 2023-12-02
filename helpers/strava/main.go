package strava

import (
    "bytes"
    "database/sql"
    "encoding/json"
	"fmt"
    "io"
	"log"
    "net/http"
    "net/url"
	"os"
    "strings"
    "time"
    
	"windspeed/utils/database"
    "windspeed/utils/weather"
)

const DB_SCHEMA string = "strava"

type Activity struct {
    Description     string       `json:"description"`
    Id              int64        `json:"id"`
    Manual          bool         `json:"manual" default:"false"`
    StartDate       string       `json:"start_date"`
    StartLatLng     [2]float64   `json:"start_latlng"`
    Trainer         bool         `json:"trainer" default:"true"`
    Type            string       `json:"type"`
}

type Tokens struct {
    AccessToken    string    `json:"access_token"`
    AthleteId      int64
	ExpiresAt      int64     `json:"expires_at"`
    RefreshToken   string    `json:"refresh_token"`
}

func AddNewUser(athleteId int64, accessToken string, refreshToken string, expiresAt int64) {
	log.Printf("adding new strava user: %v\n", athleteId)
	db := database.Connect()
	defer db.Close()

    tokens := Tokens{
        AccessToken:  accessToken,
        AthleteId:    athleteId,
        ExpiresAt:    expiresAt,
        RefreshToken: refreshToken,
    }

    updateUserTokens(db, tokens)
    database.AddEvent(fmt.Sprintf("%d", athleteId), "strava", database.NewUser, db)
}

func updateUserTokens(db *sql.DB, tokens Tokens) {
    sql := `INSERT INTO %s.%s.subscribers (id, access_token, refresh_token, expires_at) VALUES($1, $2, $3, $4) ON CONFLICT (id) DO UPDATE SET access_token = $2, refresh_token = $3, expires_at = $4;`
	stmt, err := db.Prepare(fmt.Sprintf(sql, os.Getenv("DB_DATABASE"), DB_SCHEMA))
	if err != nil {
		log.Printf("> failed to prepare insert: %v", err)
	}
	result, err := stmt.Exec(tokens.AthleteId, tokens.AccessToken, tokens.RefreshToken, tokens.ExpiresAt)
	if err != nil {
		log.Printf("> db-err: %s\n", err)
	}
    rowsAffected, err := result.RowsAffected()
	if rowsAffected == 1 {
		log.Printf("> successfully updated tokens for strava user \"%v\"\n", tokens.AthleteId)
	} else {
		log.Printf("> failed to update tokens for strava user \"%v\"\n", tokens.AthleteId)
	}
}

func AddUserSettings(athleteId int64, units string) {
    log.Printf("adding user settings\n")
    db := database.Connect()
	defer db.Close()

    sql := `INSERT INTO %s.%s.settings (id, units) VALUES($1, $2) ON CONFLICT (id) DO UPDATE SET units = $2;`
	stmt, _ := db.Prepare(fmt.Sprintf(sql, os.Getenv("DB_DATABASE"), DB_SCHEMA))
    
    result, err := stmt.Exec(athleteId, units)
	if err != nil {
		log.Printf("db-err: %s\n", err)
	}
    rowsAffected, err := result.RowsAffected()
	if rowsAffected == 1 {
		log.Printf("successfully added settings for user \"%v\" to \"%v.settings\"\n", athleteId, DB_SCHEMA)
	} else {
		log.Printf("failed to add user settings for \"%v\" to \"%v.subscribers\"\n", athleteId, DB_SCHEMA)
	}
}

func DeleteUser(athleteId int64) {
    log.Printf("> remove strava user: %v\n", athleteId)
	db := database.Connect()
	defer db.Close()
    deleteSubscriber(db, athleteId)
    deleteSettings(db, athleteId)
    database.AddEvent(fmt.Sprintf("%d", athleteId), "strava", database.DeleteUser, db)
}

func deleteSubscriber(db *sql.DB, athleteId int64) {
    sql := `DELETE FROM %s.%s.subscribers WHERE id = $1`
	stmt, err := db.Prepare(fmt.Sprintf(sql, os.Getenv("DB_DATABASE"), DB_SCHEMA))
	if err != nil {
		log.Printf("> failed to prepare insert: %v", err)
	}
	result, err := stmt.Exec(athleteId)
	if err != nil {
		log.Printf("db-err: %s\n", err)
	}
    rowsAffected, err := result.RowsAffected()
	if rowsAffected == 1 {
		log.Printf("successfully removed user \"%v\" from \"%v.subscribers\"\n", athleteId, DB_SCHEMA)
	} else {
		log.Printf("failed to remove user \"%v\" from \"%v.subscribers\"\n", athleteId, DB_SCHEMA)
	}
}

func deleteSettings(db *sql.DB, athleteId int64) {
    sql := `DELETE FROM %s.%s.settings WHERE id = $1`
	stmt, err := db.Prepare(fmt.Sprintf(sql, os.Getenv("DB_DATABASE"), DB_SCHEMA))
	if err != nil {
		log.Printf("> failed to prepare insert: %v", err)
	}
	result, err := stmt.Exec(athleteId)
	if err != nil {
		log.Printf("db-err: %s\n", err)
	}
    rowsAffected, err := result.RowsAffected()
	if rowsAffected == 1 {
		log.Printf("successfully removed user \"%v\" from \"%v.settings\"\n", athleteId, DB_SCHEMA)
	} else {
		log.Printf("failed to remove user \"%v\" from \"%v.settings\"\n", athleteId, DB_SCHEMA)
	}
}

func getUserUnits(db *sql.DB, athleteId int64) string {
    sql := `SELECT units FROM %s.%s.settings WHERE id = $1`
	stmt, _ := db.Prepare(fmt.Sprintf(sql, os.Getenv("DB_DATABASE"), DB_SCHEMA))
    
    var units string
    if err := stmt.QueryRow(athleteId).Scan(&units); err != nil {
        log.Printf("> error getting strava user units: %v", err)
        units = "imperial"
    }
    return units
}

func AddWeatherDetails(athleteId int64, activityId int64) {
    log.Printf("> adding weather details for strava user: %v, activity = %v\n", athleteId, activityId)
    tStartMs := time.Now().UnixMilli()

	db := database.Connect()
	defer db.Close()

    oldTokens := getUserTokens(db, athleteId)
    tokens := refreshUserTokens(oldTokens)
    
    // persist latest user tokens (if necessary)
    if tokens.ExpiresAt > oldTokens.ExpiresAt {
        log.Printf("> persisting updated tokens\n")
        updateUserTokens(db, tokens)
    }

    activity := getActivity(activityId, tokens)

    // only add weather details for some activities that don't already have one...
    if activity.Manual == true || activity.Trainer == true || activity.Type == "VirtualRide" {
        log.Printf("Activity %v was manually created or indoor. Skipping...\n", activityId)
    } else if strings.Contains(activity.Description, "°C") || strings.Contains(activity.Description, "°F") {
        log.Printf("Activity %v already has weather information\n", activityId)
    } else if activity.StartLatLng == [2]float64{} {
        log.Printf("No position present for activity %v\n", activityId) 
    } else if activity.StartLatLng != [2]float64{} {
        // retreive users prefered units
        userUnits := getUserUnits(db, athleteId)

        // parse activity start time into a time object
        t, err := time.Parse(time.RFC3339, activity.StartDate)
        if err != nil {
            log.Printf("Failed to parse activity start time...\n")
        }

        // Generate weather stamp
        weatherStamp := weather.CreateStamp(activity.StartLatLng[0], activity.StartLatLng[1], t.Unix(), userUnits)
        log.Printf("weather stamp: \"%v\"\n", weatherStamp)
        
        // Add weather stamp to existing activity description
        if activity.Description != "" {
            activity.Description = strings.TrimRight(activity.Description, " ") + "\n" + weatherStamp
        } else {
            activity.Description = weatherStamp
        }

        // Update Strava activity
        modifyActivity(activity, tokens)
        tDelta := time.Now().UnixMilli() - tStartMs

        // Record event
        event := map[string]interface{}{
            "event_type": activity.Type,
            "event_time":  t.Unix(),
            "lat": activity.StartLatLng[0],
            "lng": activity.StartLatLng[1],
            "weather_stamp": weatherStamp,
            "duration": tDelta,
        }
        jsonBytes, _ := json.Marshal(event)
        jsonString := string(jsonBytes)
        log.Printf("> %v\n", jsonString)
        database.AddEvent(fmt.Sprintf("%d", athleteId), "strava", jsonString, db)

        log.Printf("> finished in: %v ms\n", time.Now().UnixMilli() - tStartMs)
    }
}

func modifyActivity(activity Activity, tokens Tokens) {
    log.Printf("> modifying activity: %v for user: %v\n", activity.Id, tokens.AthleteId)

    // create payload
    payload := map[string]string{
        "description": activity.Description,
    }
    
    // set the payload
    jsonPayload, err := json.Marshal(payload)
    if err != nil {
        panic(err)
    }

     // Create a new HTTP request
    url := fmt.Sprintf("https://www.strava.com/api/v3/activities/%v", activity.Id)
    req, err := http.NewRequest("PUT", url, bytes.NewReader(jsonPayload))
    if err != nil {
        panic(err)
    }

    // Set the Authorization header
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", tokens.AccessToken))
    req.Header.Set("Content-Type", "application/json")
    
    // Send the request
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        log.Printf("> error sending put request to strava: %v", err)
    }

    // Check the response status code.
    log.Printf("> response code: %v\n", resp.StatusCode)
    if resp.StatusCode != http.StatusOK {
        log.Printf("unexpected response status code: %d", resp.StatusCode)
    }
}
    
func getActivity(activityId int64, tokens Tokens) Activity {

    log.Print("getting user activity\n")
    
     // Create a new HTTP request
    req, err := http.NewRequest("GET", fmt.Sprintf("https://www.strava.com/api/v3/activities/%v", activityId), nil)
    if err != nil {
        panic(err)
    }

    // Set the Authorization header
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", tokens.AccessToken))

    // Send the request
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    // Read the response body.
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        panic(err)
    }

    // Munge response body.
    var activity Activity
    _ = json.Unmarshal(body, &activity)
    return activity
}

func getUserTokens(db *sql.DB, athleteId int64) Tokens {
    log.Printf("> getting user tokens\n")
    sql := `SELECT id, access_token, refresh_token, expires_at FROM %s.%s.subscribers WHERE id = $1 LIMIT 1`
    stmt, _ := db.Prepare(fmt.Sprintf(sql, os.Getenv("DB_DATABASE"), DB_SCHEMA))
    
    var tokens Tokens
    if err := stmt.QueryRow(athleteId).Scan(&tokens.AthleteId, &tokens.AccessToken, &tokens.RefreshToken, &tokens.ExpiresAt); err != nil {
        log.Printf("> error getting user tokens: %v\n", err)
    }
    return tokens
}

func refreshUserTokens(tokens Tokens) Tokens {

    if tokens.ExpiresAt > time.Now().Unix() {
        log.Printf("> current user tokens are still valid\n")
        return tokens
    } else {
        log.Printf("> tokens have expired. contacting strava for latest user tokens\n")
        
        // refresh tokens
		params := url.Values{}
		params.Add("client_id", os.Getenv("STRAVA_CLIENT_ID"))
		params.Add("client_secret", os.Getenv("STRAVA_CLIENT_SECRET"))
		params.Add("refresh_token", tokens.RefreshToken)
		params.Add("grant_type",  "refresh_token")
		resp, err := http.PostForm("https://www.strava.com/oauth/token", params)
		if err != nil {
			log.Printf("error refreshing user tokens from strava")
		}
		defer resp.Body.Close()

		// parse response from strava
        body, err := io.ReadAll(resp.Body)
        if err != nil {
          // handle error
        }
		var refreshedTokens Tokens
		err = json.Unmarshal(body, &refreshedTokens)
		if err != nil {
            log.Printf("> token unmarshal error: %v", err)
		}
        refreshedTokens.AthleteId = tokens.AthleteId
        if refreshedTokens.ExpiresAt > tokens.ExpiresAt {
            log.Printf("> tokens have been refreshed\n")
        }
        return refreshedTokens
    }
}
