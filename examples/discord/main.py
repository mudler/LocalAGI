"""
This is a discord bot for generating images using OpenAI's DALL-E

Author: Stefan Rial
YouTube: https://youtube.com/@StefanRial
GitHub: https://https://github.com/StefanRial/ClaudeBot
E-Mail: mail.stefanrial@gmail.com
"""

import discord
import openai
import urllib.request
import os
from datetime import datetime
from configparser import ConfigParser
from queue import Queue
import agent
from agent import agent_actions
from localagi import LocalAGI
import asyncio
import threading
from discord import app_commands
import functools
import typing

config_file = "config.ini"
config = ConfigParser(interpolation=None)
config.read(config_file)

SERVER_ID = config["discord"]["server_id"]
DISCORD_API_KEY = config["discord"][str("api_key")]
OPENAI_ORG = config["openai"][str("organization")]
OPENAI_API_KEY = config["openai"][str("api_key")]

FILE_PATH = config["settings"][str("file_path")]
FILE_NAME_FORMAT = config["settings"][str("file_name_format")]

SIZE_LARGE = "1024x1024"
SIZE_MEDIUM = "512x512"
SIZE_SMALL = "256x256"
SIZE_DEFAULT = config["settings"][str("default_size")]

GUILD = discord.Object(id=SERVER_ID)

if not os.path.isdir(FILE_PATH):
    os.mkdir(FILE_PATH)


class Client(discord.Client):
    def __init__(self, *, intents: discord.Intents):
        super().__init__(intents=intents)
        self.tree = app_commands.CommandTree(self)

    async def setup_hook(self):
        self.tree.copy_global_to(guild=GUILD)
        await self.tree.sync(guild=GUILD)


claude_intents = discord.Intents.default()
claude_intents.messages = True
claude_intents.message_content = True
client = Client(intents=claude_intents)

openai.organization = OPENAI_ORG
openai.api_key = OPENAI_API_KEY
openai.Model.list()


async def close_thread(thread: discord.Thread):
    await thread.edit(name="closed")
    await thread.send(
        embed=discord.Embed(
            description="**Thread closed** - Context limit reached, closing...",
            color=discord.Color.blue(),
        )
    )
    await thread.edit(archived=True, locked=True)

@client.event
async def on_ready():
    print(f"We have logged in as {client.user}")

def diff(history, processed):
    return [item for item in processed if item not in history]

def analyze_history(history, processed, callback, channel):
    diff_list = diff(history, processed)
    for item in diff_list:
        if item["role"] == "function":
            content = item["content"]
            # Function result
            callback(channel.send(f"⚙️ Processed: {content}"))
        if item["role"] == "assistant" and "function_call" in item:
            function_name = item["function_call"]["name"]
            function_parameters = item["function_call"]["arguments"]
            # Function call
            callback(channel.send(f"⚙️ Called: {function_name} with {function_parameters}"))

def run_localagi_thread_history(history, message, thread, loop):
   agent.channel = message.channel
   def call(thing):
        return asyncio.run_coroutine_threadsafe(thing,loop).result()
   sent_message = call(thread.send(f"⚙️ LocalAGI starts"))

   user = message.author
   def action_callback(name, parameters):
        call(sent_message.edit(content=f"⚙️ Calling function '{name}' with {parameters}"))
   def reasoning_callback(name, reasoning):
        call(sent_message.edit(content=f"🤔 I'm thinking... '{reasoning}' (calling '{name}'), please wait.."))

   localagi = LocalAGI(
        agent_actions=agent_actions,
        llm_model=config["agent"]["llm_model"],
        tts_model=config["agent"]["tts_model"],
        action_callback=action_callback,
        reasoning_callback=reasoning_callback,     
        tts_api_base=config["agent"]["tts_api_base"],
        functions_model=config["agent"]["functions_model"],
        api_base=config["agent"]["api_base"],
        stablediffusion_api_base=config["agent"]["stablediffusion_api_base"],
        stablediffusion_model=config["agent"]["stablediffusion_model"],
    )
   # remove bot ID from the message content
   message.content = message.content.replace(f"<@{client.user.id}>", "")
   conversation_history = localagi.evaluate(
                message.content, 
                history, 
                subtaskContext=True,
        )
   
   analyze_history(history, conversation_history, call, thread)
   call(sent_message.edit(content=f"<@{user.id}> {conversation_history[-1]['content']}"))

