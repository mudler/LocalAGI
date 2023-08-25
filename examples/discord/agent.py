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
from config import config

from queue import Queue
import asyncio
import threading
from localagi import LocalAGI
from loguru import logger
from ascii_magic import AsciiArt
from duckduckgo_search import DDGS
from typing import Dict, List
import os
from langchain.text_splitter import RecursiveCharacterTextSplitter
import discord
import openai
import urllib.request
from datetime import datetime


from chromadb.config import Settings
import json
import os
from io import StringIO 
FILE_NAME_FORMAT = '%Y_%m_%d_%H_%M_%S'

EMBEDDINGS_MODEL = config["agent"]["embeddings_model"]
EMBEDDINGS_API_BASE = config["agent"]["embeddings_api_base"]
PERSISTENT_DIR = config["agent"]["persistent_dir"]
MILVUS_HOST = config["agent"]["milvus_host"]
MILVUS_PORT = config["agent"]["milvus_port"]
DB_DIR =  config["agent"]["db_dir"]

if MILVUS_HOST == "":
    if not os.environ.get("PYSQL_HACK", "false") == "false":
        # these three lines swap the stdlib sqlite3 lib with the pysqlite3 package for chroma
        __import__('pysqlite3')
        import sys
        sys.modules['sqlite3'] = sys.modules.pop('pysqlite3')

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
    chunk_size = 1024
    chunk_overlap = 110
    logger.info(">>> ingesting: ")
    logger.info(q)
    documents = []
    sitemap_loader = SitemapLoader(web_path=q["url"])
    text_splitter = RecursiveCharacterTextSplitter(chunk_size=chunk_size, chunk_overlap=chunk_overlap)
    documents.extend(sitemap_loader.load())
    texts = text_splitter.split_documents(documents)
    if MILVUS_HOST == "":
        db = Chroma.from_documents(texts,embeddings,collection_name="memories", persist_directory=DB_DIR)
        db.persist()
        db = None
    else:
        Milvus.from_documents(texts,embeddings,collection_name="memories", connection_args={"host": MILVUS_HOST, "port": MILVUS_PORT})
    return f"Documents ingested"

def create_image(a, agent_actions={}, localagi=None):
    q = json.loads(a)
    logger.info(">>> creating image: ") 
    logger.info(q["caption"])
    size=f"{q['width']}x{q['height']}"
    response = openai.Image.create(prompt=q["caption"], n=1, size=size)
    image_url = response["data"][0]["url"]
    image_name = download_image(image_url)
    image_path = f"{PERSISTENT_DIR}{image_name}"

    file = discord.File(image_path, filename=image_name)
    embed = discord.Embed(title="Generated image")
    embed.set_image(url=f"attachment://{image_name}")

    call(channel.send(file=file, content=f"Here is what I have generated", embed=embed))

    return f"Image created: {response['data'][0]['url']}"

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
    logger.info(">>> saving to memories: ") 
    logger.info(q["content"])
    if MILVUS_HOST == "":
        chroma_client = Chroma(collection_name="memories",embedding_function=embeddings, persist_directory=DB_DIR)
    else:
        chroma_client = Milvus(collection_name="memories",embedding_function=embeddings, connection_args={"host": MILVUS_HOST, "port": MILVUS_PORT})
    chroma_client.add_texts([q["content"]],[{"id": str(uuid.uuid4())}])
    if MILVUS_HOST == "":
        chroma_client.persist()
        chroma_client = None
    return f"The object was saved permanently to memory."

def search_memory(query, agent_actions={}, localagi=None):
    q = json.loads(query)
    if MILVUS_HOST == "":
        chroma_client = Chroma(collection_name="memories",embedding_function=embeddings, persist_directory=DB_DIR)
    else:
        chroma_client = Milvus(collection_name="memories",embedding_function=embeddings, connection_args={"host": MILVUS_HOST, "port": MILVUS_PORT})
    docs = chroma_client.search(q["keywords"], "mmr")
    text_res="Memories found in the database:\n"
    for doc in docs:
        # drop newlines from page_content
        doc.page_content = " ".join(doc.page_content.replace.split())
        text_res+="- "+doc.page_content+"\n"
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
        logger.error(e)
        return []
    return formatted_results

## Search on duckduckgo
def search_duckduckgo(a, agent_actions={}, localagi=None):
    a = json.loads(a)
    list=ddg(a["query"], 2)

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
    "generate_picture": {
        "function": create_image,
        "plannable": True,
        "description": 'For creating a picture, the assistant replies with "generate_picture" and a detailed caption, enhancing it with as much detail as possible.',
        "signature": {
            "name": "generate_picture",
            "parameters": {
                "type": "object",
                "properties": {
                    "caption": {
                        "type": "string",
                    },
                    "width": {
                        "type": "number",
                    },
                    "height": {
                        "type": "number",
                    },
                },
            }
        },
    },
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