import openai
#from langchain.embeddings import HuggingFaceEmbeddings
from langchain.embeddings import LocalAIEmbeddings
import uuid
import requests
import sys
from loguru import logger
from ascii_magic import AsciiArt
from duckduckgo_search import DDGS
from typing import Dict, List, Optional

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
parser = argparse.ArgumentParser(description='μAGI')
parser.add_argument('--system-prompt', dest='system_prompt', action='store',
                    help='System prompt to use')
parser.add_argument('--prompt', dest='prompt', action='store', default=False,
                    help='Prompt mode')
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
args = parser.parse_args()


FUNCTIONS_MODEL = os.environ.get("FUNCTIONS_MODEL", "functions")
EMBEDDINGS_MODEL = os.environ.get("EMBEDDINGS_MODEL", "all-MiniLM-L6-v2")
LLM_MODEL = os.environ.get("LLM_MODEL", "gpt-4")
VOICE_MODEL= os.environ.get("TTS_MODEL","en-us-kathleen-low.onnx")
DEFAULT_SD_MODEL = os.environ.get("DEFAULT_SD_MODEL", "stablediffusion")
DEFAULT_SD_PROMPT = os.environ.get("DEFAULT_SD_PROMPT", "floating hair, portrait, ((loli)), ((one girl)), cute face, hidden hands, asymmetrical bangs, beautiful detailed eyes, eye shadow, hair ornament, ribbons, bowties, buttons, pleated skirt, (((masterpiece))), ((best quality)), colorful|((part of the head)), ((((mutated hands and fingers)))), deformed, blurry, bad anatomy, disfigured, poorly drawn face, mutation, mutated, extra limb, ugly, poorly drawn hands, missing limb, blurry, floating limbs, disconnected limbs, malformed hands, blur, out of focus, long neck, long body, Octane renderer, lowres, bad anatomy, bad hands, text")
PERSISTENT_DIR = os.environ.get("PERSISTENT_DIR", "/data")

## Constants
REPLY_ACTION = "reply"
PLAN_ACTION = "generate_plan"

embeddings = LocalAIEmbeddings(model=EMBEDDINGS_MODEL)
chroma_client = Chroma(collection_name="memories", persist_directory="db", embedding_function=embeddings)

# Function to create images with LocalAI
def display_avatar(input_text=DEFAULT_SD_PROMPT, model=DEFAULT_SD_MODEL):
    response = openai.Image.create(
        prompt=input_text,
        n=1,
        size="128x128",
        api_base=os.environ.get("OPENAI_API_BASE", "http://api:8080")+"/v1"
    )
    image_url = response['data'][0]['url']
    # convert the image to ascii art
    my_art = AsciiArt.from_url(image_url)
    my_art.to_terminal()

# Function to create audio with LocalAI
def tts(input_text, model=VOICE_MODEL):
    # strip newlines from text
    input_text = input_text.replace("\n", ".")
    # Create a temp file to store the audio output
    output_file_path = '/tmp/output.wav'
    # get from OPENAI_API_BASE env var
    url = os.environ.get("OPENAI_API_BASE", "http://api:8080") + '/tts'
    headers = {'Content-Type': 'application/json'}
    data = {
        "input": input_text,
        "model": model
    }

    response = requests.post(url, headers=headers, data=json.dumps(data))

    if response.status_code == 200:
        with open(output_file_path, 'wb') as f:
            f.write(response.content)
        logger.info('Audio file saved successfully:', output_file_path)
    else:
        logger.info('Request failed with status code', response.status_code)

    # Use aplay to play the audio
    os.system('aplay ' + output_file_path)
    # remove the audio file
    os.remove(output_file_path)

