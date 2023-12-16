import openai
#from langchain.embeddings import HuggingFaceEmbeddings
from langchain.embeddings import LocalAIEmbeddings

from langchain.document_loaders import (
    SitemapLoader,
   # GitHubIssuesLoader,
   # GitLoader,
)

import uuid
import sys

from app.env import *
from queue import Queue
import asyncio
import threading
from localagi import LocalAGI

from ascii_magic import AsciiArt
from duckduckgo_search import DDGS
from typing import Dict, List
import os
from langchain.text_splitter import RecursiveCharacterTextSplitter
import openai
import urllib.request
from datetime import datetime
import json
import os
from io import StringIO 
FILE_NAME_FORMAT = '%Y_%m_%d_%H_%M_%S'



if not os.environ.get("PYSQL_HACK", "false") == "false":
    # these three lines swap the stdlib sqlite3 lib with the pysqlite3 package for chroma
    __import__('pysqlite3')
    import sys
    sys.modules['sqlite3'] = sys.modules.pop('pysqlite3')
if MILVUS_HOST == "":
    from langchain.vectorstores import Chroma
else:
    from langchain.vectorstores import Milvus

embeddings = LocalAIEmbeddings(model=EMBEDDINGS_MODEL,openai_api_base=EMBEDDINGS_API_BASE)

loop = None
channel = None
def call(thing):
    return asyncio.run_coroutine_threadsafe(thing,loop).result()

def ingest(a, agent_actions={}, localagi=None):
    q = json.loads(a)
    chunk_size = MEMORY_CHUNK_SIZE
    chunk_overlap = MEMORY_CHUNK_OVERLAP
    print(">>> ingesting: ")
    print(q)
    documents = []
    sitemap_loader = SitemapLoader(web_path=q["url"])
    text_splitter = RecursiveCharacterTextSplitter(chunk_size=chunk_size, chunk_overlap=chunk_overlap)
    documents.extend(sitemap_loader.load())
    texts = text_splitter.split_documents(documents)
    if MILVUS_HOST == "":
        db = Chroma.from_documents(texts,embeddings,collection_name=MEMORY_COLLECTION, persist_directory=PERSISTENT_DIR)
        db.persist()
        db = None
    else:
        Milvus.from_documents(texts,embeddings,collection_name=MEMORY_COLLECTION, connection_args={"host": MILVUS_HOST, "port": MILVUS_PORT})
    return f"Documents ingested"
# def create_image(a, agent_actions={}, localagi=None):
#     """
#     Create an image based on a description using OpenAI's API.

#     Args:
#         a (str): A JSON string containing the description, width, and height for the image to be created.
#         agent_actions (dict, optional): A dictionary of agent actions. Defaults to {}.
#         localagi (LocalAGI, optional): An instance of the LocalAGI class. Defaults to None.

#     Returns:
#         str: A string containing the URL of the created image.
#     """
#     q = json.loads(a)
#     print(">>> creating image: ")
#     print(q["description"])
#     size=f"{q['width']}x{q['height']}"
#     response = openai.Image.create(prompt=q["description"], n=1, size=size)
#     image_url = response["data"][0]["url"]
#     image_name = download_image(image_url)
#     image_path = f"{PERSISTENT_DIR}{image_name}"

#     file = discord.File(image_path, filename=image_name)
#     embed = discord.Embed(title="Generated image")
#     embed.set_image(url=f"attachment://{image_name}")

#     call(channel.send(file=file, content=f"Here is what I have generated", embed=embed))

#     return f"Image created: {response['data'][0]['url']}"
def download_image(url: str):
    file_name = f"{datetime.now().strftime(FILE_NAME_FORMAT)}.jpg"
    full_path = f"{PERSISTENT_DIR}{file_name}"
    urllib.request.urlretrieve(url, full_path)
    return file_name


### Agent capabilities
### These functions are called by the agent to perform actions
###
def save(memory, agent_actions={}, localagi=None):
    q = json.loads(memory)
    print(">>> saving to memories: ") 
    print(q["content"])
    if MILVUS_HOST == "":
        chroma_client = Chroma(collection_name=MEMORY_COLLECTION,embedding_function=embeddings, persist_directory=PERSISTENT_DIR)
    else:
        chroma_client = Milvus(collection_name=MEMORY_COLLECTION,embedding_function=embeddings, connection_args={"host": MILVUS_HOST, "port": MILVUS_PORT})
    chroma_client.add_texts([q["content"]],[{"id": str(uuid.uuid4())}])
    if MILVUS_HOST == "":
        chroma_client.persist()
        chroma_client = None
    return f"The object was saved permanently to memory."

