# Telegram World Cup Poll Bot

This is a basic Telegram bot written in Go for one or more Telegram group chats. It creates one poll per World Cup match, stores each user's pick, and builds a leaderboard with:

- correct guesses
- wrong guesses
- accuracy percentage

## What it does

- Fetches live World Cup fixtures from a public schedule page
- Uses an Armenia-time match window from 19:00 to 12:00 the next day
- Stores poll answers persistently in `data/state.json`
- Lets admins settle matches with final scores
- Calculates a leaderboard across all settled matches
- Optionally creates the day's polls automatically on a schedule

## Commands

- `/help`
- `/createpolls [YYYY-MM-DD]`
- `/guesses`
- `/matches [YYYY-MM-DD]`
- `/nextmatch`
- `/settlematch MATCH_ID HOME_SCORE AWAY_SCORE`
- `/leaderboard`

## Setup

1. Create a Telegram bot with [@BotFather](https://t.me/BotFather).
2. Copy `.env.example` to your own environment setup.
3. Add the bot to one or more group chats.
4. Disable privacy mode in BotFather if you want the bot to reliably see commands in groups.
5. Put one or more real group chat ids into `GROUP_CHAT_IDS`.

## Fixture Source

The bot fetches fixtures from:

- `FIXTURES_SOURCE_URL=https://en.wikipedia.org/wiki/2026_FIFA_World_Cup`

It parses the public match schedule markup and converts kickoff times into `Asia/Yerevan`.

For a requested date like `2026-06-18`, the bot treats the active match window as:

- start: `2026-06-18 19:00` Armenia time
- end: `2026-06-19 12:00` Armenia time

That same window is used by both `/matches` and `/createpolls`.

## Run

```bash
set -a
source .env
set +a
go run ./cmd/bot
```

## Oracle Deploy

The bot is designed to run as a single long-polling process. If you deploy it to a server, do not run the same bot token locally at the same time.

Recommended server layout:

- binary: `/opt/world-cup-poll-bot/bot`
- data dir: `/opt/world-cup-poll-bot/data`
- env file: `/etc/world-cup-poll-bot.env`
- systemd service: `world-cup-poll-bot`

Typical service management commands on the server:

```bash
sudo systemctl status world-cup-poll-bot
sudo systemctl restart world-cup-poll-bot
sudo systemctl stop world-cup-poll-bot
journalctl -u world-cup-poll-bot -n 100
journalctl -u world-cup-poll-bot -f
```

## Update On Oracle Server

Build the Linux binary on your Mac:

```bash
cd /Users/vahram/Desktop/telegram-bot
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bot-linux-amd64 ./cmd/bot
```

Copy the new binary to the Oracle server:

```bash
scp -i ~/.ssh/oci_worldcup_bot bot-linux-amd64 ubuntu@YOUR_SERVER_IP:/tmp/world-cup-poll-bot
```

Install the new binary on the server and restart the service:

```bash
ssh -i ~/.ssh/oci_worldcup_bot ubuntu@YOUR_SERVER_IP
sudo mv /tmp/world-cup-poll-bot /opt/world-cup-poll-bot/bot
sudo chown worldcup:worldcup /opt/world-cup-poll-bot/bot
sudo chmod 755 /opt/world-cup-poll-bot/bot
sudo systemctl restart world-cup-poll-bot
sudo systemctl status world-cup-poll-bot
journalctl -u world-cup-poll-bot -n 50
```

If the `.env` file changed, copy and install it too:

```bash
scp -i ~/.ssh/oci_worldcup_bot .env ubuntu@YOUR_SERVER_IP:/tmp/world-cup-poll-bot.env
ssh -i ~/.ssh/oci_worldcup_bot ubuntu@YOUR_SERVER_IP
sudo mv /tmp/world-cup-poll-bot.env /etc/world-cup-poll-bot.env
sudo chown root:worldcup /etc/world-cup-poll-bot.env
sudo chmod 640 /etc/world-cup-poll-bot.env
sudo systemctl restart world-cup-poll-bot
```

Recommended server env values:

```env
BOT_DATA_DIR=/opt/world-cup-poll-bot/data
BOT_TIMEZONE=Asia/Yerevan
```

## Notes

- Polls are non-anonymous so Telegram sends user vote updates to the bot.
- Only the latest answer for each user counts.
- Scheduled poll creation can target multiple chats through `GROUP_CHAT_IDS`.
- Leaderboards, upcoming guesses, and stored polls are scoped per Telegram chat.
- The current implementation fetches from a public World Cup page rather than a paid or key-based API.
- Because the source is a public web page, changes in page markup could require parser updates later.
