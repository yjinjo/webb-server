package webb_server

import (
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type GRPCServer struct {
	port           int
	maxConnections int
	grpcServer     *grpc.Server
}

func NewGRPCServer(port int, maxConnections int) *GRPCServer {
	grpcServer := grpc.NewServer()
	reflection.Register(grpcServer)

	return &GRPCServer{
		port:           port,
		maxConnections: maxConnections,
		grpcServer:     grpcServer,
	}
}

func (s *GRPCServer) Run() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	log.Printf("Start gRPC Server: port=%d, max-connections=%d", s.port, s.maxConnections)
	if err := s.grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %v", err)
	}

	return nil
}

func (s *GRPCServer) GetGRPCServer() *grpc.Server {
	return s.grpcServer
}
