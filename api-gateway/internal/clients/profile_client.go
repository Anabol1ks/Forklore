package clients

import (
	profilev1 "github.com/Anabol1ks/Forklore/pkg/pb/profile/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ProfileClient struct {
	conn   *grpc.ClientConn
	Client profilev1.ProfileServiceClient
}

func NewProfileClient(addr string) (*ProfileClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &ProfileClient{
		conn:   conn,
		Client: profilev1.NewProfileServiceClient(conn),
	}, nil
}

func (c *ProfileClient) Close() error {
	return c.conn.Close()
}
