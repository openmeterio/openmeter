package server

import (
	"github.com/openmeterio/openmeter/internal/server"
)

type Server = server.Server

type Config = server.Config

func NewServer(config *Config) (*Server, error) {
	return server.NewServer(config)
}