# Function to analyze the user input and pick the next action to do
def needs_to_do_action(user_input,agent_actions={}):

    # Get the descriptions and the actions name (the keys)
    descriptions=action_description("", agent_actions)

    messages = [
            {"role": "user",
             "content": f"""Transcript of AI assistant responding to user requests. Replies with the action to perform, including reasoning, and the confidence interval from 0 to 100.
{descriptions}"""},
            {"role": "user",
   "content": f"""{user_input}
Function call: """
             }
        ]
    functions = [
        {
        "name": "intent",
        "description": """Decide to do an action.""",
        "parameters": {
            "type": "object",
            "properties": {
            "confidence": {
                "type": "number",
                "description": "confidence of the action"
            },
            "action": {
                "type": "string",
                "enum": list(agent_actions.keys()),
                "description": "user intent"
            },
            "reasoning": {
                "type": "string",
                "description": "reasoning behind the intent"
            },
            },
            "required": ["action"]
        }
        },    
    ]
    response = openai.ChatCompletion.create(
        #model="gpt-3.5-turbo",
        model=FUNCTIONS_MODEL,
        messages=messages,
        request_timeout=1200,
        functions=functions,
        stop=None,
        temperature=0.1,
        #function_call="auto"
        function_call={"name": "intent"},
    )
    response_message = response["choices"][0]["message"]
    if response_message.get("function_call"):
        function_name = response.choices[0].message["function_call"].name
        function_parameters = response.choices[0].message["function_call"].arguments
        # read the json from the string
        res = json.loads(function_parameters)
        logger.info(">>> function name: "+function_name)
        logger.info(">>> function parameters: "+function_parameters)
        return res
    return {"action": REPLY_ACTION}

# This is used to collect the descriptions of the agent actions, used to populate the LLM prompt
def action_description(action, agent_actions):
    descriptions=""
    # generate descriptions of actions that the agent can pick
    for a in agent_actions:
        if ( action != "" and action == a ) or (action == ""):
            descriptions+=agent_actions[a]["description"]+"\n"
    return descriptions

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

### This function is used to process the functions given a user input.
### It picks a function, executes it and returns the list of messages containing the result.
def process_functions(user_input, action="", agent_actions={}):

    descriptions=action_description(action, agent_actions)

    messages = [
         #   {"role": "system", "content": "You are a helpful assistant."},
            {"role": "user",
             "content": f"""Transcript of AI assistant responding to user requests. Replies with the action to perform, including reasoning, and the confidence interval from 0 to 100.
{descriptions}"""},
            {"role": "user",
   "content": f"""{user_input}
Function call: """
             }
        ]
    response = function_completion(messages, action=action,agent_actions=agent_actions)
    response_message = response["choices"][0]["message"]
    response_result = ""
    function_result = {}
    if response_message.get("function_call"):
        function_name = response.choices[0].message["function_call"].name
        function_parameters = response.choices[0].message["function_call"].arguments
        logger.info("==> function name: ")
        logger.info(function_name)
        logger.info("==> function parameters: ")
        logger.info(function_parameters)
        function_to_call = agent_actions[function_name]["function"]

        function_result = function_to_call(function_parameters, agent_actions=agent_actions)
        logger.info("==> function result: ")
        logger.info(function_result)
        messages.append(
            {
                "role": "assistant",
                "content": None,
                "function_call": {"name": function_name, "arguments": function_parameters,},
            }
        )
        messages.append(
            {
                "role": "function",
                "name": function_name,
                "content": str(function_result)
            }
        )
    return messages, function_result

### function_completion is used to autocomplete functions given a list of messages
def function_completion(messages, action="", agent_actions={}):
    function_call = "auto"
    if action != "":
        function_call={"name": action}
    logger.info("==> function_call: ")
    logger.info(function_call)

    # get the functions from the signatures of the agent actions, if exists
    functions = []
    for action in agent_actions:
        if agent_actions[action].get("signature"):
            functions.append(agent_actions[action]["signature"])
    response = openai.ChatCompletion.create(
        #model="gpt-3.5-turbo",
        model=FUNCTIONS_MODEL,
        messages=messages,
        functions=functions,
        request_timeout=1200,
        stop=None,
        temperature=0.1,
        function_call=function_call
    )

    return response

# Rework the content of each message in the history in a way that is understandable by the LLM
# TODO: switch to templates (?)
def process_history(conversation_history):
    messages = ""
    for message in conversation_history:
        # if there is content append it
        if message.get("content") and message["role"] == "function":
            messages+="Function result: " + message["content"]+"\n"
        elif message.get("function_call"):
            # encode message["function_call" to json and appends it
            fcall = json.dumps(message["function_call"])
            messages+="Assistant calls function: " +fcall+"\n"
        elif message.get("content") and message["role"] == "user":
            messages+="User message: "+message["content"]+"\n"
        elif message.get("content") and message["role"] == "assistant":
            messages+="Assistant message: "+message["content"]+"\n"
    return messages

