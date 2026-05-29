package tcpopt

import (
	"fmt"
	"log"
	"net"

	"t2d/internal/config"
)

type Direction string

const (
	DirectionUpstream   Direction = "upstream"
	DirectionDownstream Direction = "downstream"
)

type fdConn interface {
	SetReadBuffer(int) error
	SetWriteBuffer(int) error
}

func ApplyTCPConn(conn *net.TCPConn) error {
	if conn == nil {
		return nil
	}
	if err := conn.SetReadBuffer(config.DefaultTCPBuffer); err != nil {
		return fmt.Errorf("set tcp read buffer: %w", err)
	}
	if err := conn.SetWriteBuffer(config.DefaultTCPBuffer); err != nil {
		return fmt.Errorf("set tcp write buffer: %w", err)
	}
	return nil
}

func ApplyGnetConn(conn fdConn) error {
	if conn == nil {
		return nil
	}
	if err := conn.SetReadBuffer(config.DefaultTCPBuffer); err != nil {
		return fmt.Errorf("set tcp read buffer: %w", err)
	}
	if err := conn.SetWriteBuffer(config.DefaultTCPBuffer); err != nil {
		return fmt.Errorf("set tcp write buffer: %w", err)
	}
	return nil
}

func WarnIfUnsupported(err error, context string) {
	if err != nil {
		log.Printf("[WARN] TCP option setup skipped for %s: %v", context, err)
	}
}
