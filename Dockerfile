FROM golang
MAINTAINER William J Klos (bill.klos@centricconsulting.com)

RUN git clone https://github.com/centricconsulting/devops-slack-hook-push.git --branch master /go/src/github.com/centricconsulting/devops-slack-hook-push
RUN go get github.com/pborman/uuid
RUN go get github.com/go-martini/martini
RUN go get github.com/martini-contrib/binding
RUN cd /go/src/github.com/centricconsulting/slack-hook-push; go install

# Start the Spicoli server.
WORKDIR /go/src/github.com/centricconsulting/slack-hook-push
ENTRYPOINT ["slack-hook-push"]
