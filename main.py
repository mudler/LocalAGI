import openai
#from langchain.embeddings import HuggingFaceEmbeddings
from langchain.embeddings import LocalAIEmbeddings
import uuid
import requests
import sys
from loguru import logger
from ascii_magic import AsciiArt

logger.add(sys.stderr, format="{time} {level} {message}", filter="miniAGI", level="INFO")
# these three lines swap the stdlib sqlite3 lib with the pysqlite3 package for chroma
__import__('pysqlite3')
import sys
sys.modules['sqlite3'] = sys.modules.pop('pysqlite3')

from langchain.vectorstores import Chroma
from chromadb.config import Settings
import json
import os

FUNCTIONS_MODEL = os.environ.get("FUNCTIONS_MODEL", "functions")
LLM_MODEL = os.environ.get("LLM_MODEL", "gpt-4")
VOICE_MODEL= os.environ.get("TTS_MODEL","en-us-kathleen-low.onnx")
DEFAULT_SD_MODEL = os.environ.get("DEFAULT_SD_MODEL", "stablediffusion")
DEFAULT_SD_PROMPT = os.environ.get("DEFAULT_SD_PROMPT", "floating hair, portrait, ((loli)), ((one girl)), cute face, hidden hands, asymmetrical bangs, beautiful detailed eyes, eye shadow, hair ornament, ribbons, bowties, buttons, pleated skirt, (((masterpiece))), ((best quality)), colorful|((part of the head)), ((((mutated hands and fingers)))), deformed, blurry, bad anatomy, disfigured, poorly drawn face, mutation, mutated, extra limb, ugly, poorly drawn hands, missing limb, blurry, floating limbs, disconnected limbs, malformed hands, blur, out of focus, long neck, long body, Octane renderer, lowres, bad anatomy, bad hands, text")

REPLY_ACTION = "reply"

#embeddings = HuggingFaceEmbeddings(model_name="all-MiniLM-L6-v2")
embeddings = LocalAIEmbeddings(model="all-MiniLM-L6-v2")

chroma_client = Chroma(collection_name="memories", persist_directory="db", embedding_function=embeddings)



# Function to create images with OpenAI
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
        print('Audio file saved successfully:', output_file_path)
    else:
        print('Request failed with status code', response.status_code)

    # Use aplay to play the audio
    os.system('aplay ' + output_file_path)
    # remove the audio file
    os.remove(output_file_path)

def needs_to_do_action(user_input,agent_actions={}):

    # Get the descriptions and the actions name (the keys)
    descriptions=""
    for action in agent_actions:
        descriptions+=agent_actions[action]["description"]+"\n"

    messages = [
            {"role": "user",
             "content": f"""Transcript of AI assistant responding to user requests. Replies with the action to perform, including reasoning, and the confidence interval from 0 to 100.
{descriptions}

Request: {user_input}
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
        functions=functions,
        max_tokens=200,
        stop=None,
        temperature=0.5,
        #function_call="auto"
        function_call={"name": "intent"},
    )
    response_message = response["choices"][0]["message"]
    if response_message.get("function_call"):
        function_name = response.choices[0].message["function_call"].name
        function_parameters = response.choices[0].message["function_call"].arguments
        # read the json from the string
        res = json.loads(function_parameters)
        print(">>> function name: "+function_name)
        print(">>> function parameters: "+function_parameters)
        return res
    return {"action": REPLY_ACTION}

def process_functions(user_input, action="", agent_actions={}):

    descriptions=""
    for a in agent_actions:
        descriptions+=agent_actions[a]["description"]+"\n"

    messages = [
         #   {"role": "system", "content": "You are a helpful assistant."},
            {"role": "user",
             "content": f"""Transcript of AI assistant responding to user requests.
{descriptions}

Request: {user_input}
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

        function_to_call = agent_actions[function_name]["function"]
        function_result = function_to_call(function_parameters, agent_actions=agent_actions)
        print("==> function result: ")
        print(function_result)
        messages = [
         #   {"role": "system", "content": "You are a helpful assistant."},
            {
             "role": "user",
             "content": user_input,
             }
        ]
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
                "content": f'{{"result": "{str(function_result)}"}}'
            }
        )
    return messages, function_result

def function_completion(messages, action="", agent_actions={}):
    function_call = "auto"
    if action != "":
        function_call={"name": action}
    print("==> function_call: ")
    print(function_call)

    # get the functions from the signatures of the agent actions, if exists
    functions = []
    for action in agent_actions:
        if agent_actions[action].get("signature"):
            functions.append(agent_actions[action]["signature"])
    print("==> available functions for the LLM: ")
    print(functions)
    print("==> messages LLM: ")
    print(messages)
    response = openai.ChatCompletion.create(
        #model="gpt-3.5-turbo",
        model=FUNCTIONS_MODEL,
        messages=messages,
        functions=functions,
        max_tokens=200,
        stop=None,
        temperature=0.1,
        function_call=function_call
    )

    return response

# Gets the content of each message in the history
def process_history(conversation_history):
    messages = ""
    for message in conversation_history:
        # if there is content append it
        if message.get("content"):
            messages+=message["content"]+"\n"
        if message.get("function_call"):
            # encode message["function_call" to json and appends it
            fcall = json.dumps(message["function_call"])
            messages+=fcall+"\n"
    return messages