def clarify(responses, prefix=True):
    # TODO: this needs to be optimized
    if prefix:
        responses.append(
            {
                "role": "system",
                "content": "Return an appropriate answer to the user given the context above."
            }
        ) 

    response = openai.ChatCompletion.create(
        model=LLM_MODEL,
        messages=responses,
        stop=None,
        request_timeout=1200,
        temperature=0.1,
    )
    responses.append(
        {
            "role": "assistant",
            "content": response.choices[0].message["content"],
        }
    )
    return responses

### Fine tune a string before feeding into the LLM
def post_process(string):
    messages = [
        {
        "role": "user",
        "content": f"""Summarize the following text, keeping the relevant information:

```
{string}
```
""",
        }
    ]
    logger.info("==> Post processing: {string}", string=string)
    # get the response from the model
    response = openai.ChatCompletion.create(
        model=LLM_MODEL,
        messages=messages,
        stop=None,
        temperature=0.1,
        request_timeout=1200,
    )
    result = response["choices"][0]["message"]["content"]
    logger.info("==> Processed: {string}", string=result)
    return result

### Agent capabilities
### These functions are called by the agent to perform actions
###
def save(memory, agent_actions={}):
    q = json.loads(memory)
    logger.info(">>> saving to memories: ") 
    logger.info(q["thought"])
    chroma_client.add_texts([q["thought"]],[{"id": str(uuid.uuid4())}])
    chroma_client.persist()
    return f"The object was saved permanently to memory."

def search(query, agent_actions={}):
    q = json.loads(query)
    docs = chroma_client.similarity_search(q["reasoning"])
    text_res="Memories found in the database:\n"
    for doc in docs:
        text_res+="- "+doc.page_content+"\n"
    return text_res

# Python REPL
def python_repl(args, agent_actions={}):
    args = json.loads(args)
    try:
        # evaluate the code
        # redirect standard output and standard error to a string
        sys.stdout = sys.stderr = mystdout = StringIO()
        # eval the code
        r=eval(args["code"])
        # restore stdout and stderr
        sys.stdout = sys.__stdout__
        sys.stderr = sys.__stderr__
        # get the output
        output = mystdout.getvalue()
        # return the output
        return { "result": r, "output": output}
    except Exception as e:
        return str(e)

def calculate_plan(user_input, agent_actions={}):
    res = json.loads(user_input)
    logger.info("--> Calculating plan: {description}", description=res["description"])
    descriptions=action_description("",agent_actions)
    messages = [
            {"role": "user",
             "content": f"""Transcript of AI assistant responding to user requests. 
{descriptions}

Request: {res["description"]}

The assistant replies with a plan to answer the request with a list of subtasks with logical steps. The reasoning includes a self-contained, detailed and descriptive instruction to fullfill the task.

Function call: """
             }
        ]
    # get list of plannable actions
    plannable_actions = []
    for action in agent_actions:
        if agent_actions[action]["plannable"]:
            # append the key of the dict to plannable_actions
            plannable_actions.append(action)

    functions = [
        {
        "name": "plan",
        "description": """Decide to do an action.""",
        "parameters": {
            "type": "object",
            "properties": {
                "subtasks": {
                    "type": "array",
                    "items": {
                        "type": "object",
                        "properties": {
                            "reasoning": {
                                "type": "string",
                                "description": "subtask list",
                            },
                            "function": {
                                "type": "string",
                                "enum": plannable_actions,
                            },               
                        },
                    },
                },
            },
            "required": ["subtasks"]
        }
        },    
    ]
    response = openai.ChatCompletion.create(
        #model="gpt-3.5-turbo",
        model=FUNCTIONS_MODEL,
        messages=messages,
        functions=functions,
        stop=None,
        temperature=0.1,
        #function_call="auto"
        function_call={"name": "plan"},
    )
    response_message = response["choices"][0]["message"]
    if response_message.get("function_call"):
        function_name = response.choices[0].message["function_call"].name
        function_parameters = response.choices[0].message["function_call"].arguments
        # read the json from the string
        res = json.loads(function_parameters)
        logger.info("<<< function name: {function_name} >>>> parameters: {parameters}", function_name=function_name,parameters=function_parameters)
        return res
    return {"action": REPLY_ACTION}


