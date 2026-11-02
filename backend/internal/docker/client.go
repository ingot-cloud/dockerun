package docker

import (
	"context"
	"time"

	"github.com/docker/docker/client"
)

var Client *client.Client

func Init() error {
	closePathBridge()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	Client = cli
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err = startPathBridge(ctx); err != nil {
		cli.Close()
		return err
	}
	return nil
}

func Close() {
	closePathBridge()
	if Client != nil {
		Client.Close()
	}
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