def evaluate(user_input, conversation_history = [],re_evaluate=False, agent_actions={}):
    try:
        action = needs_to_do_action(user_input,agent_actions=agent_actions)
    except Exception as e:
        print("==> error: ")
        print(e)
        action = {"action": REPLY_ACTION}

    if action["action"] != REPLY_ACTION:
        print("==> needs to do action: ")
        print(action)
        if action["action"] == "generate_plan":
            print("==> It's a plan <==: ")

        responses, function_results = process_functions(user_input+"\nReasoning: "+action["reasoning"], action=action["action"], agent_actions=agent_actions)
        # if there are no subtasks, we can just reply,
        # otherwise we execute the subtasks
        # First we check if it's an object
        if isinstance(function_results, dict) and function_results.get("subtasks") and len(function_results["subtasks"]) > 0:
            # cycle subtasks and execute functions
            for subtask in function_results["subtasks"]:
                print("==> subtask: ")
                print(subtask)
                subtask_response, function_results = process_functions(subtask["reasoning"], subtask["function"],agent_actions=agent_actions)
                responses.extend(subtask_response)
        if re_evaluate:
            all = process_history(responses)
            print("==> all: ")
            print(all)
            ## Better output or this infinite loops..
            print("-> Re-evaluate if another action is needed")
            responses = evaluate(user_input+process_history(responses), responses, re_evaluate,agent_actions=agent_actions)
        response = openai.ChatCompletion.create(
            model=LLM_MODEL,
            messages=responses,
            max_tokens=200,
            stop=None,
            temperature=0.5,
        )
        responses.append(
            {
                "role": "assistant",
                "content": response.choices[0].message["content"],
            }
        )
        # add responses to conversation history by extending the list
        conversation_history.extend(responses)
        # print the latest response from the conversation history
        print(conversation_history[-1]["content"])
        tts(conversation_history[-1]["content"])
    else:
        print("==> no action needed")
        # construct the message and add it to the conversation history
        message = {"role": "user", "content": user_input}
        conversation_history.append(message)
        #conversation_history.append({ "role": "assistant", "content": "No action needed from my side."})

        # get the response from the model
        response = openai.ChatCompletion.create(
            model=LLM_MODEL,
            messages=conversation_history,
            max_tokens=200,
            stop=None,
            temperature=0.5,
        )
        # add the response to the conversation history by extending the list
        conversation_history.append({ "role": "assistant", "content": response.choices[0].message["content"]})
        # print the latest response from the conversation history
        print(conversation_history[-1]["content"])
        tts(conversation_history[-1]["content"])
    return conversation_history



### Agent capabilities

def save(memory, agent_actions={}):
    print(">>> saving to memories: ") 
    print(memory)
    chroma_client.add_texts([memory],[{"id": str(uuid.uuid4())}])
    chroma_client.persist()
    return f"The object was saved permanently to memory."

def search(query, agent_actions={}):
    res = chroma_client.similarity_search(query)
    print(">>> query: ") 
    print(query)
    print(">>> retrieved memories: ") 
    print(res)
    return res

def calculate_plan(user_input, agent_actions={}):
    res = json.loads(user_input)
    logger.info("--> Calculating plan: {description}", description=res["description"])
    messages = [
            {"role": "user",
             "content": f"""Transcript of AI assistant responding to user requests. 
Replies with a plan to achieve the user's goal with a list of subtasks with logical steps.

Request: {res["description"]}
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
        max_tokens=200,
        stop=None,
        temperature=0.5,
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
    with open(filename, 'w') as f:
        f.write(content)
    return f"File {filename} saved successfully."

## Search on duckduckgo
def search_duckduckgo(args, agent_actions={}):
    args = json.loads(args)
    url = "https://api.duckduckgo.com/?q="+args["query"]+"&format=json&pretty=1"
    response = requests.get(url)
    if response.status_code == 200:
        return response.json()
    else:
        return {"error": "No results found."}

### End Agent capabilities


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
        "description": 'For writing a file to disk with content, the assistant replies with the action "write_file" and the filename and content to save.',
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
        "description": 'For saving a memory, the assistant replies with the action "save_memory" and the string to save.',
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
    "search_memory": {
        "function": search,
        "plannable": True,
        "description": 'For searching a memory, the assistant replies with the action "search_memory" and the query to search to find information stored previously.',
        "signature": {
            "name": "search_memory",
            "description": """Search in memory""",
            "parameters": {
                "type": "object",
                "properties": {
                    "query": {
                        "type": "string",
                        "description": "The query to be used to search informations"
                    },
                    "reasoning": {
                        "type": "string",
                        "description": "reasoning behind the intent"
                    },
                },
                "required": ["query"]
            }
        }, 
    },
    "generate_plan": {
        "function": calculate_plan,
        "plannable": False,
        "description": 'For generating a plan for complex tasks, the assistant replies with the action "generate_plan" and a detailed list of all the subtasks needed to execute the user goal using the available actions.',
        "signature": {
            "name": "generate_plan",
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

display_avatar()

# TODO: process functions also considering the conversation history? conversation history + input
while True:
    user_input = input("> ")
    conversation_history=evaluate(user_input, conversation_history, re_evaluate=True, agent_actions=agent_actions)