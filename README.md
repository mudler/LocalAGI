
![b644008916041](https://github.com/user-attachments/assets/304ad402-5ddc-441b-a4b9-55ff9eec72be)


Check also:

- [LocalAI](https://github.com/mudler/LocalAI)
- [LocalRAG](https://github.com/mudler/LocalRAG)

## Connectors

### Github (issues)

Create an user and a PAT token:

```json
{
	"token": "PAT_TOKEN",
    "repository": "testrepo",
    "owner": "ci-forks",
    "botUserName": "localai-bot"
}
```

### Discord

Follow the steps in: https://discordpy.readthedocs.io/en/stable/discord.html to create a discord bot.   

The token of the bot is in the "Bot" tab. Also enable " Message Content Intent " in the Bot tab!

```json
{
"token": "Bot DISCORDTOKENHERE",
"defaultChannel": "OPTIONALCHANNELINT"
}
```

### Slack

See slack.yaml

- Create a new App from a manifest (copy-paste from `slack.yaml`)
- Create Oauth token bot token from "OAuth & Permissions" -> "OAuth Tokens for Your Workspace"
- Create App level token (from "Basic Information" -> "App-Level Tokens" ( `scope connections:writeRoute authorizations:read` ))

In the UI, when configuring the connector:

```json
{
"botToken": "xoxb-...",
"appToken": "xapp-1-..."
}
```

### Telegram

Ask a token to @botfather

In the UI, when configuring the connector:

```json
{ "token": "botfathertoken" }
```
