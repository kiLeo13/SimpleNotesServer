package websocket

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	"github.com/labstack/gommon/log"
)

const HeaderConnectionID = "X-Connection-Id"

type GatewayClient interface {
	PostToConnection(ctx context.Context, connID string, data interface{}) error
	DeleteConnection(ctx context.Context, connID string) error
}

type AWSGatewayClient struct {
	client *apigatewaymanagementapi.Client
}

func NewAWSGatewayClient(ctx context.Context, endpoint, region string) (*AWSGatewayClient, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	client := apigatewaymanagementapi.NewFromConfig(cfg, func(o *apigatewaymanagementapi.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.Region = region
	})
	return &AWSGatewayClient{client: client}, nil
}

func (g *AWSGatewayClient) PostToConnection(ctx context.Context, connID string, data interface{}) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = g.client.PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{
		ConnectionId: aws.String(connID),
		Data:         payload,
	})

	if err != nil {
		// Usually means user disconnected already
		log.Warnf("failed to push to connection %s: %v", connID, err)
	}
	return err
}

func (g *AWSGatewayClient) DeleteConnection(ctx context.Context, connID string) error {
	_, err := g.client.DeleteConnection(ctx, &apigatewaymanagementapi.DeleteConnectionInput{
		ConnectionId: aws.String(connID),
	})
	return err
}
