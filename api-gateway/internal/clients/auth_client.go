package clients

import (
	authv1 "github.com/Anabol1ks/Forklore/pkg/pb/auth/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type AuthClient struct {
	conn   *grpc.ClientConn
	Client authv1.AuthServiceClient
}

func NewAuthClient(addr string) (*AuthClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &AuthClient{
		conn:   conn,
		Client: authv1.NewAuthServiceClient(conn),
	}, nil
}

func (c *AuthClient) Close() error {
	return c.conn.Close()
}
