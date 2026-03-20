package clients

import (
	searchv1 "github.com/Anabol1ks/Forklore/pkg/pb/search/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type SearchClient struct {
	conn   *grpc.ClientConn
	Client searchv1.SearchServiceClient
}

func NewSearchClient(addr string) (*SearchClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &SearchClient{
		conn:   conn,
		Client: searchv1.NewSearchServiceClient(conn),
	}, nil
}

func (c *SearchClient) Close() error {
	return c.conn.Close()
}
