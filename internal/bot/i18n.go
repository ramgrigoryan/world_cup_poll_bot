package bot

import (
	"fmt"
	"strings"
	"time"
)

type locale string

const (
	localeEN locale = "en"
	localeRU locale = "ru"
	localeHY locale = "hy"
)

func detectLocale(user *User) locale {
	if user == nil {
		return localeEN
	}
	code := strings.ToLower(strings.TrimSpace(user.LanguageCode))
	switch {
	case strings.HasPrefix(code, "ru"):
		return localeRU
	case strings.HasPrefix(code, "hy"), strings.HasPrefix(code, "am"):
		return localeHY
	default:
		return localeEN
	}
}

func languageCodeForCommands(loc locale) string {
	switch loc {
	case localeRU:
		return "ru"
	case localeHY:
		return "hy"
	default:
		return "en"
	}
}

func commandsForLocale(loc locale) []BotCommand {
	switch loc {
	case localeRU:
		return []BotCommand{
			{Command: "help", Description: "Показать команды"},
			{Command: "chatid", Description: "Показать ID чата"},
			{Command: "lang", Description: "Установить язык бота: en, ru, hy"},
			{Command: "createpolls", Description: "Создать опросы на окно 20:00-08:00"},
			{Command: "guesses", Description: "Показать прогнозы"},
			{Command: "matches", Description: "Показать матчи на окно 20:00-08:00"},
			{Command: "nextmatch", Description: "Показать следующий матч"},
			{Command: "settlematch", Description: "Завершить матч"},
			{Command: "leaderboard", Description: "Показать таблицу лидеров"},
		}
	case localeHY:
		return []BotCommand{
			{Command: "help", Description: "Ցույց տալ հրամանները"},
			{Command: "chatid", Description: "Ցույց տալ չատի ID-ն"},
			{Command: "lang", Description: "Դնել բոտի լեզուն՝ en, ru, hy"},
			{Command: "createpolls", Description: "Ստեղծել հարցումներ 20:00-08:00 պատուհանի համար"},
			{Command: "guesses", Description: "Ցույց տալ կանխատեսումները"},
			{Command: "matches", Description: "Ցույց տալ խաղերը 20:00-08:00 պատուհանի համար"},
			{Command: "nextmatch", Description: "Ցույց տալ հաջորդ խաղը"},
			{Command: "settlematch", Description: "Փակել խաղը"},
			{Command: "leaderboard", Description: "Ցույց տալ առաջատարներին"},
		}
	default:
		return []BotCommand{
			{Command: "help", Description: "Show commands"},
			{Command: "chatid", Description: "Show chat ID"},
			{Command: "lang", Description: "Set bot language: en, ru, hy"},
			{Command: "createpolls", Description: "Create polls for the 20:00-08:00 window"},
			{Command: "guesses", Description: "Show guesses"},
			{Command: "matches", Description: "Show matches for the 20:00-08:00 window"},
			{Command: "nextmatch", Description: "Show the next match"},
			{Command: "settlematch", Description: "Settle a match"},
			{Command: "leaderboard", Description: "Show the leaderboard"},
		}
	}
}

