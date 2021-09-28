package socket

import (
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
)

type Option func(*Server)

func Configuration(path string) Option {
	return func(s *Server) {
		cfg, err := conf.BuildWithOption(conf.LocalPath(path))
		if err != nil {
			log.StdPrintln(err)
		}
		s.Configuration = cfg
	}
}

func UseLogger() Option {
	logger := &log.Logger{}
	return func(s *Server) {
		logger.Apply(s.Configuration, "log")
		s.logger = logger
	}
}
