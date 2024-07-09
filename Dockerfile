FROM golang:1.22-bullseye as gobuild
WORKDIR /go/src/go.datalift.io/demo-simple-app
COPY . .
RUN go build -ldflags="-s -w" -o ./build/server .

FROM gcr.io/distroless/base-debian12
COPY --from=gobuild /go/src/go.datalift.io/demo-simple-app/build/server /
CMD ["/server"]
