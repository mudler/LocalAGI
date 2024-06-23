import openai
#from langchain.embeddings import HuggingFaceEmbeddings
from langchain.embeddings import LocalAIEmbeddings
import uuid
import sys

from localagi import LocalAGI
from loguru import logger
from ascii_magic import AsciiArt
from duckduckgo_search import DDGS
from typing import Dict, List
import os

# these three lines swap the stdlib sqlite3 lib with the pysqlite3 package for chroma
__import__('pysqlite3')
import sys
sys.modules['sqlite3'] = sys.modules.pop('pysqlite3')

from langchain.vectorstores import Chroma
from chromadb.config import Settings
import json
import os
from io import StringIO 

# Parse arguments such as system prompt and batch mode
import argparse
parser = argparse.ArgumentParser(description='LocalAGI')
# System prompt
parser.add_argument('--system-prompt', dest='system_prompt', action='store',
                    help='System prompt to use')
# Batch mode
parser.add_argument('--prompt', dest='prompt', action='store', default=False,
                    help='Prompt mode')
# Interactive mode
parser.add_argument('--interactive', dest='interactive', action='store_true', default=False,
                    help='Interactive mode. Can be used with --prompt to start an interactive session')
# skip avatar creation
parser.add_argument('--skip-avatar', dest='skip_avatar', action='store_true', default=False,
                    help='Skip avatar creation') 
# Reevaluate
parser.add_argument('--re-evaluate', dest='re_evaluate', action='store_true', default=False,
                    help='Reevaluate if another action is needed or we have completed the user request')
# Postprocess
parser.add_argument('--postprocess', dest='postprocess', action='store_true', default=False,
                    help='Postprocess the reasoning')
# Subtask context
parser.add_argument('--subtask-context', dest='subtaskContext', action='store_true', default=False,
                    help='Include context in subtasks')

# Search results number
parser.add_argument('--search-results', dest='search_results', type=int, action='store', default=2,
                    help='Number of search results to return')
# Plan message
parser.add_argument('--plan-message', dest='plan_message', action='store', 
                    help="What message to use during planning",
)                   

DEFAULT_PROMPT="floating hair, portrait, ((loli)), ((one girl)), cute face, hidden hands, asymmetrical bangs, beautiful detailed eyes, eye shadow, hair ornament, ribbons, bowties, buttons, pleated skirt, (((masterpiece))), ((best quality)), colorful|((part of the head)), ((((mutated hands and fingers)))), deformed, blurry, bad anatomy, disfigured, poorly drawn face, mutation, mutated, extra limb, ugly, poorly drawn hands, missing limb, blurry, floating limbs, disconnected limbs, malformed hands, blur, out of focus, long neck, long body, Octane renderer, lowres, bad anatomy, bad hands, text"
DEFAULT_API_BASE = os.environ.get("DEFAULT_API_BASE", "http://api:8080")
# TTS api base
parser.add_argument('--tts-api-base', dest='tts_api_base', action='store', default=DEFAULT_API_BASE,
                    help='TTS api base')
# LocalAI api base
parser.add_argument('--localai-api-base', dest='localai_api_base', action='store', default=DEFAULT_API_BASE,
                    help='LocalAI api base')
# Images api base
parser.add_argument('--images-api-base', dest='images_api_base', action='store', default=DEFAULT_API_BASE,
                    help='Images api base')
# Embeddings api base
parser.add_argument('--embeddings-api-base', dest='embeddings_api_base', action='store', default=DEFAULT_API_BASE,
                    help='Embeddings api base')
# Functions model
parser.add_argument('--functions-model', dest='functions_model', action='store', default="functions",
                    help='Functions model')
# Embeddings model
parser.add_argument('--embeddings-model', dest='embeddings_model', action='store', default="all-MiniLM-L6-v2",
                    help='Embeddings model')
# LLM model
parser.add_argument('--llm-model', dest='llm_model', action='store', default="gpt-4",
                    help='LLM model')
# Voice model
parser.add_argument('--tts-model', dest='tts_model', action='store', default="en-us-kathleen-low.onnx",
                    help='TTS model')
