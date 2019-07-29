#/bin/sh

set -e

REGISTRY=$1
REPOSITORY=$2
AAD_ACCESS_TOKEN=$(az account get-access-token --query accessToken -o tsv)

export ACR_REFRESH_TOKEN=$(curl -s -X POST -H "Content-Type: application/x-www-form-urlencoded" \
	-d "grant_type=access_token&service=$REGISTRY&access_token=$AAD_ACCESS_TOKEN" \
	https://$REGISTRY/oauth2/exchange \
	| jq '.refresh_token' \
	| sed -e 's/^"//' -e 's/"$//')
echo "ACR Refresh Token obtained."

# Create the repo level scope
SCOPE="repository:$REPOSITORY:push"

# to pull multiple repositories passing in multiple scope arguments. 
#&scope="repository:repo:pull,push"

export ACR_ACCESS_TOKEN=$(curl -s -X POST -H "Content-Type: application/x-www-form-urlencoded" \
	-d "grant_type=refresh_token&service=$REGISTRY&scope=$SCOPE&refresh_token=$ACR_REFRESH_TOKEN" \
	https://$REGISTRY/oauth2/token \
	| jq '.access_token' \
	| sed -e 's/^"//' -e 's/"$//')
echo "ACR Access Token obtained."

# Docker Login using the ACR_ACCESS_TOKEN
echo docker login into $REGISTRY
buildah login -u 00000000-0000-0000-0000-000000000000 -p $ACR_ACCESS_TOKEN $REGISTRY