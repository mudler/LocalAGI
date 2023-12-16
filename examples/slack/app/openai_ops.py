import threading
import time
import re
from typing import List, Dict, Any, Generator

import openai
from openai.error import Timeout
from openai.openai_object import OpenAIObject
import tiktoken

from slack_bolt import BoltContext
from slack_sdk.web import WebClient

from app.markdown import slack_to_markdown, markdown_to_slack
from app.slack_ops import update_wip_message

from app.agent import (
    localagi
)

# ----------------------------
# Internal functions
# ----------------------------

MAX_TOKENS = 1024
GPT_3_5_TURBO_0301_MODEL = "gpt-3.5-turbo-0301"


# Format message from Slack to send to OpenAI
def format_openai_message_content(content: str, translate_markdown: bool) -> str:
    if content is None:
        return None

    # Unescape &, < and >, since Slack replaces these with their HTML equivalents
    # See also: https://api.slack.com/reference/surfaces/formatting#escaping
    content = content.replace("&lt;", "<").replace("&gt;", ">").replace("&amp;", "&")

    # Convert from Slack mrkdwn to markdown format
    if translate_markdown:
        content = slack_to_markdown(content)

    return content


def ask_llm(
    *,
    messages: List[Dict[str, str]],
) -> str:
    # Remove old messages to make sure we have room for max_tokens
    # See also: https://platform.openai.com/docs/guides/chat/introduction
    # > total tokens must be below the model’s maximum limit (4096 tokens for gpt-3.5-turbo-0301)
    # TODO: currently we don't pass gpt-4 to this calculation method
    while calculate_num_tokens(messages) >= 4096 - MAX_TOKENS:
        removed = False
        for i, message in enumerate(messages):
            if message["role"] in ("user", "assistant"):
                del messages[i]
                removed = True
                break
        if not removed:
            # Fall through and let the OpenAI error handler deal with it
            break
    
    prompt=""
    
    for i, message in enumerate(messages):
        prompt += message["content"] + "\n"
    
    return localagi(prompt)

def consume_openai_stream_to_write_reply(
    *,
    client: WebClient,
    wip_reply: dict,
    context: BoltContext,
    user_id: str,
    messages: List[Dict[str, str]],
    steam: Generator[OpenAIObject, Any, None],
    timeout_seconds: int,
    translate_markdown: bool,
):
    start_time = time.time()
    assistant_reply: Dict[str, str] = {"role": "assistant", "content": ""}
    messages.append(assistant_reply)
    word_count = 0
    threads = []
    try:
        loading_character = " ... :writing_hand:"
        for chunk in steam:
            spent_seconds = time.time() - start_time
            if timeout_seconds < spent_seconds:
                raise Timeout()
            item = chunk.choices[0]
            if item.get("finish_reason") is not None:
                break
            delta = item.get("delta")
            if delta.get("content") is not None:
                word_count += 1
                assistant_reply["content"] += delta.get("content")
                if word_count >= 20:

                    def update_message():
                        assistant_reply_text = format_assistant_reply(
                            assistant_reply["content"], translate_markdown
                        )
                        wip_reply["message"]["text"] = assistant_reply_text
                        update_wip_message(
                            client=client,
                            channel=context.channel_id,
                            ts=wip_reply["message"]["ts"],
                            text=assistant_reply_text + loading_character,
                            messages=messages,
                            user=user_id,
                        )

                    thread = threading.Thread(target=update_message)
                    thread.daemon = True
                    thread.start()
                    threads.append(thread)
                    word_count = 0

        for t in threads:
            try:
                if t.is_alive():
                    t.join()
            except Exception:
                pass

        assistant_reply_text = format_assistant_reply(
            assistant_reply["content"], translate_markdown
        )
        wip_reply["message"]["text"] = assistant_reply_text
        update_wip_message(
            client=client,
            channel=context.channel_id,
            ts=wip_reply["message"]["ts"],
            text=assistant_reply_text,
            messages=messages,
            user=user_id,
        )
    finally:
        for t in threads:
            try:
                if t.is_alive():
                    t.join()
            except Exception:
                pass
        try:
            steam.close()
        except Exception:
            pass


