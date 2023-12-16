docker build -t slack-bot .
docker run -v $PWD/data:/data --rm -ti --env-file .dockerenv slack-bot