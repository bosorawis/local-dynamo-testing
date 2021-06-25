package testhelper

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"strconv"
	"testing"
	"time"
)

// GenerateTableName randomly generates a unique table name
// format: uuid_<timestamp>
func GenerateTableName() string {
	t := strconv.FormatInt(time.Now().Unix(), 10)
	u,_ := uuid.NewUUID()
	return fmt.Sprintf("%s_%s", t, u )
}

func SetupTable(t *testing.T, tableName string, dynamoClient *dynamodb.Client) func(){
	t.Helper()
	createInput := &dynamodb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("pk"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("sk"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("pk"),
				KeyType:       types.KeyTypeHash,
			},
			{
				AttributeName: aws.String("sk"),
				KeyType:       types.KeyTypeRange,
			},
		},
		BillingMode: types.BillingModePayPerRequest,
		TableName: aws.String(tableName),
	}

	_, err := dynamoClient.CreateTable(context.TODO(), createInput)
	if err != nil {
		t.Fatal(err)
	}

	// now wait until it's ACTIVE
	start := time.Now()
	time.Sleep(time.Second)

	describeInput := &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	}
	for {
		resp, err := dynamoClient.DescribeTable(context.TODO(), describeInput)
		if err != nil {
			t.Fatal(err)
		}
		if resp.Table.TableStatus == types.TableStatusActive {
			break
		}
		if time.Since(start) > time.Minute {
			t.Fatalf("timed out creating DynamoDB table %s", tableName)
		}
		time.Sleep(3 * time.Second)
	}
	return func() {
		params := &dynamodb.DeleteTableInput{
			TableName: aws.String(tableName),
		}
		_, err := dynamoClient.DeleteTable(context.TODO(), params)
		if err != nil {
			t.Logf("WARNING: Failed to delete the test DynamoDB table %q. It has been left in your AWS account and may incur charges. (error was %s)", tableName, err)
		}
	}
}