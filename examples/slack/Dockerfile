FROM python:3.11.3-slim-buster
WORKDIR /app/
COPY requirements.txt /app/

RUN apt-get update && apt-get install build-essential git -y
RUN pip install -U pip && pip install -r requirements.txt
COPY *.py /app/
COPY *.sh /app/
RUN mkdir /app/app/
COPY app/*.py /app/app/
ENTRYPOINT /app/entrypoint.sh

# docker build . -t your-repo/chat-gpt-in-slack
# export SLACK_APP_TOKEN=xapp-...
# export SLACK_BOT_TOKEN=xoxb-...
# export OPENAI_API_KEY=sk-...
# docker run -e SLACK_APP_TOKEN=$SLACK_APP_TOKEN -e SLACK_BOT_TOKEN=$SLACK_BOT_TOKEN -e OPENAI_API_KEY=$OPENAI_API_KEY -it your-repo/chat-gpt-in-slack