# Stable diffusion model
parser.add_argument('--stablediffusion-model', dest='stablediffusion_model', action='store', default="stablediffusion",
                    help='Stable diffusion model')
# Stable diffusion prompt
parser.add_argument('--stablediffusion-prompt', dest='stablediffusion_prompt', action='store', default=DEFAULT_PROMPT,
                    help='Stable diffusion prompt')
# Force action
parser.add_argument('--force-action', dest='force_action', action='store', default="",
                    help='Force an action')
# Debug mode
parser.add_argument('--debug', dest='debug', action='store_true', default=False,
                    help='Debug mode')
# Critic mode
parser.add_argument('--critic', dest='critic', action='store_true', default=False,
                    help='Enable critic')
# Parse arguments
args = parser.parse_args()

STABLEDIFFUSION_MODEL = os.environ.get("STABLEDIFFUSION_MODEL", args.stablediffusion_model)
STABLEDIFFUSION_PROMPT = os.environ.get("STABLEDIFFUSION_PROMPT", args.stablediffusion_prompt)
FUNCTIONS_MODEL = os.environ.get("FUNCTIONS_MODEL", args.functions_model)
EMBEDDINGS_MODEL = os.environ.get("EMBEDDINGS_MODEL", args.embeddings_model)
LLM_MODEL = os.environ.get("LLM_MODEL", args.llm_model)
VOICE_MODEL= os.environ.get("TTS_MODEL",args.tts_model)
STABLEDIFFUSION_MODEL = os.environ.get("STABLEDIFFUSION_MODEL",args.stablediffusion_model)
STABLEDIFFUSION_PROMPT = os.environ.get("STABLEDIFFUSION_PROMPT", args.stablediffusion_prompt)
PERSISTENT_DIR = os.environ.get("PERSISTENT_DIR", "/data")
SYSTEM_PROMPT = ""
if os.environ.get("SYSTEM_PROMPT") or args.system_prompt:
    SYSTEM_PROMPT = os.environ.get("SYSTEM_PROMPT", args.system_prompt)

LOCALAI_API_BASE = args.localai_api_base
TTS_API_BASE = args.tts_api_base
IMAGE_API_BASE = args.images_api_base
EMBEDDINGS_API_BASE = args.embeddings_api_base

# Set log level
LOG_LEVEL = "INFO"

def my_filter(record):
    return record["level"].no >= logger.level(LOG_LEVEL).no

logger.remove()
logger.add(sys.stderr, filter=my_filter)

if args.debug:
    LOG_LEVEL = "DEBUG"
logger.debug("Debug mode on")

FUNCTIONS_MODEL = os.environ.get("FUNCTIONS_MODEL", args.functions_model)
EMBEDDINGS_MODEL = os.environ.get("EMBEDDINGS_MODEL", args.embeddings_model)
LLM_MODEL = os.environ.get("LLM_MODEL", args.llm_model)
VOICE_MODEL= os.environ.get("TTS_MODEL",args.tts_model)
STABLEDIFFUSION_MODEL = os.environ.get("STABLEDIFFUSION_MODEL",args.stablediffusion_model)
STABLEDIFFUSION_PROMPT = os.environ.get("STABLEDIFFUSION_PROMPT", args.stablediffusion_prompt)
PERSISTENT_DIR = os.environ.get("PERSISTENT_DIR", "/data")
SYSTEM_PROMPT = ""
if os.environ.get("SYSTEM_PROMPT") or args.system_prompt:
    SYSTEM_PROMPT = os.environ.get("SYSTEM_PROMPT", args.system_prompt)

LOCALAI_API_BASE = args.localai_api_base
TTS_API_BASE = args.tts_api_base
IMAGE_API_BASE = args.images_api_base
EMBEDDINGS_API_BASE = args.embeddings_api_base

## Constants
REPLY_ACTION = "reply"
PLAN_ACTION = "plan"

embeddings = LocalAIEmbeddings(model=EMBEDDINGS_MODEL,openai_api_base=EMBEDDINGS_API_BASE)
chroma_client = Chroma(collection_name="memories", persist_directory="db", embedding_function=embeddings)

