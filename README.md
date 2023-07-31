<h1 align="center">
  <br>
  <img height="300" src="https://github.com/mudler/microAGI/assets/2420543/7717fafb-de72-4a2d-a47a-229fc64b5716"> <br>
    μAGI (microAGI)
<br>
</h1>

From the [LocalAI](https://localai.io) author, microAGI. 100% Local AI assistant.

Note: this is a fun project, not a serious one. Be warned!

## What is μAGI?

It is a dead simple experiment to show how to tie the various LocalAI functionalities to create a virtual assistant that can do tasks. It is simple on purpose, trying to be minimalistic and easy to understand and customize.

It is different from babyAGI or AutoGPT as it uses [OpenAI functions](https://openai.com/blog/function-calling-and-other-api-updates), but locally with [LocalAI](https://localai.io) (no API keys needed!)

## Quick start

No frills, just run docker-compose and start chatting with your virtual assistant:

```bash
docker-compose run --build -i --rm microagi
```

### Test it!

Ask it to:

- "Can you create the agenda for tomorrow?"
  -> and watch it search through memories to get your agenda!
- "How are you?"
  -> and watch it engaging into dialogues with long-term memory
- "I want you to act as a marketing and sales guy in a startup company. I want you to come up with a plan to support our new latest project, XXX, which is an open source project. you are free to come up with creative ideas to engage and attract new people to the project. The XXX project is XXX."

### How it works?

`microAGI` just does the minimal around LocalAI functions to create a virtual assistant that can do generic tasks. It works by an endless loop of `intent detection`, `function invocation`, `self-evaluation` and `reply generation` (if it decides to reply! :)). The agent is capable of planning complex tasks by invoking multiple functions, and remember things from the conversation.

In a nutshell, it goes like this:

- Decide based on the conversation history if it needs to take an action by using functions. It uses the LLM to detect the intent from the conversation.
- if it need to take an action (e.g. "remember something from the conversation" ) or generate complex tasks ( executing a chain of functions to achieve a goal ) it invokes the functions
- it re-evaluates if it needs to do any other action
- return the result back to the LLM to generate a reply for the user

Under the hood LocalAI converts functions to llama.cpp BNF grammars. While OpenAI fine-tuned a model to reply to functions, LocalAI constrains the LLM to follow grammars. This is a much more efficient way to do it, and it is also more flexible as you can define your own functions and grammars. For learning more about this, check out the [LocalAI documentation](https://localai.io/docs/llm) and my tweet that explains how it works under the hoods: https://twitter.com/mudler_it/status/1675524071457533953.

## Roadmap

- [x] 100% Local, with Local AI. NO API KEYS NEEDED!
- [x] Create a simple virtual assistant
- [x] Make the virtual assistant do functions like store long-term memory and autonomously search between them when needed
- [x] Create the assistant avatar with Stable Diffusion
- [x] Give it a voice 
- [] Get voice input (push to talk or wakeword)
- [] Make a REST API (OpenAI compliant?) so can be plugged by e.g. a third party service
- [] Take a system prompt so can act with a "character" (e.g. "answer in rick and morty style")

## Development

Run docker-compose with main.py checked-out:

```bash
docker-compose run -v main.py:/app/main.py -i --rm microagi
```