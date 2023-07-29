# miniAGI

From the [LocalAI](https://localai.io) author, miniAGI. 100% Local AI assistant.

Note: this is a fun project, not a serious one. Be warned!

## What is miniAGI?

It is a dead simple experiment to show how to tie the various LocalAI functionalities to create a virtual assistant that can do tasks. It is simple on purpose, trying to be minimalistic and easy to understand and customize.

## Quick start

No frills, just run docker-compose and start chatting with your virtual assistant:

```bash
docker-compose run --build -i --rm miniagi
```

### Test it!

Ask it to:

- "Can you create the agenda for tomorrow?"
  -> and watch it search through memories to get your agenda!
- "How are you?"
  -> and watch it engaging into dialogues with long-term memory

### How it works?

miniAGI uses `LocalAI` and OpenAI functions. 

The approach is the following:
- Decide based on the conversation history if we need to take an action by using functions.It uses the LLM to detect the intent from the conversation.
- if we need to take an action (e.g. "remember something from the conversation" ) or generate complex tasks ( executing a chain of functions to achieve a goal)

Under the hood:

- LocalAI converts functions to llama.cpp BNF grammars

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