func helpText(loc locale) string {
	switch loc {
	case localeRU:
		return strings.Join([]string{
			"Команды:",
			"/chatid - показать ID текущего чата",
			"/createpolls [YYYY-MM-DD] - создать опросы на окно 20:00-08:00 по Армении",
			"/guesses - показать прогнозы на ближайшие матчи",
			"/matches [YYYY-MM-DD] - показать матчи на окно 20:00-08:00 по Армении",
			"/nextmatch - показать ближайший матч",
			"/settlematch MATCH_ID HOME_SCORE AWAY_SCORE - завершить матч",
			"/leaderboard - показать таблицу лидеров",
		}, "\n")
	case localeHY:
		return strings.Join([]string{
			"Հրամաններ՝",
			"/chatid - ցույց տալ այս չատի ID-ն",
			"/createpolls [YYYY-MM-DD] - ստեղծել հարցումներ Երևանի 20:00-08:00 պատուհանի համար",
			"/guesses - ցույց տալ մոտակա խաղերի կանխատեսումները",
			"/matches [YYYY-MM-DD] - ցույց տալ Երևանի 20:00-08:00 խաղերը",
			"/nextmatch - ցույց տալ հաջորդ խաղը",
			"/settlematch MATCH_ID HOME_SCORE AWAY_SCORE - փակել խաղը",
			"/leaderboard - ցույց տալ առաջատարների աղյուսակը",
		}, "\n")
	default:
		return strings.Join([]string{
			"Commands:",
			"/chatid - show the current chat ID",
			"/lang [en|ru|hy] - set the bot language for this chat",
			"/createpolls [YYYY-MM-DD] - create polls for the Armenia window 20:00 to next-day 08:00",
			"/guesses - show stored guesses for upcoming matches",
			"/matches [YYYY-MM-DD] - list live matches for the Armenia window 20:00 to next-day 08:00",
			"/nextmatch - show the next upcoming World Cup match",
			"/settlematch MATCH_ID HOME_SCORE AWAY_SCORE - settle one match",
			"/leaderboard - show standings",
		}, "\n")
	}
}

func normalizeLocaleCode(raw string) (locale, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "en":
		return localeEN, true
	case "ru":
		return localeRU, true
	case "hy", "am":
		return localeHY, true
	default:
		return "", false
	}
}

func localeLabel(loc locale) string {
	switch loc {
	case localeRU:
		return "ru"
	case localeHY:
		return "hy"
	default:
		return "en"
	}
}

func textLanguageUsage(loc locale) string {
	switch loc {
	case localeRU:
		return "Используйте /lang en, /lang ru или /lang hy"
	case localeHY:
		return "Օգտագործիր /lang en, /lang ru կամ /lang hy"
	default:
		return "Use /lang en, /lang ru, or /lang hy"
	}
}

func textLanguageSet(loc locale, selected locale) string {
	switch loc {
	case localeRU:
		return "Язык бота установлен: " + localeLabel(selected)
	case localeHY:
		return "Բոտի լեզուն դրվեց՝ " + localeLabel(selected)
	default:
		return "Bot language set to: " + localeLabel(selected)
	}
}

func textThisChatID(loc locale, chatID int64) string {
	switch loc {
	case localeRU:
		return fmt.Sprintf("ID этого чата: %d", chatID)
	case localeHY:
		return fmt.Sprintf("Այս չատի ID-ն է՝ %d", chatID)
	default:
		return fmt.Sprintf("This chat ID is: %d", chatID)
	}
}

func textVerifyAdminError(loc locale) string {
	switch loc {
	case localeRU:
		return "Сейчас не удалось проверить права администратора."
	case localeHY:
		return "Հիմա չհաջողվեց ստուգել ադմինի իրավունքը։"
	default:
		return "Could not verify admin rights right now."
	}
}

func textOnlyAdminsCanCreate(loc locale) string {
	switch loc {
	case localeRU:
		return "Только администраторы могут создавать опросы."
	case localeHY:
		return "Միայն ադմինները կարող են ստեղծել հարցումներ։"
	default:
		return "Only admins can create polls."
	}
}

func textOnlyAdminsCanSettle(loc locale) string {
	switch loc {
	case localeRU:
		return "Только администраторы могут завершать матчи."
	case localeHY:
		return "Միայն ադմինները կարող են փակել խաղերը։"
	default:
		return "Only admins can settle matches."
	}
}

func textUseDateForCreate(loc locale) string {
	switch loc {
	case localeRU:
		return "Используйте /createpolls YYYY-MM-DD"
	case localeHY:
		return "Օգտագործիր /createpolls YYYY-MM-DD"
	default:
		return "Use /createpolls YYYY-MM-DD"
	}
}

