package goredis

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
)

type Handler func(ctx Context) Value
type ServerOption func(s *Server)

type Server struct {
	handlers map[string]Handler
}

func NewServer(opts ...ServerOption) *Server {
	server := &Server{
		handlers: map[string]Handler{
			"COMMAND": func(Context) Value {
				return StringValue("OK")
			},
		},
	}
	for _, opt := range opts {
		opt(server)
	}
	return server
}
func (s *Server) Command(command string, handler Handler) {
	s.handlers[strings.ToUpper(command)] = handler
}

func (s *Server) Start(address string) error {
	l, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("can't start listening on tcp: %w", err)
	}
	log.Printf("start listening on %s...\n", address)
	conn, err := l.Accept()
	if err != nil {
		return fmt.Errorf("can't accept connection: %w", err)
	}
	defer conn.Close()

	for {
		resp := NewResp(conn)
		val, err := resp.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return fmt.Errorf("connection closed")
			}
			return err
		}
		if val.typ != ARRAY {
			err := fmt.Errorf("wrong type of request body: expected array, got %s\n", string(val.typ))
			log.Println(err.Error())
			WriteValue(conn, ErrorValue(err))
			continue
		}
		if len(val.array) == 0 {
			err := fmt.Errorf("request's body array is empty\n")
			log.Println(err.Error())
			WriteValue(conn, ErrorValue(err))
			continue
		}
		command := strings.ToUpper(val.array[0].str)
		handler, ok := s.handlers[command]
		if !ok {
			err := fmt.Errorf("unexpected command: %s\n", command)
			log.Println(err.Error())
			WriteValue(conn, ErrorValue(err))
			continue
		}
		WriteValue(conn, handler(Context{
			command: val,
			Context: context.TODO(),
		}))
	}
}