def calculate_num_tokens(
    messages: List[Dict[str, str]],
    # TODO: adjustment for gpt-4
    model: str = GPT_3_5_TURBO_0301_MODEL,
) -> int:
    """Returns the number of tokens used by a list of messages."""
    try:
        encoding = tiktoken.encoding_for_model(model)
    except KeyError:
        encoding = tiktoken.get_encoding("cl100k_base")
    if model == GPT_3_5_TURBO_0301_MODEL:
        # note: future models may deviate from this
        num_tokens = 0
        for message in messages:
            # every message follows <im_start>{role/name}\n{content}<im_end>\n
            num_tokens += 4
            for key, value in message.items():
                num_tokens += len(encoding.encode(value))
                if key == "name":  # if there's a name, the role is omitted
                    num_tokens += -1  # role is always required and always 1 token
        num_tokens += 2  # every reply is primed with <im_start>assistant
        return num_tokens
    else:
        error = (
            f"Calculating the number of tokens for for model {model} is not yet supported. "
            "See https://github.com/openai/openai-python/blob/main/chatml.md "
            "for information on how messages are converted to tokens."
        )
        raise NotImplementedError(error)


# Format message from OpenAI to display in Slack
def format_assistant_reply(content: str, translate_markdown: bool) -> str:
    for o, n in [
        # Remove leading newlines
        ("^\n+", ""),
        # Remove prepended Slack user ID
        ("^<@U.*?>\\s?:\\s?", ""),
        # Remove OpenAI syntax tags since Slack doesn't render them in a message
        ("```\\s*[Rr]ust\n", "```\n"),
        ("```\\s*[Rr]uby\n", "```\n"),
        ("```\\s*[Ss]cala\n", "```\n"),
        ("```\\s*[Kk]otlin\n", "```\n"),
        ("```\\s*[Jj]ava\n", "```\n"),
        ("```\\s*[Gg]o\n", "```\n"),
        ("```\\s*[Ss]wift\n", "```\n"),
        ("```\\s*[Oo]objective[Cc]\n", "```\n"),
        ("```\\s*[Cc]\n", "```\n"),
        ("```\\s*[Cc][+][+]\n", "```\n"),
        ("```\\s*[Cc][Pp][Pp]\n", "```\n"),
        ("```\\s*[Cc]sharp\n", "```\n"),
        ("```\\s*[Mm]atlab\n", "```\n"),
        ("```\\s*[Jj][Ss][Oo][Nn]\n", "```\n"),
        ("```\\s*[Ll]a[Tt]e[Xx]\n", "```\n"),
        ("```\\s*bash\n", "```\n"),
        ("```\\s*zsh\n", "```\n"),
        ("```\\s*sh\n", "```\n"),
        ("```\\s*[Ss][Qq][Ll]\n", "```\n"),
        ("```\\s*[Pp][Hh][Pp]\n", "```\n"),
        ("```\\s*[Pp][Ee][Rr][Ll]\n", "```\n"),
        ("```\\s*[Jj]ava[Ss]cript", "```\n"),
        ("```\\s*[Ty]ype[Ss]cript", "```\n"),
        ("```\\s*[Pp]ython\n", "```\n"),
    ]:
        content = re.sub(o, n, content)

    # Convert from OpenAI markdown to Slack mrkdwn format
    if translate_markdown:
        content = markdown_to_slack(content)

    return content


def build_system_text(
    system_text_template: str, translate_markdown: bool, context: BoltContext
):
    system_text = system_text_template.format(bot_user_id=context.bot_user_id)
    # Translate format hint in system prompt
    if translate_markdown is True:
        system_text = slack_to_markdown(system_text)
    return system_text
