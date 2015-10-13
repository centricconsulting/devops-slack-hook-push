// This utility redirects requests to Slack channels.
package main

import (
	"bytes"
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
	AcceptingNewSlacks bool     `json:"accepting_new_slacks"`
	TelemetriURL       string   `json:"telemetri_url"`
	Domains            []string `json:"domains"`
}

// init runs before everything else.
func init() {
	// Set the API version.
	apiv = "1.00"
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
	r.Delete(`/slack/config/:key_id`, DeleteSlacker)
	r.Get(`/slack/configs`, GetSlackerCount)
	r.Get(`/slack/request/:email`, RequestSlackerId)
	r.Get(`/slack/requests`, GetRequestCount)
	r.Get(`/slack/ping`, PingTheApi)
	r.Get(`/slack/version`, GetSHPApiVersion)
	// Add the router action
	m.Action(r.Handle)
} // func

// DepleteInboundList will run through the all of the inbound Slack requests and process them for output.
// When done they are loaded on the output list.
func DepleteInboundList() {
	var sout SlackMessageOut

	z := len(InboundList)
	for i := 0; i < z; i++ {
		doc := <-InboundList
		// We have good parms, so let's make sure the key is good before doing any real work.
		scfg := GetSlacker(doc.Key)
		if scfg.Key != "" {
			// Load up the outbound message for Slack.
			sout.Payload.UserName = scfg.SlackData.UserName
			// We will use the Icon URL if it is specified.  If not, use the build it
			// based on the Action.
			switch doc.Action {
			case "info":
				sout.Payload.IconURL = ICON_INFO
			case "error":
				sout.Payload.IconURL = ICON_ERROR
			case "success":
				sout.Payload.IconURL = ICON_SUCCESS
			case "warn":
				sout.Payload.IconURL = ICON_WARN
			default:
				sout.Payload.IconURL = scfg.SlackData.IconURL
			} // switch

			sout.Hook = scfg.Hook
			sout.Payload.IconEmoji = scfg.SlackData.IconEmoji
			sout.Payload.Channel = scfg.SlackData.Channel
			// Now load up the text.
			sout.Payload.Text = doc.Text

			// OK, prep for sending out to Slack.  If we can't, send an error
			// to the error channel.
			if !FillOutboundList(sout) {
				log.Printf("error: Outbound list is full")
				// TODO: Send out to error channel.
			}
			log.Printf("%s queued to outbound", doc.Key)
		} else {
			log.Printf("error: Could not find key")
			// TODO: Send an error to the error channel.
		} // else
	} // for
} // func

// DepleteOutboundList will take everything queued from the inbound side and send
// them out.
func DepleteOutboundList() {
	z := len(OutboundList)
	for i := 0; i < z; i++ {
		doc := <-OutboundList
		// Convert to HTTP-needs so we can send the message out.
		body, err = json.Marshal(doc.Payload)
		if err != nil {
			log.Printf("error: Could not marshal payload/%s", err.Error())
			//return http.StatusBadRequest, "JSON Error"
		}
		req, err := http.NewRequest("POST", doc.Hook, bytes.NewBuffer(body))

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Error when connecting to Slack: %s", err.Error())
			//return http.StatusBadRequest, "Could not connect to Slack."
		}
		defer resp.Body.Close()
		log.Printf("sent to channel %s", doc.Payload.Channel)
	} // for
} // func

// GetSHPApiVersion
func GetSHPApiVersion() (int, string) {
	return http.StatusOK, apiv
} // func

// PingTheApi
func PingTheApi() (int, string) {
	return http.StatusOK, "PONG"
} // func

// LoadConfig
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

	// Allocate memory for the map.  We use this map to lookup configurations
	// when sending out Slack posts.
	slackers = make(map[string]SlackConfig)

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
