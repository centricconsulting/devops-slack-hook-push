package main

import (
    "net/http"
	"time"
)

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
