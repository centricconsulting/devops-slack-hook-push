package main

import (
	"bytes"
	"encoding/json"
	"github.com/go-martini/martini"
	"github.com/pborman/uuid"
	"log"
	"net/http"
	"net/mail"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	requestFile  string
	request_keys map[string]time.Time
)

// This is what gets sent to Slack.
type Request struct {
	Key     string    `json:"key"`
	Expires time.Time `json: "expires"`
}

// DeleteRequest will remove the specified key from the Request map.
func DeleteRequest(id string) {
	delete(request_keys, id)
} // func

// FlushRequests will write all of the outstanding requests to disk.
func FlushRequests() {
	file, err := os.Create(requestFile)
	if err != nil {
		log.Printf("error: Unable to open file/%s", err.Error())
	}
	defer file.Close()

	// Let's make the JSON pretty.
	buf, err := json.MarshalIndent(request_keys, "", "  ")
	if err != nil {
		log.Printf("error: Unable to encode Requests JSON file/%s", err.Error())
	}

	// Now output the lot.
	out := bytes.NewBuffer(buf)
	_, err = out.WriteTo(file)
	if err != nil {
		log.Printf("error: Could not write to buffer/%s", err.Error())
	} else {
		log.Printf("info: Saved %d Requests to disk.", len(request_keys))
	}
} // func

// GetRequestCount returns the current number of slacker requests being served.
func GetRequestCount() (int, string) {
	return http.StatusOK, strconv.Itoa(len(request_keys))
} // func

// LoadRequests
func LoadRequests() bool {
	// Check credentials to make sure this is a legit request.
	file, err := os.Open(requestFile)
	// If there is a problem with the file, err on the side of caution and
	// reject the request.
	if err != nil {
		log.Printf("error: Unable to open file/%s", err.Error())
		return false
	}
	defer file.Close()

	// Allocate memory for the map.  We use this map to lookup configurations
	// when sending out Slack posts.
	request_keys = make(map[string]time.Time)

	// Decode the json into something we can process.  The JSON is set up to load
	// into a map.  We could also do an array and move it to a map, but why?
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&request_keys)
	if err != nil {
		log.Printf("error: Could not decode Requests JSON/%s", err.Error())
		return false
	}
	log.Printf("info: Loaded %d Requests from disk.", len(request_keys))

	// Check for expired key requests.
	for key, value := range request_keys {
		if time.Now().UTC().After(value) {
			log.Printf("info: Expiring %s", key)
			delete(request_keys, key)
		} // if
	} // for

	// Everything was cool, but the supplied key simply doesn't match anything.
	return false
} // func

// RequestSlackerId
func RequestSlackerId(params martini.Params) (int, string) {
	// We need an email, otherwise we will ignore the request.
	email, err := mail.ParseAddress(params["email"])
	if err != nil {
		return http.StatusBadRequest, "Invalid email address."
	}

	// Now make sure the address domain is one of the ones we are looking for.
	email_parts := strings.Split(email.Address, "@")
	if !ValidateDomain(email_parts[1]) {
		return http.StatusBadRequest, "Invalid email domain."
	}

	// Ok, give out the UUID.
	new_uuid := uuid.New()
	// Expire the UUID in a couple of hours so we don't have it hanging out there forever.
	// TODO: Put this in something like Redis where we can expire the invitation easily.
	request_keys[new_uuid] = time.Now().UTC().Add(time.Hour * 2)
	return http.StatusOK, new_uuid
} // func

// ValidateRequest
func ValidateRequest(id string) bool {
	if request_keys[id].IsZero() {
		return false
	}
	return true
} // func