def run_localagi_message(message, loop):
   agent.channel = message.channel
   def call(thing):
        return asyncio.run_coroutine_threadsafe(thing,loop).result()
   sent_message = call(message.channel.send(f"⚙️ LocalAGI starts"))

   user = message.author
   def action_callback(name, parameters):
        call(sent_message.edit(content=f"⚙️ Calling function '{name}' with {parameters}"))
   def reasoning_callback(name, reasoning):
        call(sent_message.edit(content=f"🤔 I'm thinking... '{reasoning}' (calling '{name}'), please wait.."))

   localagi = LocalAGI(
        agent_actions=agent_actions,
        llm_model=config["agent"]["llm_model"],
        tts_model=config["agent"]["tts_model"],
        action_callback=action_callback,
        reasoning_callback=reasoning_callback,     
        tts_api_base=config["agent"]["tts_api_base"],
        functions_model=config["agent"]["functions_model"],
        api_base=config["agent"]["api_base"],
        stablediffusion_api_base=config["agent"]["stablediffusion_api_base"],
        stablediffusion_model=config["agent"]["stablediffusion_model"],
    )
   # remove bot ID from the message content
   message.content = message.content.replace(f"<@{client.user.id}>", "")

   conversation_history = localagi.evaluate(
                message.content, 
                [], 
                subtaskContext=True,
        )
   analyze_history([], conversation_history, call, message.channel)
   call(sent_message.edit(content=f"<@{user.id}> {conversation_history[-1]['content']}"))

def run_localagi(interaction, prompt, loop):
    agent.channel = interaction.channel

    def call(thing):
        return asyncio.run_coroutine_threadsafe(thing,loop).result()
    
    user = interaction.user
    embed = discord.Embed(
        description=f"<@{user.id}> wants to chat! 🤖💬",
        color=discord.Color.green(),
    )
    embed.add_field(name=user.name, value=prompt)

    call(interaction.response.send_message(embed=embed))
    response = call(interaction.original_response())

    # create the thread
    thread = call(response.create_thread(
        name=prompt,
        slowmode_delay=1,
        reason="gpt-bot",
        auto_archive_duration=60,
    ))
    thread.typing()

    sent_message = call(thread.send(f"⚙️ LocalAGI starts"))
    messages = []
    def action_callback(name, parameters):
        call(sent_message.edit(content=f"⚙️ Calling function '{name}' with {parameters}"))
    def reasoning_callback(name, reasoning):
        call(sent_message.edit(content=f"🤔 I'm thinking... '{reasoning}' (calling '{name}'), please wait.."))

    localagi = LocalAGI(
        agent_actions=agent_actions,
        llm_model=config["agent"]["llm_model"],
        tts_model=config["agent"]["tts_model"],
        action_callback=action_callback,
        reasoning_callback=reasoning_callback,     
        tts_api_base=config["agent"]["tts_api_base"],
        functions_model=config["agent"]["functions_model"],
        api_base=config["agent"]["api_base"],
        stablediffusion_api_base=config["agent"]["stablediffusion_api_base"],
        stablediffusion_model=config["agent"]["stablediffusion_model"],
    )
    # remove bot ID from the message content
    prompt = prompt.replace(f"<@{client.user.id}>", "")

    conversation_history = localagi.evaluate(
                prompt, 
                messages, 
                subtaskContext=True,
        )
    analyze_history(messages, conversation_history, call, interaction.channel)
    call(sent_message.edit(content=f"<@{user.id}> {conversation_history[-1]['content']}"))

@client.tree.command()
@app_commands.describe(prompt="Ask me anything!")
async def localai(interaction: discord.Interaction, prompt: str):
    loop = asyncio.get_running_loop()
    threading.Thread(target=run_localagi, args=[interaction, prompt,loop]).start()    

# https://github.com/openai/gpt-discord-bot/blob/1161634a59c6fb642e58edb4f4fa1a46d2883d3b/src/utils.py#L15
def discord_message_to_message(message):
    if (
        message.type == discord.MessageType.thread_starter_message
        and message.reference.cached_message
        and len(message.reference.cached_message.embeds) > 0
        and len(message.reference.cached_message.embeds[0].fields) > 0
    ):
        field = message.reference.cached_message.embeds[0].fields[0]
        if field.value:
            return { "role": "user", "content": field.value }
    else:
        if message.content:
            return { "role": "user", "content": message.content }
    return None

@client.event
async def on_ready():
    loop = asyncio.get_running_loop() 
    agent.loop = loop

@client.event
async def on_message(message):
    # ignore messages from the bot
    if message.author == client.user:
        return
    loop = asyncio.get_running_loop() 
    # ignore messages not in a thread
    channel = message.channel
    if not isinstance(channel, discord.Thread) and client.user.mentioned_in(message):
        threading.Thread(target=run_localagi_message, args=[message,loop]).start()    
        return
    if not isinstance(channel, discord.Thread):
        return
    # ignore threads not created by the bot
    thread = channel
    if thread.owner_id != client.user.id:
        return
    
    if thread.message_count > 5:
        # too many messages, no longer going to reply
        await close_thread(thread=thread)
        return
    
    channel_messages = [
        discord_message_to_message(message)
        async for message in thread.history(limit=5)
    ]
    channel_messages = [x for x in channel_messages if x is not None]
    channel_messages.reverse()
    threading.Thread(target=run_localagi_thread_history, args=[channel_messages[:-1],message,thread,loop]).start()    

client.run(DISCORD_API_KEY)
