package clients

import (
	contentv1 "github.com/Anabol1ks/Forklore/pkg/pb/content/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ContentClient struct {
	conn   *grpc.ClientConn
	Client contentv1.ContentServiceClient
}

func NewContentClient(addr string) (*ContentClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &ContentClient{
		conn:   conn,
		Client: contentv1.NewContentServiceClient(conn),
	}, nil
}

func (c *ContentClient) Close() error {
	return c.conn.Close()
}
