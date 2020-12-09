FROM golang:1.12.0-alpine3.9

RUN apk add git

# create /app WORKDIR and copy all project files to /app
RUN mkdir /app
ADD . /app
WORKDIR /app

# download deps
RUN go mod download

# build
RUN go build -o main .

# run main.go
CMD ["/app/main"]