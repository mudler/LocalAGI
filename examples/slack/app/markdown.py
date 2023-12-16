import re


# Conversion from Slack mrkdwn to OpenAI markdown
# See also: https://api.slack.com/reference/surfaces/formatting#basics
def slack_to_markdown(content: str) -> str:
    # Split the input string into parts based on code blocks and inline code
    parts = re.split(r"(```.+?```|`[^`\n]+?`)", content)

    # Apply the bold, italic, and strikethrough formatting to text not within code
    result = ""
    for part in parts:
        if part.startswith("```") or part.startswith("`"):
            result += part
        else:
            for o, n in [
                (r"\*(?!\s)([^\*\n]+?)(?<!\s)\*", r"**\1**"),  # *bold* to **bold**
                (r"_(?!\s)([^_\n]+?)(?<!\s)_", r"*\1*"),  # _italic_ to *italic*
                (r"~(?!\s)([^~\n]+?)(?<!\s)~", r"~~\1~~"),  # ~strike~ to ~~strike~~
            ]:
                part = re.sub(o, n, part)
            result += part
    return result


# Conversion from OpenAI markdown to Slack mrkdwn
# See also: https://api.slack.com/reference/surfaces/formatting#basics
def markdown_to_slack(content: str) -> str:
    # Split the input string into parts based on code blocks and inline code
    parts = re.split(r"(```.+?```|`[^`\n]+?`)", content)

    # Apply the bold, italic, and strikethrough formatting to text not within code
    result = ""
    for part in parts:
        if part.startswith("```") or part.startswith("`"):
            result += part
        else:
            for o, n in [
                (
                    r"\*\*\*(?!\s)([^\*\n]+?)(?<!\s)\*\*\*",
                    r"_*\1*_",
                ),  # ***bold italic*** to *_bold italic_*
                (
                    r"(?<![\*_])\*(?!\s)([^\*\n]+?)(?<!\s)\*(?![\*_])",
                    r"_\1_",
                ),  # *italic* to _italic_
                (r"\*\*(?!\s)([^\*\n]+?)(?<!\s)\*\*", r"*\1*"),  # **bold** to *bold*
                (r"__(?!\s)([^_\n]+?)(?<!\s)__", r"*\1*"),  # __bold__ to *bold*
                (r"~~(?!\s)([^~\n]+?)(?<!\s)~~", r"~\1~"),  # ~~strike~~ to ~strike~
            ]:
                part = re.sub(o, n, part)
            result += part
    return result
