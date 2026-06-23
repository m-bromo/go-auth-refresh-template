package config

import (
	"errors"
	"fmt"
	"net"
	"strconv"

	"github.com/Netflix/go-env"
	"github.com/joho/godotenv"
)

const (
	MaxPort = 65535
)

func NewConfig(envPaths ...string) (*Config, error) {
	var config Config
	if err := godotenv.Load(envPaths...); err != nil {
		return nil, err
	}

	_, err := env.UnmarshalFromEnviron(&config)
	if err != nil {
		return nil, err
	}

	if config.IsDevelopment() {
		port, err := strconv.Atoi(config.API.Port)
		if err != nil {
			return nil, err
		}

		port, err = listenFromPort(config.API.Host, port)
		if err != nil {
			return nil, err
		}

		config.API.Port = strconv.Itoa(port)
	}

	return &config, nil
}

func listenFromPort(host string, startPort int) (int, error) {
	for port := startPort; port <= MaxPort; port++ {
		addr := net.JoinHostPort(host, strconv.Itoa(port))

		listener, err := net.Listen("tcp", addr)
		if err == nil {
			_ = listener.Close()
			return port, nil
		}

		var netErr *net.OpError
		if !errors.As(err, &netErr) {
			return 0, fmt.Errorf("checking port %d: %w", port, err)
		}
	}

	return 0, fmt.Errorf("no available port found from %d", startPort)

}
