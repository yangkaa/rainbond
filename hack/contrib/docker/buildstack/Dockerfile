ARG GO_VERSION
FROM golang:${GO_VERSION}

RUN apk --no-cache add gcc libc-dev tzdata ca-certificates libc6-compat libgcc libstdc++ g++
    
RUN apk add --no-cache binutils-gold

WORKDIR /go

    


