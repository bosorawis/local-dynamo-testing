# Local DynamoDB testing
Small project demonstrating how I test interaction with DynamoDB locally


## Requirements
- [docker](https://docs.docker.com/get-docker/)
- [aws-cli](https://aws.amazon.com/cli/)

## Getting Started

Launch dynamoDB by running
```bash
docker run -d -p 8000:8000 amazon/dynamodb-local -jar DynamoDBLocal.jar -sharedDb
```

This will run DynamoDB API on port 8080. Test the API by running
```bash
# just append `--endpoint-url http://localhost:8000` to any aws-cli command to test
aws dynamodb list-tables --endpoint-url http://localhost:8000
```

## Components



### Local dynamoDB client
To point a DynamoDB client to local DB, use the following snippets
```go

customResolver := aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
    return aws.Endpoint{
        PartitionID:   "aws",
        URL:           "http://localhost:8000", // IMPORTANT
    }, nil
})
awsCfg, err := config.LoadDefaultConfig(context.TODO(),
    config.WithEndpointResolver(customResolver),
)
if err != nil {
    return nil, nil, err
}

svc := dynamodb.NewFromConfig(awsCfg)
```

To set up dynamoDB locally, just call `CreateTable` method on `dynamodb.Client.CreateTable()` as you normally would. 
By instantiating `dynamodb.Client` with `aws.Endpoint.URL = "http://localhost:8000"`, that tells the dynamoDB client 
to call an API at that endpoint instead of the real one.


### Table setup
Once you have a locally pointed dynamoDB client, you can simply call `client.CreateTable`
to create a table.

Use the following snippets by passing in the local client as `dynamodb.Client`. Note that this 
method returns a function that's used to clean up after itself. See example below on how to use the cleanup func.
```go
// testhelper/db.go
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
            t.Logf("WARNING: Failed to delete the test DynamoDB table %q", err)
        }
    }
}

```

Example test:
```go

t.Run("testing", func(t *testing.T) {
    // setup custom resolver
    svc := dynamodb.NewFromConfig(awsCfg)
    cleanup := SetupTable(t, "table", svc)
    t.Cleanup(cleanup) // or defer cleanup()
    // Do the actual test
}
```

## Important Notes

AWS credentials are still required eventhough no real AWS API being called.
That means if you're running your test in a CI system, you need to set credentials on the runner.
These credentials **DO NOT** have to be valid though.

Example:
```bash
# Skip if you had AWS Credentials already
export AWS_ACCESS_KEY_ID=fakeCred
export AWS_SECRET_ACCESS_KEY=fakeCred
export AWS_DEFAULT_REGION=us-west-2
# --------------------------------------
docker run --name dynamo-local -d -p 8000:8000 amazon/dynamodb-local -jar DynamoDBLocal.jar -sharedDb 
go test ./... -count=1 
docker stop dynamo-local 
```

