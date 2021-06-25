package local_dynamo_testing
import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"strings"
)

var (
	ErrUserNotfound = errors.New("cannot find userModel with matchind ID")
)
const (
	userTypeName = "userModel"
)
type Tier string

const (
	TierFREE = "FREE"
	TierPREMIUM = "PREMIUM"
)

type User struct {
	ID string
	Tier Tier
}
type userModel struct {
	Pk       string `dynamodbav:"pk"`
	Sk       string `dynamodbav:"sk"`
	Tier     string `dynamodbav:"userTier"`
	Typename string `dynamodbav:"typeName"`
}

func (u userModel) toSingleTableUser() *User {
	realId := u.Pk[strings.IndexByte(u.Pk, '#')+1:]
	return &User{
		ID:   realId,
		Tier: Tier(u.Tier),
	}
}
func newUser(id string, tier string) *userModel {
	return &userModel{
		Pk:       userPartitionKey(id),
		Sk:       userPartitionKey(id),
		Tier:     tier,
		Typename: userTypeName,
	}
}

func userPartitionKey(id string) string {
	return fmt.Sprintf("USER#%s", id)
}

func (d Db) Create(ctx context.Context, id string, tier Tier) (*User, error) {
	u := newUser(id, string(tier))
	item, err := attributevalue.MarshalMap(u)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal dynamo record, %v", err)
	}
	input := &dynamodb.PutItemInput{
		Item: item,
		TableName: aws.String(d.tableName),
	}
	_, err = d.client.PutItem(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create userModel %w", err)
	}

	return &User{
		ID : id,
		Tier: tier,
	}, nil
}

func (d Db) Get(ctx context.Context, id string) (*User, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(d.tableName),
		Key: map[string]types.AttributeValue{
			"pk": &types.AttributeValueMemberS{Value: userPartitionKey(id)},
			"sk": &types.AttributeValueMemberS{Value: userPartitionKey(id)},
		},
		ProjectionExpression: aws.String("pk, sk, userTier, typeName"),
	}
	output, err := d.client.GetItem(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get userModel %w", err)
	}

	if output == nil {
		return nil, ErrUserNotfound
	}

	item := userModel{}
	err = attributevalue.UnmarshalMap(output.Item, &item)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal Record, %v", err)
	}

	return item.toSingleTableUser(), nil
}
