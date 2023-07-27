import openai
from langchain.embeddings import HuggingFaceEmbeddings
import uuid
from langchain.vectorstores import Chroma
from chromadb.config import Settings
import json
import os

FUNCTIONS_MODEL = os.environ.get("FUNCTIONS_MODEL", "functions")
LLM_MODEL = os.environ.get("LLM_MODEL", "gpt-4")

embeddings = HuggingFaceEmbeddings(model_name="all-MiniLM-L6-v2")

chroma_client = Chroma(collection_name="memories", persist_directory="db", embedding_function=embeddings)

def needs_to_do_action(user_input):
    messages = [
         #   {"role": "system", "content": "You are a helpful assistant."},
            {"role": "user",
             "content": f"""Transcript of AI assistant responding to user requests. Replies with the action to perform, including reasoning, and the confidence interval from 0 to 100.

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
                "enum": ["save_memory", "search_memory", "reply"],
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
        return res["action"]
    return "reply"

### Agent capabilities
def save(memory):
    print(">>> saving to memories: ") 
    print(memory)
    chroma_client.add_texts([memory],[{"id": str(uuid.uuid4())}])
    chroma_client.persist()
    return "saved to memory"

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

Request: {user_input}
Function call: """
             }
        ]
    response = get_completion(messages, action=action)
    response_message = response["choices"][0]["message"]
    response_result = ""
    if response_message.get("function_call"):
        function_name = response.choices[0].message["function_call"].name
        function_parameters = response.choices[0].message["function_call"].arguments
        function_result = ""


        available_functions = {
            "save_memory": save,
            "search_memory": search,
        }

        function_to_call = available_functions[function_name]
        function_result = function_to_call(function_parameters)
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
                "content": f'{{"result": {str(function_result)}}}'
            }
        )
        response = openai.ChatCompletion.create(
            model=LLM_MODEL,
            messages=messages,
            max_tokens=200,
            stop=None,
            temperature=0.5,
        )
        messages.append(
            {
                "role": "assistant",
                "content": response.choices[0].message["content"],
            }
        )
    return messages

def get_completion(messages, action=""):
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
            "string": {
                "type": "string",
                "description": "information to save"
            },
            },
            "required": ["string"]
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

conversation_history = []
while True:
    user_input = input("> ")
    action = needs_to_do_action(user_input) 
    if action != "reply":
        print("==> needs to do action: "+action)
        responses = process_functions(user_input, action=action)
        # add responses to conversation history by extending the list
        conversation_history.extend(responses)
        # print the latest response from the conversation history
        print(conversation_history[-1])
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
        print(conversation_history[-1])
       