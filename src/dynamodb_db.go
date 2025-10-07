//go:build production

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type dynamoDatabase struct {
	client       *dynamodb.Client
	storiesTable string
	playersTable string
}

func newDatabase() (Database, error) {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	db := &dynamoDatabase{
		client:       dynamodb.NewFromConfig(cfg),
		storiesTable: getEnvOrDefault("DYNAMO_STORIES_TABLE", "stories"),
		playersTable: getEnvOrDefault("DYNAMO_PLAYERS_TABLE", "players"),
	}

	if err := db.ensureTables(ctx); err != nil {
		return nil, err
	}

	return db, nil
}

func getEnvOrDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func (d *dynamoDatabase) ensureTables(ctx context.Context) error {
	if err := d.ensureTable(ctx, d.storiesTable); err != nil {
		return err
	}
	if err := d.ensureTable(ctx, d.playersTable); err != nil {
		return err
	}
	return nil
}

func (d *dynamoDatabase) ensureTable(ctx context.Context, tableName string) error {
	_, err := d.client.DescribeTable(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(tableName)})
	if err == nil {
		return nil
	}

	var notFound *types.ResourceNotFoundException
	if errors.As(err, &notFound) {
		input := &dynamodb.CreateTableInput{
			TableName: aws.String(tableName),
			AttributeDefinitions: []types.AttributeDefinition{
				{
					AttributeName: aws.String("id"),
					AttributeType: types.ScalarAttributeTypeS,
				},
			},
			KeySchema: []types.KeySchemaElement{
				{
					AttributeName: aws.String("id"),
					KeyType:       types.KeyTypeHash,
				},
			},
			BillingMode: types.BillingModePayPerRequest,
		}
		if _, err := d.client.CreateTable(ctx, input); err != nil {
			return fmt.Errorf("create dynamodb table %s: %w", tableName, err)
		}

		waiter := dynamodb.NewTableExistsWaiter(d.client)
		if err := waiter.Wait(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(tableName)}, 5*time.Minute); err != nil {
			return fmt.Errorf("wait for table %s: %w", tableName, err)
		}
		return nil
	}

	return fmt.Errorf("describe table %s: %w", tableName, err)
}

func (d *dynamoDatabase) Close() error {
	return nil
}

func (d *dynamoDatabase) UpsertStory(title string) error {
	ctx := context.Background()
	_, err := d.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.storiesTable),
		Item: map[string]types.AttributeValue{
			"id":    &types.AttributeValueMemberS{Value: "1"},
			"title": &types.AttributeValueMemberS{Value: title},
		},
	})
	if err != nil {
		return fmt.Errorf("upsert story: %w", err)
	}
	return nil
}

func (d *dynamoDatabase) GetStory() (Story, error) {
	ctx := context.Background()
	out, err := d.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(d.storiesTable),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: "1"},
		},
	})
	if err != nil {
		return Story{}, fmt.Errorf("get story: %w", err)
	}
	if out.Item == nil {
		return Story{Id: 1}, nil
	}

	story := Story{Id: 1}
	if title, ok := out.Item["title"].(*types.AttributeValueMemberS); ok {
		story.Title = title.Value
	}
	return story, nil
}

func (d *dynamoDatabase) UpsertPlayer(id, name, points string, observer bool) error {
	ctx := context.Background()

	if name != "" {
		existing, err := d.getPlayerByName(ctx, name)
		if err != nil {
			return err
		}
		if existing != nil && existing.Id != id {
			return fmt.Errorf("name already taken, use a new name")
		}
	}

	_, err := d.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.playersTable),
		Item: map[string]types.AttributeValue{
			"id":       &types.AttributeValueMemberS{Value: id},
			"name":     &types.AttributeValueMemberS{Value: name},
			"points":   &types.AttributeValueMemberS{Value: points},
			"observer": &types.AttributeValueMemberBOOL{Value: observer},
		},
	})
	if err != nil {
		return fmt.Errorf("upsert player: %w", err)
	}
	return nil
}

func (d *dynamoDatabase) getPlayerByName(ctx context.Context, name string) (*Player, error) {
	if name == "" {
		return nil, nil
	}

	out, err := d.client.Scan(ctx, &dynamodb.ScanInput{
		TableName:        aws.String(d.playersTable),
		FilterExpression: aws.String("#n = :name"),
		ExpressionAttributeNames: map[string]string{
			"#n": "name",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":name": &types.AttributeValueMemberS{Value: name},
		},
		ProjectionExpression: aws.String("id, name"),
		Limit:                aws.Int32(1),
	})
	if err != nil {
		return nil, fmt.Errorf("scan player by name: %w", err)
	}

	if len(out.Items) == 0 {
		return nil, nil
	}

	item := out.Items[0]
	player := d.playerFromItem(item)
	return &player, nil
}

func (d *dynamoDatabase) GetPlayerByID(id string) (Player, error) {
	ctx := context.Background()
	out, err := d.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(d.playersTable),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return Player{}, fmt.Errorf("get player by id: %w", err)
	}
	if out.Item == nil {
		return Player{}, fmt.Errorf("no player found with id: %s", id)
	}

	return d.playerFromItem(out.Item), nil
}

func (d *dynamoDatabase) GetPlayers() ([]Player, error) {
	ctx := context.Background()
	out, err := d.client.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String(d.playersTable),
	})
	if err != nil {
		return nil, fmt.Errorf("scan players: %w", err)
	}

	players := make([]Player, 0, len(out.Items))
	for _, item := range out.Items {
		players = append(players, d.playerFromItem(item))
	}
	return players, nil
}

func (d *dynamoDatabase) ClearPlayersPoints() error {
	players, err := d.GetPlayers()
	if err != nil {
		return err
	}

	ctx := context.Background()
	for _, player := range players {
		_, err := d.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
			TableName: aws.String(d.playersTable),
			Key: map[string]types.AttributeValue{
				"id": &types.AttributeValueMemberS{Value: player.Id},
			},
			UpdateExpression: aws.String("SET points = :empty"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":empty": &types.AttributeValueMemberS{Value: ""},
			},
		})
		if err != nil {
			return fmt.Errorf("clear points for player %s: %w", player.Id, err)
		}
	}
	return nil
}

func (d *dynamoDatabase) ClearStory() error {
	ctx := context.Background()
	_, err := d.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.storiesTable),
		Item: map[string]types.AttributeValue{
			"id":    &types.AttributeValueMemberS{Value: "1"},
			"title": &types.AttributeValueMemberS{Value: ""},
		},
	})
	if err != nil {
		return fmt.Errorf("clear story: %w", err)
	}
	return nil
}

func (d *dynamoDatabase) DeletePlayer(id string) error {
	ctx := context.Background()
	_, err := d.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(d.playersTable),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return fmt.Errorf("delete player: %w", err)
	}
	return nil
}

func (d *dynamoDatabase) playerFromItem(item map[string]types.AttributeValue) Player {
	player := Player{}
	if id, ok := item["id"].(*types.AttributeValueMemberS); ok {
		player.Id = id.Value
	}
	if name, ok := item["name"].(*types.AttributeValueMemberS); ok {
		player.Name = name.Value
	}
	if points, ok := item["points"].(*types.AttributeValueMemberS); ok {
		player.Points = points.Value
	}
	if observer, ok := item["observer"].(*types.AttributeValueMemberBOOL); ok {
		player.Observer = observer.Value
	}
	return player
}
