import openai
#from langchain.embeddings import HuggingFaceEmbeddings
from langchain.embeddings import LocalAIEmbeddings
import uuid
import requests
from ascii_magic import AsciiArt

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

#embeddings = HuggingFaceEmbeddings(model_name="all-MiniLM-L6-v2")
embeddings = LocalAIEmbeddings(model="all-MiniLM-L6-v2")

chroma_client = Chroma(collection_name="memories", persist_directory="db", embedding_function=embeddings)


# Function to create images with OpenAI
def create_image(input_text=DEFAULT_SD_PROMPT, model=DEFAULT_SD_MODEL):
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

def calculate_plan(user_input):
    print("--> Calculating plan ")
    res = json.loads(user_input)
    print(res["description"])
    messages = [
            {"role": "user",
             "content": f"""Transcript of AI assistant responding to user requests. 
Replies with a plan to achieve the user's goal with a list of subtasks with logical steps.

Request: {res["description"]}
Function call: """
             }
        ]
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
                                "enum": ["save_memory", "search_memory"],
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
        print(">>> function name: "+function_name)
        print(">>> function parameters: "+function_parameters)
        return res
    return {"action": "none"}

def needs_to_do_action(user_input):
    messages = [
            {"role": "user",
             "content": f"""Transcript of AI assistant responding to user requests. Replies with the action to perform, including reasoning, and the confidence interval from 0 to 100.
For saving a memory, the assistant replies with the action "save_memory" and the string to save. 
For searching a memory, the assistant replies with the action "search_memory" and the query to search. 
For generating a plan for complex tasks, the assistant replies with the action "generate_plan" and a detailed plan to execute the user goal.
For replying to the user, the assistant replies with the action "reply" and the reply to the user.

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
                "enum": ["save_memory", "search_memory", "reply", "generate_plan"],
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
    return {"action": "reply"}

### Agent capabilities
def save(memory):
    print(">>> saving to memories: ") 
    print(memory)
    chroma_client.add_texts([memory],[{"id": str(uuid.uuid4())}])
    chroma_client.persist()
    return f"The object was saved permanently to memory."

def search(query):
    res = chroma_client.similarity_search(query)
    print(">>> query: ") 
    print(query)
    print(">>> retrieved memories: ") 
    print(res)
    return res

def process_functions(user_input, action=""):
    messages = [
         #   {"role": "system", "content": "You are a helpful assistant."},
            {"role": "user",
             "content": f"""Transcript of AI assistant responding to user requests.
For saving a memory, the assistant replies with the action "save_memory" and the string to save. 
For searching a memory, the assistant replies with the action "search_memory" and the query to search to find information stored previously. 
For replying to the user, the assistant replies with the action "reply" and the reply to the user directly when there is nothing to do.
For generating a plan for complex tasks, the assistant replies with the action "generate_plan" and a detailed list of all the subtasks needed to execute the user goal using the available actions.

Request: {user_input}
Function call: """
             }
        ]
    response = function_completion(messages, action=action)
    response_message = response["choices"][0]["message"]
    response_result = ""
    function_result = {}
    if response_message.get("function_call"):
        function_name = response.choices[0].message["function_call"].name
        function_parameters = response.choices[0].message["function_call"].arguments


        available_functions = {
            "save_memory": save,
            "generate_plan": calculate_plan,
            "search_memory": search,
        }

        function_to_call = available_functions[function_name]
        function_result = function_to_call(function_parameters)
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

def function_completion(messages, action=""):
    function_call = "auto"
    if action != "":
        function_call={"name": action}
    print("==> function_call: ")
    print(function_call)
    functions = [
        {
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
        {
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
        {
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
    ]
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


def evaluate(user_input, conversation_history = [],re_evaluate=False):
    try:
        action = needs_to_do_action(user_input) 
    except Exception as e:
        print("==> error: ")
        print(e)
        action = {"action": "reply"}

    if action["action"] != "reply":
        print("==> needs to do action: ")
        print(action)
        if action["action"] == "generate_plan":
            print("==> It's a plan <==: ")

        responses, function_results = process_functions(user_input+"\nReasoning: "+action["reasoning"], action=action["action"])
        # if there are no subtasks, we can just reply,
        # otherwise we execute the subtasks
        # First we check if it's an object
        if isinstance(function_results, dict) and len(function_results["subtasks"]) != 0:
            # cycle subtasks and execute functions
            for subtask in function_results["subtasks"]:
                print("==> subtask: ")
                print(subtask)
                subtask_response, function_results = process_functions(subtask["reasoning"], subtask["function"])
                responses.extend(subtask_response)
        if re_evaluate:
            all = process_history(responses)
            print("==> all: ")
            print(all)
            ## Better output or this infinite loops..
            print("-> Re-evaluate if another action is needed")
            responses = evaluate(user_input+process_history(responses), responses, re_evaluate)
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

#create_image()

conversation_history = []
# TODO: process functions also considering the conversation history? conversation history + input
while True:
    user_input = input("> ")
    conversation_history=evaluate(user_input, conversation_history, re_evaluate=True)