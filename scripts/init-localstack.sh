#!/bin/bash
set -e

ENDPOINT="http://localhost:4566"
REGION="us-east-1"
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test

echo "Creating DLQ..."
DLQ_URL=$(aws sqs create-queue \
  --queue-name conversion-queue-dlq \
  --region "$REGION" \
  --endpoint-url "$ENDPOINT" \
  --attributes '{"VisibilityTimeout":"300"}' \
  --query 'QueueUrl' --output text)

echo "DLQ URL: $DLQ_URL"

DLQ_ARN=$(aws sqs get-queue-attributes \
  --queue-url "$DLQ_URL" \
  --attribute-names QueueArn \
  --region "$REGION" \
  --endpoint-url "$ENDPOINT" \
  --query 'Attributes.QueueArn' --output text)

echo "DLQ ARN: $DLQ_ARN"

echo "Creating main queue with redrive policy..."
aws sqs create-queue \
  --queue-name conversion-queue \
  --region "$REGION" \
  --endpoint-url "$ENDPOINT" \
  --attributes "{\"VisibilityTimeout\":\"300\",\"RedrivePolicy\":\"{\\\"deadLetterTargetArn\\\":\\\"$DLQ_ARN\\\",\\\"maxReceiveCount\\\":3}\"}"

EXISTING_TABLES=$(aws dynamodb list-tables --region "$REGION" --endpoint-url "$ENDPOINT" --query 'TableNames[]' --output text)

if echo "$EXISTING_TABLES" | grep -qw "playswap-conversions"; then
  echo "Table playswap-conversions already exists, skipping..."
else
  echo "Creating DynamoDB table: playswap-conversions..."
  aws dynamodb create-table \
    --table-name playswap-conversions \
    --region "$REGION" \
    --endpoint-url "$ENDPOINT" \
    --attribute-definitions \
      AttributeName=id,AttributeType=S \
      AttributeName=userId,AttributeType=S \
      AttributeName=createdAt,AttributeType=S \
    --key-schema \
      AttributeName=id,KeyType=HASH \
    --global-secondary-indexes \
      '[{"IndexName":"userId-createdAt-index","KeySchema":[{"AttributeName":"userId","KeyType":"HASH"},{"AttributeName":"createdAt","KeyType":"RANGE"}],"Projection":{"ProjectionType":"ALL"}}]' \
    --billing-mode PAY_PER_REQUEST
fi

if echo "$EXISTING_TABLES" | grep -qw "playswap-conversion-logs"; then
  echo "Table playswap-conversion-logs already exists, skipping..."
else
  echo "Creating DynamoDB table: playswap-conversion-logs..."
  aws dynamodb create-table \
    --table-name playswap-conversion-logs \
    --region "$REGION" \
    --endpoint-url "$ENDPOINT" \
    --attribute-definitions \
      AttributeName=id,AttributeType=S \
      AttributeName=createdAt,AttributeType=S \
      AttributeName=conversionId,AttributeType=S \
    --key-schema \
      AttributeName=id,KeyType=HASH \
      AttributeName=createdAt,KeyType=RANGE \
    --global-secondary-indexes \
      '[{"IndexName":"conversionId-createdAt-index","KeySchema":[{"AttributeName":"conversionId","KeyType":"HASH"},{"AttributeName":"createdAt","KeyType":"RANGE"}],"Projection":{"ProjectionType":"ALL"}}]' \
    --billing-mode PAY_PER_REQUEST

  echo "Enabling TTL on playswap-conversion-logs..."
  aws dynamodb update-time-to-live \
    --table-name playswap-conversion-logs \
    --region "$REGION" \
    --endpoint-url "$ENDPOINT" \
    --time-to-live-specification Enabled=true,AttributeName=ttl
fi

echo "Done."
