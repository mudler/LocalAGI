<h1 align="center">
  <br>
  <img height="300" src="https://github.com/mudler/microAGI/assets/2420543/7717fafb-de72-4a2d-a47a-229fc64b5716"> <br>
    μAGI (microAGI)
<br>
</h1>

From the [LocalAI](https://localai.io) author, μAGI. 100% Local AI assistant.

[AutoGPT](https://github.com/Significant-Gravitas/Auto-GPT), [babyAGI](https://github.com/yoheinakajima/babyagi), ... and now LocalAGI!

LocalAGI is a microAGI that you can run locally.

The goal is:
- Keep it simple, hackable and easy to understand
- If you can't run it locally, it is not AGI
- No API keys needed, No cloud services needed, 100% Local
- Do a smart-agent/virtual assistant that can do tasks
- Small set of dependencies
- Run with Docker everywhere

Note: this is a fun project, not a serious one. Be warned!

## What is μAGI?

It is a dead simple experiment to show how to tie the various LocalAI functionalities to create a virtual assistant that can do tasks. It is simple on purpose, trying to be minimalistic and easy to understand and customize for everyone.

It is different from babyAGI or AutoGPT as it uses [LocalAI functions](https://localai.io/features/openai-functions/) - it is a from scratch attempt built on purpose to run locally with [LocalAI](https://localai.io) (no API keys needed!) instead of expensive, cloud services. It sets apart from other projects as it strives to be small, and easy to fork on.

## Quick start

No frills, just run docker-compose and start chatting with your virtual assistant:

```bash
docker-compose run -i --rm microagi
```

## How to use it

By default microagi starts in interactive mode

### Basics

### Advanced

microagi has several options in the CLI to tweak the experience:

- `--system-prompt` is the system prompt to use. If not specified, it will use none.
- `--prompt` is the prompt to use for batch mode. If not specified, it will default to interactive mode.
- `--interactive` is the interactive mode. When used with `--prompt` will drop you in an interactive session after the first prompt is evaluated.
- `--skip-avatar` will skip avatar creation. Useful if you want to run it in a headless environment.
- `--re-evaluate` will re-evaluate if another action is needed or we have completed the user request.
- `--postprocess` will postprocess the reasoning for analysis.
- `--subtask-context` will include context in subtasks.
- `--search-results` is the number of search results to use.
- `--plan-message` is the message to use during planning. You can override the message for example to force a plan to have a different message.
- `--tts-api-base` is the TTS API base. Defaults to `http://api:8080`.
- `--localai-api-base` is the LocalAI API base. Defaults to `http://api:8080`.
- `--images-api-base` is the Images API base. Defaults to `http://api:8080`.
- `--embeddings-api-base` is the Embeddings API base. Defaults to `http://api:8080`.
- `--functions-model` is the functions model to use. Defaults to `functions`.
- `--embeddings-model` is the embeddings model to use. Defaults to `all-MiniLM-L6-v2`.
- `--llm-model` is the LLM model to use. Defaults to `gpt-4`.
- `--tts-model` is the TTS model to use. Defaults to `en-us-kathleen-low.onnx`.
- `--stablediffusion-model` is the Stable Diffusion model to use. Defaults to `stablediffusion`.
- `--stablediffusion-prompt` is the Stable Diffusion prompt to use. Defaults to `DEFAULT_PROMPT`.
- `--force-action` will force a specific action.
- `--debug` will enable debug mode.


### Test it!

Ask it to:

- "Can you create the agenda for tomorrow?"
  -> and watch it search through memories to get your agenda!
- "How are you?"
  -> and watch it engaging into dialogues with long-term memory
- "I want you to act as a marketing and sales guy in a startup company. I want you to come up with a plan to support our new latest project, XXX, which is an open source project. you are free to come up with creative ideas to engage and attract new people to the project. The XXX project is XXX."

#### Examples

Road trip planner by limiting searching to internet to 3 results only:

```bash
docker-compose run -i --rm microagi --skip-avatar --subtask-context --postprocess --prompt "prepare a plan for my roadtrip to san francisco" --search-results 3
```

Limit results of planning to 3 steps:

```bash
docker-compose run -v $PWD/main.py:/app/main.py -i --rm microagi --skip-avatar --subtask-context --postprocess --prompt "do a plan for my roadtrip to san francisco" --search-results 1 --plan-message "The assistant replies with a plan of 3 steps to answer the request with a list of subtasks with logical steps. The reasoning includes a self-contained, detailed and descriptive instruction to fullfill the task."
```

### Customize

To use a different model, you can see the examples in the `config` folder.
To select a model, modify the `.env` file and change the `PRELOAD_MODELS_CONFIG` variable to use a different configuration file.

### Caveats

The "goodness" of a model has a big impact on how μAGI works. Currently `13b` models are powerful enough to actually able to perform multi-step tasks or do more actions. However, it is quite slow when running on CPU (no big surprise here).

The context size is a limitation - you can find in the `config` examples to run with superhot 8k context size, but the quality is not good enough to perform complex tasks.

### How it works?

`μAGI` just does the minimal around LocalAI functions to create a virtual assistant that can do generic tasks. It works by an endless loop of `intent detection`, `function invocation`, `self-evaluation` and `reply generation` (if it decides to reply! :)). The agent is capable of planning complex tasks by invoking multiple functions, and remember things from the conversation.

In a nutshell, it goes like this:

- Decide based on the conversation history if it needs to take an action by using functions. It uses the LLM to detect the intent from the conversation.
- if it need to take an action (e.g. "remember something from the conversation" ) or generate complex tasks ( executing a chain of functions to achieve a goal ) it invokes the functions
- it re-evaluates if it needs to do any other action
- return the result back to the LLM to generate a reply for the user

Under the hood LocalAI converts functions to llama.cpp BNF grammars. While OpenAI fine-tuned a model to reply to functions, LocalAI constrains the LLM to follow grammars. This is a much more efficient way to do it, and it is also more flexible as you can define your own functions and grammars. For learning more about this, check out the [LocalAI documentation](https://localai.io/docs/llm) and my tweet that explains how it works under the hoods: https://twitter.com/mudler_it/status/1675524071457533953.

### Agent functions

The intention of this project is to keep the agent minimal, so can be built on top of it or forked. The agent is capable of doing the following functions:
- remember something from the conversation
- recall something from the conversation
- search something from the internet
- plan a complex task by invoking multiple functions
- write files to disk

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