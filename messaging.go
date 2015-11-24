package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

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
				// TODO: Send out to system channel.
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
			log.Printf("error: Could not connect to Slack/%s", err.Error())
		}
		defer resp.Body.Close()
		log.Printf("sent to channel %s", doc.Payload.Channel)
	} // for
} // func


// Functions for reading and pushing notifications for the inbound Slack requests.
func GetInboundNotifier() chan bool {
	return InboundNotifier
}
func NotifyInboundList() {
	// Since this is only a "call to action" channel, it only needs one call.
	// If there is already a message in it, then someone else made that call.
	if len(InboundNotifier) < cap(InboundNotifier) {
		InboundNotifier <- true
	}
}
func FillInboundList(smi SlackMessageIn) bool {
	// Make sure there is room in the list before adding any thing to it.
	if len(InboundList) < cap(InboundList) {
		InboundList <- smi
		NotifyInboundList()
		return true
	}
	return false
}

// Functions for reading and pushing notifications for the outbound Slack posts.
func GetOutboundNotifier() chan bool {
	return OutboundNotifier
}
func NotifyOutboundList() {
	// Since this is only a "call to action" channel, it only needs one call.
	// If there is already a message in it, then someone else made that call.
	if len(OutboundNotifier) < cap(OutboundNotifier) {
		OutboundNotifier <- true
	}
}
func FillOutboundList(smo SlackMessageOut) bool {
	// Make sure there is room in the list before adding any thing to it.
	if len(OutboundList) < cap(OutboundList) {
		OutboundList <- smo
		NotifyOutboundList()
		return true
	}
	return false
}

// Ticker for flushing and reloading the config file.
func GetFlushTicker() <-chan time.Time {
	return FlushTicker.C
}

//
func PushToSlack(smi SlackMessageIn) (int, string) {
	// Make sure we have a good set of parameters before we go anywhere.
	if smi.Key == "" {
		return http.StatusBadRequest, "Key not provided.  Have you registered?"
	}

	// We need text.  Otherwise, what's the point?
	if smi.Text == "" {
		return http.StatusBadRequest, "Slack text not provided.  What do you want me to say?"
	}

	// The basics look good, throw it on the list to be processed in the background.
	if FillInboundList(smi) {
		// We've accepted the message.  There's another process for notifying the user of issues.
		return http.StatusAccepted, "Accepted"
	}
	return http.StatusBadRequest, "Inbound list is full."
} // func
