#!/bin/bash

CGO_ENABLED=1 go run cmd/app/main.go

# CGO_ENABLED=1 go build -o tracker.exe cmd/app/main.go

# CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ go build -o tracker.exe cmd/app/main.go


# migrations
# create migration
# migrate create -ext sql -dir db/migrations -seq create_users_table