func textUseDateForMatches(loc locale) string {
	switch loc {
	case localeRU:
		return "Используйте /matches YYYY-MM-DD"
	case localeHY:
		return "Օգտագործիր /matches YYYY-MM-DD"
	default:
		return "Use /matches YYYY-MM-DD"
	}
}

func textCouldNotCreatePolls(loc locale, err error) string {
	switch loc {
	case localeRU:
		return "Не удалось создать опросы: " + err.Error()
	case localeHY:
		return "Չհաջողվեց ստեղծել հարցումները․ " + err.Error()
	default:
		return "Could not create polls: " + err.Error()
	}
}

func textCouldNotLoadMatches(loc locale, err error) string {
	switch loc {
	case localeRU:
		return "Не удалось загрузить матчи: " + err.Error()
	case localeHY:
		return "Չհաջողվեց բեռնել խաղերը․ " + err.Error()
	default:
		return "Could not load matches: " + err.Error()
	}
}

func textCouldNotLoadNextMatch(loc locale, err error) string {
	switch loc {
	case localeRU:
		return "Не удалось загрузить следующий матч: " + err.Error()
	case localeHY:
		return "Չհաջողվեց բեռնել հաջորդ խաղը․ " + err.Error()
	default:
		return "Could not load next match: " + err.Error()
	}
}

func textLateVoteRejected(loc locale, username, homeTeam, awayTeam string) string {
	switch loc {
	case localeRU:
		return fmt.Sprintf("%s, уже поздно голосовать или менять прогноз на матч %s vs %s. Матч уже начался.", username, homeTeam, awayTeam)
	case localeHY:
		return fmt.Sprintf("%s, այլևս չես կարող քվեարկել կամ փոխել կանխատեսումը %s vs %s խաղի համար։ Խաղն արդեն սկսվել է։", username, homeTeam, awayTeam)
	default:
		return fmt.Sprintf("%s, voting or revoting for %s vs %s is not allowed because the match has already started.", username, homeTeam, awayTeam)
	}
}

func textNoFixtures(loc locale, start, end time.Time) string {
	switch loc {
	case localeRU:
		return fmt.Sprintf("Не найдено матчей для окна по Армении %s - %s", start.Format("2006-01-02 15:04"), end.Format("2006-01-02 15:04"))
	case localeHY:
		return fmt.Sprintf("Երևանի %s - %s պատուհանի համար խաղեր չեն գտնվել", start.Format("2006-01-02 15:04"), end.Format("2006-01-02 15:04"))
	default:
		return fmt.Sprintf("No fixtures found for the Armenia window %s to %s", start.Format("2006-01-02 15:04"), end.Format("2006-01-02 15:04"))
	}
}

func textPollsAlreadyCreated(loc locale, link string) string {
	switch loc {
	case localeRU:
		return "Опросы уже созданы для этого окна. Перейти: " + link
	case localeHY:
		return "Այս պատուհանի հարցումներն արդեն ստեղծված են։ Անցիր այստեղ՝ " + link
	default:
		return "Polls were already created for this window. Jump here: " + link
	}
}

func textCreatedPolls(loc locale, start, end time.Time) string {
	switch loc {
	case localeRU:
		return fmt.Sprintf("Опросы созданы для окна по Армении %s - %s", start.Format("2006-01-02 15:04"), end.Format("2006-01-02 15:04"))
	case localeHY:
		return fmt.Sprintf("Հարցումները ստեղծված են Երևանի %s - %s պատուհանի համար", start.Format("2006-01-02 15:04"), end.Format("2006-01-02 15:04"))
	default:
		return fmt.Sprintf("Created polls for the Armenia window %s to %s", start.Format("2006-01-02 15:04"), end.Format("2006-01-02 15:04"))
	}
}