# Function to create images with LocalAI
def display_avatar(agi, input_text=STABLEDIFFUSION_PROMPT, model=STABLEDIFFUSION_MODEL):
    image_url = agi.get_avatar(input_text)
    # convert the image to ascii art
    my_art = AsciiArt.from_url(image_url)
    my_art.to_terminal()

## This function is called to ask the user if does agree on the action to take and execute
def ask_user_confirmation(action_name, action_parameters):
    logger.info("==> Ask user confirmation")
    logger.info("==> action_name: {action_name}", action_name=action_name)
    logger.info("==> action_parameters: {action_parameters}", action_parameters=action_parameters)
    # Ask via stdin
    logger.info("==> Do you want to execute the action? (y/n)")
    user_input = input()
    if user_input == "y":
        logger.info("==> Executing action")
        return True
    else:
        logger.info("==> Skipping action")
        return False

### Agent capabilities
### These functions are called by the agent to perform actions
###
def save(memory, agent_actions={}, localagi=None):
    q = json.loads(memory)
    logger.info(">>> saving to memories: ") 
    logger.info(q["content"])
    chroma_client.add_texts([q["content"]],[{"id": str(uuid.uuid4())}])
    chroma_client.persist()
    return f"The object was saved permanently to memory."

def search_memory(query, agent_actions={}, localagi=None):
    q = json.loads(query)
    docs = chroma_client.similarity_search(q["reasoning"])
    text_res="Memories found in the database:\n"
    for doc in docs:
        text_res+="- "+doc.page_content+"\n"

    #if args.postprocess:
    #    return post_process(text_res)
    #return text_res
    return localagi.post_process(text_res)


# write file to disk with content
def save_file(arg, agent_actions={}, localagi=None):
    arg = json.loads(arg)
    filename = arg["filename"]
    content = arg["content"]
    # create persistent dir if does not exist
    if not os.path.exists(PERSISTENT_DIR):
        os.makedirs(PERSISTENT_DIR)
    # write the file in the directory specified
    filename = os.path.join(PERSISTENT_DIR, filename)
    with open(filename, 'w') as f:
        f.write(content)
    return f"File {filename} saved successfully."


def ddg(query: str, num_results: int, backend: str = "api") -> List[Dict[str, str]]:
    """Run query through DuckDuckGo and return metadata.

    Args:
        query: The query to search for.
        num_results: The number of results to return.

    Returns:
        A list of dictionaries with the following keys:
            snippet - The description of the result.
            title - The title of the result.
            link - The link to the result.
    """

    with DDGS() as ddgs:
        results = ddgs.text(
            query,
            backend=backend,
        )
        if results is None:
            return [{"Result": "No good DuckDuckGo Search Result was found"}]

        def to_metadata(result: Dict) -> Dict[str, str]:
            if backend == "news":
                return {
                    "date": result["date"],
                    "title": result["title"],
                    "snippet": result["body"],
                    "source": result["source"],
                    "link": result["url"],
                }
            return {
                "snippet": result["body"],
                "title": result["title"],
                "link": result["href"],
            }

        formatted_results = []
        for i, res in enumerate(results, 1):
            if res is not None:
                formatted_results.append(to_metadata(res))
            if len(formatted_results) == num_results:
                break
    return formatted_results

## Search on duckduckgo
def search_duckduckgo(a, agent_actions={}, localagi=None):
    a = json.loads(a)
    list=ddg(a["query"], args.search_results)

    text_res=""   
    for doc in list:
        text_res+=f"""{doc["link"]}: {doc["title"]} {doc["snippet"]}\n"""  

    #if args.postprocess:
    #    return post_process(text_res)
    return text_res
    #l = json.dumps(list)
    #return l

### End Agent capabilities
###

