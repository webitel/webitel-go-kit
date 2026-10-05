package main

import (
	"io"

	gokitlog "github.com/webitel/webitel-go-kit/pkg/depenlog"
	"google.golang.org/grpc/grpclog"
)

func main() {
	gokitlog.New(gokitlog.Config{Level: "debug", JSON: true, Console: true})

	grpclog.Info("parsed scheme: ", "dns")

	grpclog.Warningf("transport: authentication handshake failed: %v", io.EOF)

	if grpclog.V(1) {
		grpclog.Info("verbose framework detail")
	}
}