func textSettleUsage(loc locale) string {
	switch loc {
	case localeRU:
		return "Используйте /settlematch, /settlematch MATCH_ID или /settlematch MATCH_ID HOME_SCORE AWAY_SCORE"
	case localeHY:
		return "Օգտագործիր /settlematch, /settlematch MATCH_ID կամ /settlematch MATCH_ID HOME_SCORE AWAY_SCORE"
	default:
		return "Use /settlematch, /settlematch MATCH_ID, or /settlematch MATCH_ID HOME_SCORE AWAY_SCORE"
	}
}

func textUnknownMatch(loc locale) string {
	switch loc {
	case localeRU:
		return "Неизвестный ID матча."
	case localeHY:
		return "Խաղի ID-ն չի գտնվել։"
	default:
		return "Unknown match ID."
	}
}

func textHomeScoreMustBeNumber(loc locale) string {
	switch loc {
	case localeRU:
		return "HOME_SCORE должен быть числом."
	case localeHY:
		return "HOME_SCORE-ը պետք է թիվ լինի։"
	default:
		return "HOME_SCORE must be a number."
	}
}

func textAwayScoreMustBeNumber(loc locale) string {
	switch loc {
	case localeRU:
		return "AWAY_SCORE должен быть числом."
	case localeHY:
		return "AWAY_SCORE-ը պետք է թիվ լինի։"
	default:
		return "AWAY_SCORE must be a number."
	}
}

func textMatchHasNoPoll(loc locale) string {
	switch loc {
	case localeRU:
		return "Для этого матча еще нет опроса."
	case localeHY:
		return "Այս խաղի համար հարցում դեռ չկա։"
	default:
		return "This match does not have a poll yet."
	}
}

func textSettledMatch(loc locale, matchID, homeTeam string, home, away int, awayTeam, leaderboard string) string {
	switch loc {
	case localeRU:
		return fmt.Sprintf("Матч %s завершен: %s %d-%d %s\n\n%s", matchID, homeTeam, home, away, awayTeam, leaderboard)
	case localeHY:
		return fmt.Sprintf("Խաղը փակվեց %s: %s %d-%d %s\n\n%s", matchID, homeTeam, home, away, awayTeam, leaderboard)
	default:
		return fmt.Sprintf("Settled %s: %s %d-%d %s\n\n%s", matchID, homeTeam, home, away, awayTeam, leaderboard)
	}
}

func textNoFinishedMatchesToSettle(loc locale) string {
	switch loc {
	case localeRU:
		return "Нет завершенных матчей для автоматического расчета."
	case localeHY:
		return "Ավտոմատ հաշվելու համար ավարտված խաղեր չկան։"
	default:
		return "No finished matches found to settle automatically."
	}
}

func textAutoSettledSummary(loc locale, settled int, leaderboard string) string {
	switch loc {
	case localeRU:
		return fmt.Sprintf("Автоматически завершено матчей: %d\n\n%s", settled, leaderboard)
	case localeHY:
		return fmt.Sprintf("Ավտոմատ փակված խաղերի քանակը՝ %d\n\n%s", settled, leaderboard)
	default:
		return fmt.Sprintf("Auto-settled matches: %d\n\n%s", settled, leaderboard)
	}
}

func textNoMatchesInWindow(loc locale, start, end time.Time) string {
	switch loc {
	case localeRU:
		return fmt.Sprintf("Матчи ЧМ не найдены между %s и %s по армянскому времени.", start.Format("2006-01-02 15:04"), end.Format("2006-01-02 15:04"))
	case localeHY:
		return fmt.Sprintf("ԱԱ խաղեր չեն գտնվել Երևանի ժամով %s - %s միջակայքում։", start.Format("2006-01-02 15:04"), end.Format("2006-01-02 15:04"))
	default:
		return fmt.Sprintf("No World Cup matches found between %s and %s Armenia time.", start.Format("2006-01-02 15:04"), end.Format("2006-01-02 15:04"))
	}
}

