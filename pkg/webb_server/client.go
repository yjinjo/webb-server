package webb_server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
)

const (
	maxRetries = 2
)

var (
	grpcChannels = make(map[string]*GRPCClient)
	channelMutex sync.Mutex
)

type GRPCClient struct {
	conn             *grpc.ClientConn
	options          map[string]interface{}
	channelKey       string
	reflectionClient grpc_reflection_v1alpha.ServerReflectionClient
}

type ClientInterceptor struct {
	metadata   map[string]string
	channelKey string
}

func (c *GRPCClient) GetChannelKey() string {
	return c.channelKey
}

func (c *GRPCClient) GetConnection() *grpc.ClientConn {
	return c.conn
}

func (ci *ClientInterceptor) UnaryInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	md := metadata.New(ci.metadata)
	newCtx := metadata.NewOutgoingContext(ctx, md)

	err := invoker(newCtx, method, req, reply, cc, opts...)
	if err != nil {
		return ci.handleError(err)
	}

	return nil
}

func (ci *ClientInterceptor) handleError(err error) error {
	// 여기에 에러 처리 로직 구현
	// Python 코드의 _check_error 메서드와 유사한 기능 구현
	return err
}

func NewGRPCClient(endpoint string, sslEnabled bool, maxMessageLength int, options map[string]interface{}) (*GRPCClient, error) {
	channelMutex.Lock()
	defer channelMutex.Unlock()

	if options == nil {
		options = make(map[string]interface{})
	}

	metadata, ok := options["metadata"].(map[string]string)
	if !ok {
		metadata = make(map[string]string)
	}

	interceptor := &ClientInterceptor{
		metadata:   metadata,
		channelKey: endpoint,
	}

	if client, exists := grpcChannels[endpoint]; exists {
		return client, nil
	}

	var opts []grpc.DialOption
	if maxMessageLength > 0 {
		opts = append(opts, grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(maxMessageLength),
			grpc.MaxCallSendMsgSize(maxMessageLength),
		))
	}

	if sslEnabled {
		creds, err := credentials.NewClientTLSFromFile("path/to/cert", "")
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS credentials: %v", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	interceptor := &ClientInterceptor{
		metadata:   options["metadata"].(map[string]string),
		channelKey: endpoint,
	}
	opts = append(opts, grpc.WithUnaryInterceptor(interceptor.UnaryInterceptor))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	conn, err := grpc.DialContext(ctx, endpoint, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %v", err)
	}

	client := &GRPCClient{
		conn:       conn,
		options:    options,
		channelKey: endpoint,
	}

	client.reflectionClient = grpc_reflection_v1alpha.NewServerReflectionClient(conn)

	grpcChannels[endpoint] = client
	return client, nil
}

func (c *GRPCClient) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

// 여기에 추가적인 메서드 구현 (예: ListServices, GetService 등)

func GetGRPCMethod(uriInfo map[string]interface{}) (interface{}, error) {
	endpoint := uriInfo["endpoint"].(string)
	sslEnabled := uriInfo["ssl_enabled"].(bool)
	service := uriInfo["service"].(string)
	method := uriInfo["method"].(string)

	client, err := NewGRPCClient(endpoint, sslEnabled, 0, map[string]interface{}{})
	if err != nil {
		return nil, fmt.Errorf("gRPC 클라이언트 생성 실패: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 서버 리플렉션 정보 가져오기
	stream, err := client.reflectionClient.ServerReflectionInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("서버 리플렉션 정보 가져오기 실패: %v", err)
	}

	// 서비스 정보 요청
	if err := stream.Send(&grpc_reflection_v1alpha.ServerReflectionRequest{
		MessageRequest: &grpc_reflection_v1alpha.ServerReflectionRequest_FileContainingSymbol{
			FileContainingSymbol: service,
		},
	}); err != nil {
		return nil, fmt.Errorf("리플렉션 요청 전송 실패: %v", err)
	}

	resp, err := stream.Recv()
	if err != nil {
		return nil, fmt.Errorf("리플렉션 응답 수신 실패: %v", err)
	}

	fileDescProto := &descriptorpb.FileDescriptorProto{}
	if err := proto.Unmarshal(resp.GetFileDescriptorResponse().FileDescriptorProto[0], fileDescProto); err != nil {
		return nil, fmt.Errorf("파일 디스크립터 언마샬링 실패: %v", err)
	}

	fileDesc, err := protodesc.NewFile(fileDescProto, nil)
	if err != nil {
		return nil, fmt.Errorf("파일 디스크립터 생성 실패: %v", err)
	}

	// 서비스 찾기
	var serviceDesc *grpc.ServiceDesc
	services := fileDesc.Services()
	for i := 0; i < services.Len(); i++ {
		s := services.Get(i)
		if string(s.Name()) == service {
			serviceDesc = &grpc.ServiceDesc{
				ServiceName: service,
				HandlerType: (*interface{})(nil),
				Methods:     make([]grpc.MethodDesc, 0),
				Streams:     make([]grpc.StreamDesc, 0),
			}

			// 메서드 찾기
			methods := s.Methods()
			for j := 0; j < methods.Len(); j++ {
				m := methods.Get(j)
				if string(m.Name()) == method {
					methodDesc := grpc.MethodDesc{
						MethodName: method,
						Handler:    nil, // 실제 핸들러는 서버 측에 있으므로 여기서는 nil
					}
					serviceDesc.Methods = append(serviceDesc.Methods, methodDesc)
					break
				}
			}
			break
		}
	}

	if serviceDesc == nil || len(serviceDesc.Methods) == 0 {
		return nil, fmt.Errorf("메서드 %s를 서비스 %s에서 찾을 수 없습니다", method, service)
	}

	// 메서드에 대한 gRPC 호출 함수 생성
	methodFunc := func(ctx context.Context, req interface{}, opts ...grpc.CallOption) (interface{}, error) {
		var reply interface{}
		err := client.conn.Invoke(ctx, fmt.Sprintf("/%s/%s", service, method), req, &reply, opts...)
		return reply, err
	}

	return methodFunc, nil
}
