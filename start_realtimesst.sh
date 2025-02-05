docker build -t realtimesst -f Dockerfile.realtimesst .
docker run -v $PWD/cache:/root/.cache -p 8012:8012 -p 8011:8011 -ti --rm realtimesst -w "jarvis" -D
