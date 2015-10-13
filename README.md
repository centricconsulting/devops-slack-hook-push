## Purpose
Spicoli is a lightweight server that listens for requests to be sent to Slack and passes them via pre-configured "slackers" to their destination.

## Getting Started

### Dependencies
To run the server, you will need to `go get` these libraries:

* `github.com/go-martini/martini`
* `github.com/martini-contrib/binding`
* `github.com/pborman/uuid`


Respecitvely, these libraries manage the service requests, translate JSON objects to structs, and generate the IDs for new slacker requests.

## Creating a Slack Request
The base unit on Spicoli is of course, a `slacker`. A slacker stores all the information you need to communicate with you slack instance, and allows you to customize things like the message, channel, and icon shown for each message.

In order to start sending messages to your Slack instance, you need to first setup a WebHook integration through Slack.  The main thing you'll need from that is the WebHook string.  Once you have that, you can start the Spicoli process.

### The Spicoli Process
The process is actually pretty simple:  

0. Edit the `config.json` File
1. Request a UUID
2. Create a Slacker
3. Submit Slack Messages

#### Edit the config.json File
The primary field of concern in the config file is the `domains` array.  This is currently the crux of security for Spicoli and is meant simply to keep random people from getting IDs and spamming your Slack instance.  When a user requests a Slacker Id, an email address must be provided.  The domain of that email address is compared against the domains you have specified in `config.json`.  If it's a match, then a UUID is generated, if not, then an error is returned.

#### Request a UUID
With your `config.json` file all ready to go, you can now allow your users to request a Slacker Id.  To do so is a simple REST GET:

    curl -X GET http://yourdomain.com:1966/slack/request/me@yourdomain.com

If the domain of the email you provided is in the `config.json` file, then you will get a UUID returned to you.  Otherwise, you will receive a `Invalid email domain.` if the domain does not match, or a `Invalid email address.` message if the email address cannot be parsed.

Make note of your UUID, as you will need it for the next step.

#### Create a Slacker
Once you have a UUID from the request service,

    curl -d '{"key":"7361c2a5-2ad6-4ca2-86c4-9349a0a61e1","hook":"https://hooks.slack.com/services/aaaa/bbbb/cccc","slack_data":{"username":"","icon_url":"","channel":""}}' -X POST http://yourdomain.com:1966/slack/config/7361c2a5-2ad6-4ca2-86c4-9349a0a61e1

In this example, if you do not specify any of the `slack_data`, then the defaults that you specified on the WebHook integration definition in your Slack account will be used.  NOTE: *The only required fields when creating a slacker are the __key__ and the __hook__.*

Also, you can specify multiple slackers for the same Slack WebHook.  This allows you to setup different slackers for different channels, bot personas, or applications you have.

#### Submit Slack Messages
Once you have the slacker created, you can now start to use it to send messages to your Slack instance.  There is only a minimal amount if information required to do this, as the goal is to make it easy for applications to share information on the platform.

If you are happy with the default settings on your slacker, you need only send the __key__ and the __text__ where the key corresponds to the slacker you just created, and the text to the message you want displayed on your slack channel.  If you specify an __action__ (info, success, warn, error), it should change the icon displayed assuming you do not have an override specified in your slacker definition.  

## Updating a Slacker
A slacker can also be updated.  All values you submit are the same as in the creation of the slacker and the new values will overwrite those that already exist (except the __key__).

    curl -d '{"key":"7361c2a5-2ad6-4ca2-86c4-9349a0a61e1","hook":"https://hooks.slack.com/services/aaaa/bbbb/cccc","slack_data":{"username":"mycoolbot","icon_url":"https://yourdomain.com/icon.png","channel":"mycoolchannel"}}' -X PUT http://yourdomain.com:1966/slack/config/7361c2a5-2ad6-4ca2-86c4-9349a0a61e1

A check will be made to make sure you still are provided the minimum amount of information, and that the key exists.  You do not have to get a new UUID to update an existing slacker.

## Configuration Files
These are the files used in running the server.  In an attempt to build a simple process, the goal was to use no database integration so all data is in the form of JSON formatted files that are read upon startup and updated every minute while the server is operational.  NOTE: *All of the configuration files should reside in the same directory as the binary.*

### config.json
The `config.json` file is the only file that is required to be present at startup.  The others will be created if necessary as part of the ticker that saves data every minute.

    {
        "accepting_new_slacks": true,
        "telemetri_url": "",
        "domains": ["abc.com","abc.cc"]
    }

The configuration is loaded along with the other data files every time the ticker is fired.  This allows modifications to the configuration without having to restart the server.

### requests.json
In order to add a new slacker to the system, you must first request an Id.  When you successfully request an Id, it is stored in the `requests.json` file with an expiration timestamp (of 2h after request time).  The system ticker that runs every minute will save any new requests to the file.  If there are any expired requests detected, the will be deleted.  As an example, the file looks like this:

    {
      "7361c2a5-2ad6-4ca2-86c4-9349a0a61e14": "2015-10-13T19:10:53.494716028Z",
      "5bab8f21-4f77-41ae-a0a6-7f1f26ee68d7": "2015-10-13T10:16:22.775736128Z"
    }

Requests do not have to be persisted in order to be used.  As soon as a valid request is created in the `map`, it can be turned into a `slacker` (see below).

### slackers.json

    {
      "0fde7b49-52e0-47a0-95b8-829850884a2f": {
        "key": "0fde7b49-52e0-47a0-95b8-829850884a2f",
        "use_telemetri": false,
        "message_template_id": "",
        "action": "info",
        "is_active": true,
        "hook": "https://hooks.slack.com/services/def567/abc123/1234",
        "is_system": false,
        "error_channel": "",
        "slack_data": {
          "username": "MyCoolBot",
          "icon_url": "https://abc.com/icon.png",
          "icon_emoji": "",
          "channel": "#mycoolchannel",
          "text": ""
        }
      }
    }

Refer to the Incoming WebHooks documentation on slack.com for more details on WebHook integration.

## TO-DO

* Add a system slacker that will notify a specific slack channel about various system events (e.g. filled Go channels, new requests, etc.)
* Honor the "is_active" flag.
* Honor the "accepting_new_slacks" flag.
* Add an email confirmation step to the *Request a UUID* process.
