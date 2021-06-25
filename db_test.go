package local_dynamo_testing

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/dihmuzikien/local-dynamo-testing/testhelper"
	"testing"
)

func newTestDynamo(t *testing.T) (*Db, func(), error){
	t.Helper()
	customResolver := aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
		return aws.Endpoint{
			PartitionID:   "aws",
			URL:           "http://localhost:8000",
		}, nil
	})
	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithEndpointResolver(customResolver),
	)
	if err != nil {
		return nil, nil, err
	}

	svc := dynamodb.NewFromConfig(awsCfg)
	tableName := testhelper.GenerateTableName()
	cleanup := testhelper.SetupTable(t, tableName, svc)
	db := &Db{
		client: svc,
		tableName: tableName,
	}
	return db, cleanup, nil
}

func TestDynamo(t *testing.T){
	t.Run("test successful flow", func(t *testing.T) {
		db, cleanup, err := newTestDynamo(t)
		t.Cleanup(cleanup)
		if err != nil {
			t.Fatal("cannot lauch dynamo")
		}
		testcases := []struct {
			id string
			tier Tier
		}{
			{id: "first-id", tier: TierFREE},
			{id: "second-id", tier: TierFREE},
			{id: "third-id", tier: TierPREMIUM},
		}
		for _, tc := range testcases {
			_, err = db.Create(context.TODO(), tc.id, tc.tier)
			if err != nil {
				t.Errorf("failed create %v", err)
			}
			u, terr := db.Get(context.TODO(), tc.id)
			if terr != nil {
				t.Errorf("failed create %v", err)
			}
			if u.Tier != tc.tier{
				t.Errorf("unexpected user")
			}
		}
	})

}