# write file to disk with content
def write_file(arg, agent_actions={}):
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
def search_duckduckgo(args, agent_actions={}):
    args = json.loads(args)
    list=ddg(args["query"], 5)
    l = json.dumps(list)
    return l

### End Agent capabilities
###

### Main evaluate function
### This function evaluates the user input and the conversation history.
### It returns the conversation history with the latest response from the assistant.
def evaluate(user_input, conversation_history = [],re_evaluate=False, agent_actions={},re_evaluation_in_progress=False, postprocess=False, subtaskContext=False):

    messages = [
        {
        "role": "user",
        "content": user_input,
        }
    ]

    conversation_history.extend(messages)

    # pulling the old history make the context grow exponentially
    # and most importantly it repeates the first message with the commands again and again.
    # it needs a bit of cleanup and process the messages and piggyback more LocalAI functions templates
    # old_history = process_history(conversation_history)
    # action_picker_message = "Conversation history:\n"+old_history
    # action_picker_message += "\n"
    action_picker_message = "Request: "+user_input

    if re_evaluation_in_progress:
        action_picker_message+="\nRe-evaluation if another action is needed or we have completed the user request."
        action_picker_message+="\nReasoning: If no action is needed, I will use "+REPLY_ACTION+" to reply to the user."

    try:
        action = needs_to_do_action(action_picker_message,agent_actions=agent_actions)
    except Exception as e:
        logger.error("==> error: ")
        logger.error(e)
        action = {"action": REPLY_ACTION}

    if action["action"] != REPLY_ACTION:
        logger.info("==> μAGI wants to call '{action}'", action=action["action"])
        logger.info("==> Reasoning '{reasoning}'", reasoning=action["reasoning"])
        if action["action"] == PLAN_ACTION:
            logger.info("==> It's a plan <==: ")

        if postprocess:
            action["reasoning"] = post_process(action["reasoning"])
        
        #function_completion_message = "Conversation history:\n"+old_history+"\n"+
        function_completion_message = "Request: "+user_input+"\nReasoning: "+action["reasoning"]
        responses, function_results = process_functions(function_completion_message, action=action["action"], agent_actions=agent_actions)
        # if there are no subtasks, we can just reply,
        # otherwise we execute the subtasks
        # First we check if it's an object
        if isinstance(function_results, dict) and function_results.get("subtasks") and len(function_results["subtasks"]) > 0:
            # cycle subtasks and execute functions
            subtask_result=""
            for subtask in function_results["subtasks"]:
                logger.info("==> subtask: ")
                logger.info(subtask)
                #ctr="Context: "+user_input+"\nThought: "+action["reasoning"]+ "\nRequest: "+subtask["reasoning"]
                cr="Context: "+user_input+"\n"
                #cr=""
                if subtask_result != "" and subtaskContext:
                    # Include cumulative results of previous subtasks
                    # TODO: this grows context, maybe we should use a different approach or summarize
                    if postprocess:
                        cr+= "Subtask results: "+post_process(subtask_result)+"\n"
                    else:
                        cr+="Subtask results: "+subtask_result+"\n"
                
                if postprocess:
                    cr+= "Request: "+post_process(subtask["reasoning"])
                else:
                    cr+= "Request: "+subtask["reasoning"]
                subtask_response, function_results = process_functions(cr, subtask["function"],agent_actions=agent_actions)
                subtask_result+=process_history(subtask_response)
                if postprocess:
                    subtask_result=post_process(subtask_result)
                responses.extend(subtask_response)
        if re_evaluate:
            ## Better output or this infinite loops..
            logger.info("-> Re-evaluate if another action is needed")
            ## ? conversation history should go after the user_input maybe?
            re_eval = user_input +"\n"
            re_eval += "Conversation history: \n"
            if postprocess:
                re_eval+= post_process(process_history(responses[1:])) +"\n"
            else:
                re_eval+= process_history(responses[1:]) +"\n"
            responses = evaluate(re_eval, responses, re_evaluate,agent_actions=agent_actions,re_evaluation_in_progress=True)

        if re_evaluation_in_progress:
            conversation_history.extend(responses)
            return conversation_history       

        # TODO: this needs to be optimized
        responses = clarify(responses, prefix=True)

        # add responses to conversation history by extending the list
        conversation_history.extend(responses)
        # logger.info the latest response from the conversation history
        logger.info(conversation_history[-1]["content"])
        tts(conversation_history[-1]["content"])
    else:
        logger.info("==> no action needed")

        if re_evaluation_in_progress:
            logger.info("==> μAGI has completed the user request")
            logger.info("==> μAGI will reply to the user")
            return conversation_history        

        # TODO: this needs to be optimized
        # get the response from the model
        response = clarify(conversation_history, prefix=False)
        
        # add the response to the conversation history by extending the list
        conversation_history.append(response)
        # logger.info the latest response from the conversation history
        logger.info(conversation_history[-1]["content"])
        tts(conversation_history[-1]["content"])
    return conversation_history


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
    "write_file": {
        "function": write_file,
        "plannable": True,
        "description": 'The assistant replies with the action "write_file", the filename and content to save for writing a file to disk permanently. This can be used to store the result of complex actions locally.',
        "signature": {
            "name": "write_file",
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
                    "thought": {
                        "type": "string",
                        "description": "information to save"
                    },
                },
                "required": ["thought"]
            }
        },
    },
    "python": {
        "function": python_repl,
        "plannable": True,
        "description": 'The assistant replies with the action "python" to execute Python code.',
        "signature": {
            "name": "python",
            "description": """Search in memory""",
            "parameters": {
                "type": "object",
                "properties": {
                    "code": {
                        "type": "string",
                        "description": "reasoning behind the intent"
                    },
                    "reasoning": {
                        "type": "string",
                        "description": "reasoning behind the intent"
                    },
                },
                "required": ["code"]
            }
        }, 
    },
    "search_memory": {
        "function": search,
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
    PLAN_ACTION: {
        "function": calculate_plan,
        "plannable": False,
        "description": 'The assistant for solving complex tasks that involves more than one action or planning actions in sequence, replies with the action "'+PLAN_ACTION+'" and a detailed list of all the subtasks.',
        "signature": {
            "name": PLAN_ACTION,
            "description": """Plan complex tasks.""",
            "parameters": {
                "type": "object",
                "properties": {
                    "description": {
                        "type": "string",
                        "description": "reasoning behind the planning"
                    },
                },
                "required": ["description"]
            }
        },
    },
    REPLY_ACTION: {
        "function": None,
        "plannable": False,
        "description": 'For replying to the user, the assistant replies with the action "'+REPLY_ACTION+'" and the reply to the user directly when there is nothing to do.',
    },
}

