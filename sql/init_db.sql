-- schemas
CREATE SCHEMA IF NOT EXISTS events
CREATE SCHEMA IF NOT EXISTS strava

-- subscribers    
CREATE TABLE IF NOT EXISTS strava.subscribers (
    id            integer    NOT NULL PRIMARY KEY,
    access_token  text       NOT NULL,
    refresh_token text       NOT NULL,
    expires_at    integer    NOT NULL);

-- settings
CREATE TABLE IF NOT EXISTS strava.settings (
    id            integer    NOT NULL PRIMARY KEY,
    units         text       NOT NULL);

-- events
CREATE TABLE IF NOT EXISTS events.events (
    event_time 	  int8      NOT NULL,
    anonymous_id  text      NOT NULL,
    service       text      NOT NULL,
    event         text      NOT NULL);
