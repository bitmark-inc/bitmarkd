#!/usr/bin/env bash

sed -i '' 's/setupTestLogger/fixtures.SetupTestLogger/g' $1
sed -i '' 's/teardownTestLogger/fixtures.TeardownTestLogger/g' $1
sed -i '' 's/logCategory/fixtures.LogCategory/g' $1
sed -i '' 's/issuerPublicKey/fixtures.IssuerPublicKey/g' $1
sed -i '' 's/issuerPrivateKey/fixtures.IssuerPrivateKey/g' $1
sed -i '' 's/litecoinAddress/fixtures.LitecoinAddress/g' $1
sed -i '' 's/rateLimitN/ratelimit.LimitN/g' $1
