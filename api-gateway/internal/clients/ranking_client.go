package clients

import (
	rankingv1 "github.com/Anabol1ks/Forklore/pkg/pb/ranking/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type RankingClient struct {
	conn   *grpc.ClientConn
	Client rankingv1.RankingServiceClient
}

func NewRankingClient(addr string) (*RankingClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &RankingClient{
		conn:   conn,
		Client: rankingv1.NewRankingServiceClient(conn),
	}, nil
}

func (c *RankingClient) Close() error {
	return c.conn.Close()
}
