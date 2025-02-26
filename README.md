![](https://github.com/mudler/LocalAgent/assets/2420543/1f9a974e-3d57-45bd-9e80-709622a48964)

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