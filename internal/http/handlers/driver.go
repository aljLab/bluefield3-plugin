package handlers

import (
	"github.com/go-logr/logr"
	"github.com/mantra6g/bluefield3-plugin/pkg/p4client"
)

// Driver holds the P4Runtime client and gRPC connection.
type Driver struct {
	Client   *p4client.Bf3Client
	DeviceId uint64
	Log      logr.Logger
}