def search_memory(query, agent_actions={}, localagi=None):
    q = json.loads(query)
    if MILVUS_HOST == "":
        chroma_client = Chroma(collection_name=MEMORY_COLLECTION,embedding_function=embeddings, persist_directory=PERSISTENT_DIR)
    else:
        chroma_client = Milvus(collection_name=MEMORY_COLLECTION,embedding_function=embeddings, connection_args={"host": MILVUS_HOST, "port": MILVUS_PORT})
    #docs = chroma_client.search(q["keywords"], "mmr")
    retriever = chroma_client.as_retriever(search_type=MEMORY_SEARCH_TYPE, search_kwargs={"k": MEMORY_RESULTS})

    docs = retriever.get_relevant_documents(q["keywords"])
    text_res="Memories found in the database:\n"

    sources = set()  # To store unique sources
    
    # Collect unique sources
    for document in docs:
        if "source" in document.metadata:
            sources.add(document.metadata["source"])
    
    for doc in docs:
        # drop newlines from page_content
        content = doc.page_content.replace("\n", " ")
        content = " ".join(content.split())
        text_res+="- "+content+"\n"

    # Print the relevant sources used for the answer
    for source in sources:
        if source.startswith("http"):
            text_res += "" + source + "\n"

    chroma_client = None
    #if args.postprocess:
    #    return post_process(text_res)
    return text_res
    #return localagi.post_process(text_res)

# write file to disk with content
def save_file(arg, agent_actions={}, localagi=None):
    arg = json.loads(arg)
    file = filename = arg["filename"]
    content = arg["content"]
    # create persistent dir if does not exist
    if not os.path.exists(PERSISTENT_DIR):
        os.makedirs(PERSISTENT_DIR)
    # write the file in the directory specified
    file = os.path.join(PERSISTENT_DIR, filename)

    # Check if the file already exists
    if os.path.exists(file):
        mode = 'a'  # Append mode
    else:
        mode = 'w'  # Write mode

    with open(file, mode) as f:
        f.write(content)

    file = discord.File(file, filename=filename)
    call(channel.send(file=file, content=f"Here is what I have generated"))
    return f"File {file} saved successfully."

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
    ddgs = DDGS()
    try:
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
    except Exception as e:
        print(e)
        return []
    return formatted_results

## Search on duckduckgo
def search_duckduckgo(a, agent_actions={}, localagi=None):
    a = json.loads(a)
    list=ddg(a["query"], 2)

    text_res=""   
    for doc in list:
        text_res+=f"""{doc["link"]}: {doc["title"]} {doc["snippet"]}\n"""  
    print("Found")
    print(text_res)
    #if args.postprocess:
    #    return post_process(text_res)
    return text_res
    #l = json.dumps(list)
    #return l

### End Agent capabilities
###

### Agent action definitions
agent_actions = {
    # "generate_picture": {
    #     "function": create_image,
    #     "plannable": True,
    #     "description": 'For creating a picture, the assistant replies with "generate_picture" and a detailed description, enhancing it with as much detail as possible.',
    #     "signature": {
    #         "name": "generate_picture",
    #         "parameters": {
    #             "type": "object",
    #             "properties": {
    #                 "description": {
    #                     "type": "string",
    #                 },
    #                 "width": {
    #                     "type": "number",
    #                 },
    #                 "height": {
    #                     "type": "number",
    #                 },
    #             },
    #         }
    #     },
    # },
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
    "ingest": {
        "function": ingest,
        "plannable": True,
        "description": 'The assistant replies with the action "ingest" when there is an url to a sitemap to ingest memories from.',
        "signature": {
            "name": "ingest",
            "description": """Save or store informations into memory.""",
            "parameters": {
                "type": "object",
                "properties": {
                    "url": {
                        "type": "string",
                        "description": "information to save"
                    },
                },
                "required": ["url"]
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
                    "keywords": {
                        "type": "string",
                        "description": "reasoning behind the intent"
                    },
                },
                "required": ["keywords"]
            }
        }, 
    },
}



def localagi(q):
    localagi = LocalAGI(
        agent_actions=agent_actions,
        llm_model=LLM_MODEL,
        tts_model=VOICE_MODEL,
        tts_api_base=TTS_API_BASE,
        functions_model=FUNCTIONS_MODEL,
        api_base=LOCALAI_API_BASE,
        stablediffusion_api_base=IMAGE_API_BASE,
        stablediffusion_model=STABLEDIFFUSION_MODEL,
    )
    conversation_history = []

    conversation_history=localagi.evaluate(
        q, 
        conversation_history, 
        critic=False,
        re_evaluate=False, 
        # Enable to lower context usage but increases LLM calls
        postprocess=False,
        subtaskContext=True,
        )
    return conversation_history[-1]["content"]