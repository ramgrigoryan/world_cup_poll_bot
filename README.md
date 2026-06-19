# Telegram World Cup Poll Bot

This is a basic Telegram bot written in Go for one or more Telegram group chats. It creates one poll per World Cup match, stores each user's pick, and builds a leaderboard with:

- correct guesses
- wrong guesses
- accuracy percentage

## What it does

- Fetches live World Cup fixtures from a public schedule page
- Uses an Armenia-time match window from 20:00 to 08:00 the next day
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

- start: `2026-06-18 20:00` Armenia time
- end: `2026-06-19 08:00` Armenia time

That same window is used by both `/matches` and `/createpolls`.

## Run

```bash
set -a
source .env
set +a
go run ./cmd/bot
```

## Notes

- Polls are non-anonymous so Telegram sends user vote updates to the bot.
- Only the latest answer for each user counts.
- Scheduled poll creation can target multiple chats through `GROUP_CHAT_IDS`.
- Leaderboards, upcoming guesses, and stored polls are scoped per Telegram chat.
- The current implementation fetches from a public World Cup page rather than a paid or key-based API.
- Because the source is a public web page, changes in page markup could require parser updates later.
