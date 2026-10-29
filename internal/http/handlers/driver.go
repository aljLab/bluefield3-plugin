package handlers

import (
	// "github.com/mantra6g/bluefield3-plugin/api"
	"github.com/mantra6g/bluefield3-plugin/pkg/p4client"

	"github.com/go-logr/logr"
	// "google.golang.org/grpc"
)

// Driver holds the P4Runtime client and gRPC connection.
type Driver struct {
	Client         *p4client.Bf3Client
	// Conn           *grpc.ClientConn
	// CurrentProgram *api.P4Program
	Log            logr.Logger
}