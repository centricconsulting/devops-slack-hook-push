// This utility redirects requests to Slack channels.
package main

import (
	"encoding/json"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/binding"
	"log"
	"net/http"
	"os"
	"time"
)

// Determine current operating environment.
var (
	m                                 *martini.Martini
	FlushTicker                       *time.Ticker
	InboundList                       chan SlackMessageIn
	OutboundList                      chan SlackMessageOut
	InboundNotifier, OutboundNotifier chan bool
	appConfig                         Config
	body                              []byte
	configFile                        string
	err                               error
	out                               []byte
	systemKey                         string
	apiv                              string
)

// Default icons for Slack posts.
const (
	ICON_ERROR   = "https://s3.amazonaws.com/centric-slack/cbot-error.png"
	ICON_INFO    = "https://s3.amazonaws.com/centric-slack/cbot-info.png"
	ICON_SUCCESS = "https://s3.amazonaws.com/centric-slack/cbot-success.png"
	ICON_WARN    = "https://s3.amazonaws.com/centric-slack/cbot-warning.png"
)

// This is what the user sends in.
type SlackMessageIn struct {
	Key            string `json: "key"`
	Action         string `json: "action"`
	Text           string `json: "text"`
	NotiftyOnError bool   `json: "notify_on_error"`
}

// This is what gets sent to Slack.
type SlackMessageOut struct {
	Hook    string       `json:"hook"`
	Payload SlackMessage `json: "payload"`
}

// Some application conifugration settings.
type Config struct {
	AcceptingNewSlackers bool     `json:"accepting_new_slackers"`
	AdminKey             string   `json:"admin_key"`
	Domains              []string `json:"domains"`
	TelemetriURL         string   `json:"telemetri_url"`
}

// init runs before everything else.
func init() {
	// Set the API version.
	apiv = "1.05"
	// Check credentials to make sure this is a legit request.
	slackerFile = "slackers.json"
	requestFile = "requests.json"

	configFile = "config.json"
	_, err := os.Stat(configFile)
	// If there is a problem with the file, err on the side of caution and
	// reject the request.
	if err != nil {
		log.Printf("error: Could not find configuration file/%s", configFile)
		os.Exit(1)
	}

	// These are the background processes we need to keep track of.
	InboundList = make(chan SlackMessageIn, 100)
	OutboundList = make(chan SlackMessageOut, 100)
	InboundNotifier = make(chan bool, 1)
	OutboundNotifier = make(chan bool, 1)
	FlushTicker = time.NewTicker(time.Minute * 1)

	// Set up the router.
	m = martini.New()
	// Setup Routes
	r := martini.NewRouter()
	r.Post(`/slack`, binding.Json(SlackMessageIn{}), PushToSlack)
	r.Post(`/slack/config`, binding.Json(SlackConfig{}), AddSlacker)
	r.Put(`/slack/config/:key_id`, binding.Json(SlackConfig{}), UpdateSlacker)
	r.Put(`/slack/config/:key_id/system`, AuthorizeAdmin, MakeSystemSlacker)
	r.Delete(`/slack/config/:key_id`, DeleteSlacker)
	r.Get(`/slack/configs`, GetSlackerCount)
	r.Get(`/slack/request/:email`, RequestSlackerId)
	r.Get(`/slack/requests`, GetRequestCount)
	r.Get(`/slack/ping`, PingTheApi)
	r.Get(`/slack/version`, GetSHPApiVersion)
	// Add the router action
	m.Action(r.Handle)
} // func

// Authorize is a middleware function that provides basic assurances that the
// request is legit.  It only returns on the negative path because the positive
// path is a passthrough to whatever action is after this in the route.
func AuthorizeAdmin(req *http.Request, rsp http.ResponseWriter) {
	// Make sure this is an admin request.
	if req.Header.Get("SPICOLI-ADMIN") != appConfig.AdminKey {
		rsp.WriteHeader(http.StatusUnauthorized)
	}
} // func

// 
func AuthorizeUsr(req *http.Request, rsp http.ResponseWriter) {
	// Make sure this is an admin request.
	if req.Header.Get("SPICOLI-USER") != appConfig.AdminKey {
		rsp.WriteHeader(http.StatusUnauthorized)
	}
} // func

// GetSHPApiVersion
func GetSHPApiVersion() (int, string) {
	return http.StatusOK, apiv
} // func

// PingTheApi
func PingTheApi() (int, string) {
	return http.StatusOK, "PONG"
} // func

// LoadConfig will read the configuration file and load the contents into a struct.
func LoadConfig() bool {
	// Check credentials to make sure this is a legit request.
	file, err := os.Open(configFile)
	// If there is a problem with the file, err on the side of caution and
	// reject the request.
	if err != nil {
		log.Printf("error: Unable to open file/%s", err.Error())
		return false
	}
	defer file.Close()

	// Decode the json into something we can process.  The JSON is set up to load
	// into a map.  We could also do an array and move it to a map, but why?
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&appConfig)
	if err != nil {
		log.Printf("error: Could not decode Config JSON/%s", err.Error())
		return false
	}
	log.Printf("info: Loaded config from disk.")
	// Everything was cool, but the supplied key simply doesn't match anything.
	return false
} // func

// ValidateDomain will tell the caller if the specified domain is within the list
// of domains allowed by the configuration file.
func ValidateDomain(domain string) bool {
	for _, item := range appConfig.Domains {
		if item == domain {
			return true
		}
	}
	return false
} // func

func main() {
	log.Printf("Starting Spicoli version %s ...", apiv)
	// Do an initial load of the JSON configuration files.
	LoadConfig()
	LoadSlackers()
	LoadRequests()

	// Set up a background process to load the configs periodically so that
	// new people can play and we can delete entries dynamicaclly.
	go func() {
		for {
			select {
			case <-GetInboundNotifier():
				DepleteInboundList()
			case <-GetOutboundNotifier():
				DepleteOutboundList()
			case <-GetFlushTicker():
				FlushSlackers()
				FlushRequests()
				LoadSlackers()
				LoadRequests()
			}
		}
	}()

	// Let's go!  You can change the listening port to whatever you want.
	m.RunOnAddr(":1966")
} // func
