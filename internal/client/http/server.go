package http

import (
	"runtime"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type (
	SerConfig struct {
		Port string
	}

	APIServer struct {
		l      *zap.Logger
		Server *fiber.App
		sem    chan struct{}
	}
)

func New(
	l *zap.Logger,
	conf SerConfig,
) (*APIServer, error) {
	app := fiber.New()
	ser := &APIServer{
		Server: app,
		l:      l,
		sem:    make(chan struct{}, runtime.NumCPU()),
	}

	return ser, nil
}
