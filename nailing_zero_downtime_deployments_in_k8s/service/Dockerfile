FROM golang:1.17


WORKDIR /go/src/app
COPY go.* ./
RUN go mod tidy

COPY ./service ./service

RUN go get -d -v ./...
RUN go build  -o app -v ./service/*.go
RUN chmod u+x app
RUN ls -la

CMD ["./app"]

# To build, run the following from the root folder
# sudo docker build -f service/Dockerfile .
# 
# Find it on dockerhub at jimmiebtlr/infra-debug-service:0.1.0