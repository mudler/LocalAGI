# miniAGI

From the [LocalAI](https://localai.io) author, miniAGI. 100% Local AI assistant.

Note: this is a fun project, not a serious one. It's a toy, not a tool. Be warned!

## What is miniAGI?

It is a dead simple experiment to show how to tie the various LocalAI functionalities to create a virtual assistant that can do tasks. It is simple on purpose, trying to be minimalistic and easy to understand and customize.

## Quick start

No frills, just run docker-compose and start chatting with your virtual assistant:

```bash
docker-compose run --build -i --rm miniagi
```

## Roadmap

- [x] 100% Local, with Local AI. NO API KEYS NEEDED!
- [x] Create a simple virtual assistant
- [x] Make the virtual assistant do functions like store long-term memory and autonomously search between them when needed
- [] Create the assistant avatar with Stable Diffusion
- [] Give it a voice (push to talk or wakeword)
- [] Get voice input
- [] Make a REST API (OpenAI compliant?) so can be plugged by e.g. a third party service
- [] Take a system prompt so can act with a "character" (e.g. "answer in rick and morty style")

## Development

Run docker-compose with main.py checked-out:

```bash
docker-compose run -v main.py:/app/main.py -i --rm miniagi
```