### Agent action definitions
agent_actions = {
    "search_internet": {
        "function": search_duckduckgo,
        "plannable": True,
        "description": 'For searching the internet with a query, the assistant replies with the action "search_internet" and the query to search.',
        "signature": {
            "name": "search_internet",
            "description": """For searching internet.""",
            "parameters": {
                "type": "object",
                "properties": {
                    "query": {
                        "type": "string",
                        "description": "information to save"
                    },
                },
            }
        },
    },
    "save_file": {
        "function": save_file,
        "plannable": True,
        "description": 'The assistant replies with the action "save_file", the filename and content to save for writing a file to disk permanently. This can be used to store the result of complex actions locally.',
        "signature": {
            "name": "save_file",
            "description": """For saving a file to disk with content.""",
            "parameters": {
                "type": "object",
                "properties": {
                    "filename": {
                        "type": "string",
                        "description": "information to save"
                    },
                    "content": {
                        "type": "string",
                        "description": "information to save"
                    },
                },
            }
        },
    },
    "save_memory": {
        "function": save,
        "plannable": True,
        "description": 'The assistant replies with the action "save_memory" and the string to remember or store an information that thinks it is relevant permanently.',
        "signature": {
            "name": "save_memory",
            "description": """Save or store informations into memory.""",
            "parameters": {
                "type": "object",
                "properties": {
                    "content": {
                        "type": "string",
                        "description": "information to save"
                    },
                },
                "required": ["content"]
            }
        },
    },
    "search_memory": {
        "function": search_memory,
        "plannable": True,
        "description": 'The assistant replies with the action "search_memory" for searching between its memories with a query term.',
        "signature": {
            "name": "search_memory",
            "description": """Search in memory""",
            "parameters": {
                "type": "object",
                "properties": {
                    "reasoning": {
                        "type": "string",
                        "description": "reasoning behind the intent"
                    },
                },
                "required": ["reasoning"]
            }
        }, 
    },
}

if __name__ == "__main__":
    conversation_history = []

    # Create a LocalAGI instance
    logger.info("Creating LocalAGI instance")
    localagi = LocalAGI(
        agent_actions=agent_actions,
        llm_model=LLM_MODEL,
        tts_model=VOICE_MODEL,
        tts_api_base=TTS_API_BASE,
        functions_model=FUNCTIONS_MODEL,
        api_base=LOCALAI_API_BASE,
        stablediffusion_api_base=IMAGE_API_BASE,
        stablediffusion_model=STABLEDIFFUSION_MODEL,
        force_action=args.force_action,
        plan_message=args.plan_message,
    )

    # Set a system prompt if SYSTEM_PROMPT is set
    if SYSTEM_PROMPT != "":
        conversation_history.append({
            "role": "system",
            "content": SYSTEM_PROMPT
        })

    logger.info("Welcome to LocalAGI")

    # Skip avatar creation if --skip-avatar is set
    if not args.skip_avatar:
        logger.info("Creating avatar, please wait...")
        display_avatar(localagi)

    actions = ""
    for action in agent_actions:
        actions+=" '"+action+"'"
    logger.info("LocalAGI internally can do the following actions:{actions}", actions=actions)

    if not args.prompt:
        logger.info(">>> Interactive mode <<<")
    else:
        logger.info(">>> Prompt mode <<<")
        logger.info(args.prompt)

    # IF in prompt mode just evaluate, otherwise loop
    if args.prompt:
        conversation_history=localagi.evaluate(
            args.prompt, 
            conversation_history, 
            critic=args.critic,
            re_evaluate=args.re_evaluate, 
            # Enable to lower context usage but increases LLM calls
            postprocess=args.postprocess,
            subtaskContext=args.subtaskContext,
            )
        localagi.tts_play(conversation_history[-1]["content"])

    if not args.prompt or args.interactive:
        # TODO: process functions also considering the conversation history? conversation history + input
        logger.info(">>> Ready! What can I do for you? ( try with: plan a roadtrip to San Francisco ) <<<")

        while True:
            user_input = input(">>> ")
            # we are going to use the args to change the evaluation behavior
            conversation_history=localagi.evaluate(
                user_input, 
                conversation_history, 
                critic=args.critic,
                re_evaluate=args.re_evaluate, 
                # Enable to lower context usage but increases LLM calls
                postprocess=args.postprocess,
                subtaskContext=args.subtaskContext,
                )
            localagi.tts_play(conversation_history[-1]["content"])