conversation_history = []

# Set a system prompt if SYSTEM_PROMPT is set
if os.environ.get("SYSTEM_PROMPT") or args.system_prompt:
    sprompt = os.environ.get("SYSTEM_PROMPT", args.system_prompt)
    conversation_history.append({
        "role": "system",
        "content": sprompt
    })

logger.info("Welcome to μAGI")

# Skip avatar creation if --skip-avatar is set
if not args.skip_avatar:
    logger.info("Creating avatar, please wait...")
    display_avatar()

if not args.prompt:
    logger.info("μAGI has the following actions available at its disposal:")
    for action in agent_actions:
        logger.info("{action} - {description}", action=action, description=agent_actions[action]["description"])
else:
    logger.info(">>> Prompt mode <<<")
    logger.info(args.prompt)

# IF in prompt mode just evaluate, otherwise loop
if args.prompt:
    evaluate(
        args.prompt, 
        conversation_history, 
        re_evaluate=args.re_evaluate, 
        agent_actions=agent_actions,
        # Enable to lower context usage but increases LLM calls
        postprocess=args.postprocess,
        subtaskContext=args.subtaskContext,
        )
else:
    # TODO: process functions also considering the conversation history? conversation history + input
    while True:
        user_input = input("> ")
        # we are going to use the args to change the evaluation behavior
        conversation_history=evaluate(
            user_input, 
            conversation_history, 
            re_evaluate=args.re_evaluate, 
            agent_actions=agent_actions,
            # Enable to lower context usage but increases LLM calls
            postprocess=args.postprocess,
            subtaskContext=args.subtaskContext,
            )