package docker

import (
	"context"

	"github.com/docker/docker/client"
)

var Client *client.Client

func Init() error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	Client = cli
	return nil
}

func GetClient() *client.Client {
	return Client
}

func IsAvailable() bool {
	if Client == nil {
		return false
	}
	_, err := Client.Ping(context.Background())
	return err == nil
}

