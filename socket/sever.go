package socket

import (
	"fmt"
	"github.com/quickfixgo/quickfix"
	"github.com/tsingsun/fixmore"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/pkg/log"
	"os"
	"os/signal"
	"syscall"
)

type Server struct {
	Configuration *conf.Configuration
	logger        *log.Logger
	acceptor      *quickfix.Acceptor
}

func New(opts ...Option) *Server {
	fs := &Server{}
	for _, opt := range opts {
		opt(fs)
	}
	return fs
}

func (s *Server) RegisterService(service *fixmore.FixService) (err error) {
	s.acceptor, err = quickfix.NewAcceptor(service, service.MessageStoreFactory, service.Settings, service.LogFactory)
	return err
}

func (s *Server) Start() error {
	err := s.acceptor.Start()
	if err != nil {
		return fmt.Errorf("Unable to start Acceptor: %s\n", err)
	}
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	<-interrupt
	s.Stop()
	return nil
}

func (s *Server) Stop() error {
	s.acceptor.Stop()
	s.logger.Sync()
	return nil
}
