from typing import Optional

import openai
from slack_bolt import BoltContext

from .openai_ops import GPT_3_5_TURBO_0301_MODEL

# All the supported languages for Slack app as of March 2023
_locale_to_lang = {
    "en-US": "English",
    "en-GB": "English",
    "de-DE": "German",
    "es-ES": "Spanish",
    "es-LA": "Spanish",
    "fr-FR": "French",
    "it-IT": "Italian",
    "pt-BR": "Portuguese",
    "ru-RU": "Russian",
    "ja-JP": "Japanese",
    "zh-CN": "Chinese",
    "zh-TW": "Chinese",
    "ko-KR": "Korean",
}


def from_locale_to_lang(locale: Optional[str]) -> Optional[str]:
    if locale is None:
        return None
    return _locale_to_lang.get(locale)


_translation_result_cache = {}


def translate(*, openai_api_key: str, context: BoltContext, text: str) -> str:
    lang = from_locale_to_lang(context.get("locale"))
    if lang is None or lang == "English":
        return text

    cached_result = _translation_result_cache.get(f"{lang}:{text}")
    if cached_result is not None:
        return cached_result
    response = openai.ChatCompletion.create(
        api_key=openai_api_key,
        model=GPT_3_5_TURBO_0301_MODEL,
        messages=[
            {
                "role": "system",
                "content": "You're the AI model that primarily focuses on the quality of language translation. "
                "You must not change the meaning of sentences when translating them into a different language. "
                "You must provide direct translation result as much as possible. "
                "When the given text is a single verb/noun, its translated text must be a norm/verb form too. "
                "Slack's emoji (e.g., :hourglass_flowing_sand:) and mention parts must be kept as-is. "
                "Your response must not include any additional notes in English. "
                "Your response must omit English version / pronunciation guide for the result. ",
            },
            {
                "role": "user",
                "content": f"Can you translate {text} into {lang} in a professional tone? "
                "Please respond with the only the translated text in a format suitable for Slack user interface. "
                "No need to append any English notes and guides.",
            },
        ],
        top_p=1,
        n=1,
        max_tokens=1024,
        temperature=1,
        presence_penalty=0,
        frequency_penalty=0,
        logit_bias={},
        user="system",
    )
    translated_text = response["choices"][0]["message"].get("content")
    _translation_result_cache[f"{lang}:{text}"] = translated_text
    return translated_text
