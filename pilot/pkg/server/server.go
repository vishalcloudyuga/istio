// Copyright 2018 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"fmt"
	"net"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"istio.io/istio/pkg/log"
	pb "istio.io/istio/security/proto"
	"github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/grpc-ecosystem/go-grpc-middleware"

	xdsapi "github.com/envoyproxy/go-control-plane/api"

	"github.com/pkg/errors"
)

const certExpirationBuffer = time.Minute

// Server implements the grpc server.
// Mostly cut&paste from similar mixer and security packages - not reusing since it's
// generic boilerplate.
type Server struct {
	listener  net.Listener
	shutdown  chan error
	server    *grpc.Server

	// Not using GoroutinePool like mixer - push has different model.

	EdsClients map[string]*EdsClient
}

// EdsClient holds an Envoy sidecar connection for pushing EDS changes.
type EdsClient struct {

}

func (s *Server) HandleCSR(ctx context.Context, request *pb.CsrRequest) (*pb.CsrResponse, error) {
	return nil, nil
}

func (s *Server) StreamEndpoints(xdsapi.EndpointDiscoveryService_StreamEndpointsServer) error {
	return errors.New("NOT_IMPLEMENTED")
}

func (s *Server) FetchEndpoints(context.Context, *xdsapi.DiscoveryRequest) (*xdsapi.DiscoveryResponse, error) {


	return nil, errors.New("NOT_IMPLEMENTED")
}

func (s *Server) StreamLoadStats(xdsapi.EndpointDiscoveryService_StreamLoadStatsServer) error {
	return errors.New("NOT_IMPLEMENTED")
}

// Run starts a GRPC server on the specified port.
func (s *Server) Run() error {
	s.shutdown = make(chan error, 1)

	// grpcServer.Serve() is a blocking call, so run it in a goroutine.
	go func() {
		log.Infof("Starting GRPC server on port %d", s.Addr())

		err := s.server.Serve(s.listener)

		// grpcServer.Serve() always returns a non-nil error.
		log.Warnf("GRPC server returns an error: %v", err)
		// notify closer we're done
		s.shutdown <- err
	}()

	return nil
}

// Close cleans up resources used by the server.
func (s *Server) Close() error {
	if s.listener != nil {
		_ = s.listener.Close()
	}
	return nil
}



// Wait waits for the server to exit.
func (s *Server) Wait() error {
	if s.shutdown == nil {
		return fmt.Errorf("server not running")
	}

	err := <-s.shutdown
	s.shutdown = nil
	return err
}


// New creates a new instance of `IstioCAServiceServer`.
func New(hostname string, port int) *Server {
	var grpcOptions []grpc.ServerOption
	//grpcOptions = append(grpcOptions, grpc.MaxConcurrentStreams(uint32(a.MaxConcurrentStreams)))

	var interceptors []grpc.UnaryServerInterceptor

	s := &Server{}

	// TODO: log request interceptor if debug enabled.

	// setup server prometheus monitoring (as final interceptor in chain)
	interceptors = append(interceptors, grpc_prometheus.UnaryServerInterceptor)
	grpc_prometheus.EnableHandlingTimeHistogram()

	grpcOptions = append(grpcOptions, grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(interceptors...)))

	// get the grpc server wired up
	grpc.EnableTracing = true
	s.server = grpc.NewServer(grpcOptions...)

	xdsapi.RegisterEndpointDiscoveryServiceServer(s.server, s)

	return s
}

// Addr returns the address of the server's API port, where gRPC requests can be sent.
func (s *Server) Addr() net.Addr {
	return s.listener.Addr()
}