func textMatchesHeader(loc locale, start, end time.Time) string {
	switch loc {
	case localeRU:
		return fmt.Sprintf("Матчи ЧМ с %s до %s (армянское время):", start.Format("2006-01-02 15:04"), end.Format("2006-01-02 15:04"))
	case localeHY:
		return fmt.Sprintf("ԱԱ խաղերը %s - %s (Երևանի ժամանակով)՝", start.Format("2006-01-02 15:04"), end.Format("2006-01-02 15:04"))
	default:
		return fmt.Sprintf("World Cup matches from %s to %s (Armenia time):", start.Format("2006-01-02 15:04"), end.Format("2006-01-02 15:04"))
	}
}

func textNoSettledPredictions(loc locale) string {
	switch loc {
	case localeRU:
		return "Пока нет завершенных прогнозов."
	case localeHY:
		return "Փակված կանխատեսումներ դեռ չկան։"
	default:
		return "No settled predictions yet."
	}
}

func textLeaderboardHeader(loc locale) string {
	switch loc {
	case localeRU:
		return "Таблица лидеров:"
	case localeHY:
		return "Առաջատարների աղյուսակ՝"
	default:
		return "Leaderboard:"
	}
}

func textNoUpcomingGuesses(loc locale) string {
	switch loc {
	case localeRU:
		return "Пока нет сохраненных прогнозов на ближайшие матчи."
	case localeHY:
		return "Մոտակա խաղերի պահպանված կանխատեսումներ դեռ չկան։"
	default:
		return "No stored guesses for upcoming matches yet."
	}
}

func textUpcomingGuessesHeader(loc locale) string {
	switch loc {
	case localeRU:
		return "Ближайшие прогнозы:"
	case localeHY:
		return "Մոտակա կանխատեսումներ՝"
	default:
		return "Upcoming guesses:"
	}
}

func textNextMatch(loc locale, fixture Fixture, countdown string) string {
	switch loc {
	case localeRU:
		return fmt.Sprintf("%s vs %s начнется через %s в %s по армянскому времени%s", fixture.HomeTeam, fixture.AwayTeam, countdown, fixture.Kickoff.Format("2006-01-02 15:04"), formatVenueSuffix(fixture.Venue))
	case localeHY:
		return fmt.Sprintf("%s vs %s խաղը կսկսվի %s հետո՝ Երևանի ժամանակով %s%s", fixture.HomeTeam, fixture.AwayTeam, countdown, fixture.Kickoff.Format("2006-01-02 15:04"), formatVenueSuffix(fixture.Venue))
	default:
		return fmt.Sprintf("%s vs %s match starts in %s at %s Armenia time%s", fixture.HomeTeam, fixture.AwayTeam, countdown, fixture.Kickoff.Format("2006-01-02 15:04"), formatVenueSuffix(fixture.Venue))
	}
}

func textNoUpcomingMatches(loc locale) string {
	switch loc {
	case localeRU:
		return "Ближайших матчей ЧМ не найдено."
	case localeHY:
		return "Առաջիկա ԱԱ խաղեր չեն գտնվել։"
	default:
		return "No upcoming World Cup matches found."
	}
}

func localizedOutcomeLabel(loc locale, match MatchRecord, outcome string) string {
	switch outcome {
	case "home":
		if loc == localeRU {
			return match.HomeTeam + " победа"
		}
		if loc == localeHY {
			return match.HomeTeam + " հաղթանակ"
		}
		return match.HomeTeam + " win"
	case "away":
		if loc == localeRU {
			return match.AwayTeam + " победа"
		}
		if loc == localeHY {
			return match.AwayTeam + " հաղթանակ"
		}
		return match.AwayTeam + " win"
	case "draw":
		if loc == localeRU {
			return "ничья"
		}
		if loc == localeHY {
			return "ոչ-ոքի"
		}
		return "draw"
	default:
		return outcome
	}
}
