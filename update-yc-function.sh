#!/bin/bash 

FUNC=$1

DIR=$(dirname $0)
. $DIR/awssecret.sh
AUTHENV=AWS_REGION=$AWS_REGION,AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID,AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY 

case $FUNC in
	apigw)
		yc serverless api-gateway update --spec $DIR/apigw.yaml d5d4hhdgu8rlgdm0s5gp
		;;
	upload)
		yc serverless function version create --function-id d4epjth7l1h1jkv9kurk \
			--environment $AUTHENV --runtime golang116 \
			--entrypoint upload.Form --source-path $DIR/functions/upload
		;;
	render)
		yc serverless function version create --function-id d4e9hfvt317cm7jm1sgt \
			--environment $AUTHENV --execution-timeout 30s --runtime golang116 \
			--entrypoint render.Render --source-path $DIR/functions/render
		;;	
	*)
		echo "Do not know how to update $FUNC"
esac
	