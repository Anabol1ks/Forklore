package clients

import (
	repositoryv1 "github.com/Anabol1ks/Forklore/pkg/pb/repository/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type RepositoryClient struct {
	conn   *grpc.ClientConn
	Client repositoryv1.RepositoryServiceClient
}

func NewRepositoryClient(addr string) (*RepositoryClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &RepositoryClient{
		conn:   conn,
		Client: repositoryv1.NewRepositoryServiceClient(conn),
	}, nil
}

func (c *RepositoryClient) Close() error {
	return c.conn.Close()
}
