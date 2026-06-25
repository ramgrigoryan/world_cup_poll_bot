package bot

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"
)

var pollOptions = []string{"Home win", "Draw", "Away win"}

type App struct {
	cfg      Config
	store    *Store
	client   *TelegramClient
	fixtures *FixtureProvider
	lock     *InstanceLock
}

func NewApp(cfg Config) (*App, error) {
	lock, err := AcquireInstanceLock(cfg.DataDir)
	if err != nil {
		return nil, err
	}

	store, err := NewStore(cfg.DataDir)
	if err != nil {
		lock.Release()
		return nil, err
	}

	now := time.Now().In(cfg.TimeZone).Format(time.RFC3339)
	for _, chatID := range cfg.GroupChatIDs {
		store.RegisterChat(RegisteredChat{
			ChatID:       chatID,
			Type:         "group",
			RegisteredAt: now,
			LastSeenAt:   now,
		})
	}
	if len(cfg.GroupChatIDs) > 0 {
		if err := store.Save(); err != nil {
			lock.Release()
			return nil, err
		}
	}

	return &App{
		cfg:      cfg,
		store:    store,
		client:   NewTelegramClient(cfg.Token, cfg.ListenTimeout),
		fixtures: NewFixtureProvider(cfg.FixturesSourceURL, cfg.TimeZone),
		lock:     lock,
	}, nil
}

func (a *App) Close() error {
	if a == nil {
		return nil
	}
	return a.lock.Release()
}

func (a *App) Run(ctx context.Context) error {
	if err := a.registerCommandMenus(ctx); err != nil {
		log.Printf("setMyCommands failed: %v", err)
	}
	go a.runDailyScheduler(ctx)

	offset := a.store.LastUpdateID() + 1
	backoff := 3 * time.Second
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		updates, err := a.client.GetUpdates(ctx, offset, a.cfg.ListenTimeout)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			log.Printf("get updates: %v", err)
			time.Sleep(backoff)
			if backoff < 30*time.Second {
				backoff *= 2
				if backoff > 30*time.Second {
					backoff = 30 * time.Second
				}
			}
			continue
		}
		backoff = 3 * time.Second

		for _, update := range updates {
			if err := a.handleUpdate(ctx, update); err != nil {
				log.Printf("handle update %d: %v", update.UpdateID, err)
			}
			offset = update.UpdateID + 1
			a.store.SetLastUpdateID(update.UpdateID)
			if err := a.store.Save(); err != nil {
				log.Printf("save state: %v", err)
			}
		}
	}
}

func (a *App) runDailyScheduler(ctx context.Context) {
	timer := time.NewTimer(timeUntilNext(a.cfg.TimeZone, a.cfg.DailyPollHour, a.cfg.DailyPollMinute))
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			for _, chatID := range a.store.RegisteredChatIDs() {
				if err := a.createPollsForDate(ctx, chatID, time.Now().In(a.cfg.TimeZone), localeEN); err != nil {
					log.Printf("daily poll creation failed: %v", err)
				}
			}
			timer.Reset(24 * time.Hour)
		}
	}
}

func timeUntilNext(loc *time.Location, hour, minute int) time.Duration {
	now := time.Now().In(loc)
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc)
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next.Sub(now)
}

func (a *App) handleUpdate(ctx context.Context, update Update) error {
	switch {
	case update.Message != nil:
		return a.handleMessage(ctx, *update.Message)
	case update.PollAnswer != nil:
		return a.handlePollAnswer(ctx, *update.PollAnswer)
	default:
		return nil
	}
}

func (a *App) handleMessage(ctx context.Context, message Message) error {
	text := strings.TrimSpace(message.Text)
	log.Printf("received message in chat_id=%d chat_type=%s text=%q", message.Chat.ID, message.Chat.Type, text)
	a.registerChat(message.Chat)
	loc := a.localeForChat(message.Chat.ID, message.From)
	if text == "" || !strings.HasPrefix(text, "/") {
		return nil
	}

	command, args := splitCommand(text)
	switch command {
	case "/start", "/help":
		return a.client.SendMessage(ctx, message.Chat.ID, helpText(loc))
	case "/chatid":
		return a.client.SendMessage(ctx, message.Chat.ID, textThisChatID(loc, message.Chat.ID))
	case "/lang":
		ok, err := a.canChangeLanguage(ctx, message.Chat, message.From)
		if err != nil {
			return a.client.SendMessage(ctx, message.Chat.ID, textVerifyAdminError(loc))
		}
		if !ok {
			return a.client.SendMessage(ctx, message.Chat.ID, textOnlyAdminsCanChangeLanguage(loc))
		}
		if len(args) != 1 {
			return a.client.SendMessage(ctx, message.Chat.ID, textLanguageUsage(loc))
		}
		selected, valid := normalizeLocaleCode(args[0])
		if !valid {
			return a.client.SendMessage(ctx, message.Chat.ID, textLanguageUsage(loc))
		}
		a.setChatLocale(message.Chat, selected)
		if err := a.store.Save(); err != nil {
			return a.client.SendMessage(ctx, message.Chat.ID, "Could not save language setting.")
		}
		return a.client.SendMessage(ctx, message.Chat.ID, textLanguageSet(selected, selected))
	case "/createpolls":
		day := time.Now().In(a.cfg.TimeZone)
		if len(args) == 1 {
			parsed, err := time.ParseInLocation("2006-01-02", args[0], a.cfg.TimeZone)
			if err != nil {
				return a.client.SendMessage(ctx, message.Chat.ID, textUseDateForCreate(loc))
			}
			day = parsed
		}
		if err := a.createPollsForDate(ctx, message.Chat.ID, day, loc); err != nil {
			return a.client.SendMessage(ctx, message.Chat.ID, textCouldNotCreatePolls(loc, err))
		}
		return nil
	case "/settlematch":
		ok, err := a.isAdmin(ctx, message.Chat, message.From)
		if err != nil {
			return a.client.SendMessage(ctx, message.Chat.ID, textVerifyAdminError(loc))
		}
		if !ok {
			return a.client.SendMessage(ctx, message.Chat.ID, textOnlyAdminsCanSettle(loc))
		}
		return a.handleSettleMatch(ctx, message.Chat.ID, args, loc)
	case "/leaderboard":
		return a.client.SendMessage(ctx, message.Chat.ID, a.formatLeaderboard(message.Chat.ID, loc))
	case "/guesses":
		return a.client.SendMessage(ctx, message.Chat.ID, a.formatUpcomingGuesses(time.Now().In(a.cfg.TimeZone), message.Chat.ID, loc))
	case "/matches":
		day := time.Now().In(a.cfg.TimeZone)
		if len(args) == 1 {
			parsed, err := time.ParseInLocation("2006-01-02", args[0], a.cfg.TimeZone)
			if err != nil {
				return a.client.SendMessage(ctx, message.Chat.ID, textUseDateForMatches(loc))
			}
			day = parsed
		}
		output, err := a.formatMatches(ctx, day, loc)
		if err != nil {
			return a.client.SendMessage(ctx, message.Chat.ID, textCouldNotLoadMatches(loc, err))
		}
		return a.client.SendMessage(ctx, message.Chat.ID, output)
	case "/nextmatch":
		output, err := a.formatNextMatch(ctx, time.Now().In(a.cfg.TimeZone), loc)
		if err != nil {
			return a.client.SendMessage(ctx, message.Chat.ID, textCouldNotLoadNextMatch(loc, err))
		}
		return a.client.SendMessage(ctx, message.Chat.ID, output)
	default:
		return nil
	}
}

func splitCommand(text string) (string, []string) {
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return "", nil
	}
	command := parts[0]
	if idx := strings.Index(command, "@"); idx >= 0 {
		command = command[:idx]
	}
	return command, parts[1:]
}

func (a *App) handlePollAnswer(ctx context.Context, answer PollAnswer) error {
	if len(answer.OptionIDs) == 0 {
		return nil
	}

	match, ok := a.store.MatchByPollID(answer.PollID)
	if !ok {
		return nil
	}

	username := answer.User.Username
	if username == "" {
		username = strings.TrimSpace(answer.User.FirstName + " " + answer.User.LastName)
	}
	if username == "" {
		username = strconv.FormatInt(answer.User.ID, 10)
	}

	if !time.Now().In(a.cfg.TimeZone).Before(match.Kickoff.In(a.cfg.TimeZone)) {
		loc := a.localeForChat(match.PollChatID, &answer.User)
		if err := a.client.SendMessage(ctx, match.PollChatID, textLateVoteRejected(loc, formatUserLabel(username), match.HomeTeam, match.AwayTeam)); err != nil {
			log.Printf("late vote warning failed for %s: %v", match.ID, err)
		}
		return nil
	}

	a.store.SavePrediction(answer.PollID, Prediction{
		UserID:    answer.User.ID,
		Username:  username,
		Option:    answer.OptionIDs[0],
		Outcome:   optionToOutcome(answer.OptionIDs[0]),
		UpdatedAt: time.Now().In(a.cfg.TimeZone).Format(time.RFC3339),
	})

	log.Printf("saved prediction for %s by %s on match %s", answer.PollID, username, match.ID)
	return nil
}

func (a *App) registerChat(chat Chat) {
	if chat.Type != "group" && chat.Type != "supergroup" {
		return
	}

	now := time.Now().In(a.cfg.TimeZone).Format(time.RFC3339)
	title := chat.Title
	if title == "" {
		title = strconv.FormatInt(chat.ID, 10)
	}

	a.store.RegisterChat(RegisteredChat{
		ChatID:     chat.ID,
		Title:      title,
		Type:       chat.Type,
		LastSeenAt: now,
	})
}

func (a *App) setChatLocale(chat Chat, loc locale) {
	now := time.Now().In(a.cfg.TimeZone).Format(time.RFC3339)
	existing, _ := a.store.ChatByID(chat.ID)
	a.store.RegisterChat(RegisteredChat{
		ChatID:       chat.ID,
		Title:        firstNonEmpty(chat.Title, existing.Title),
		Type:         firstNonEmpty(chat.Type, existing.Type),
		Locale:       localeLabel(loc),
		RegisteredAt: existing.RegisteredAt,
		LastSeenAt:   now,
	})
}

func (a *App) localeForChat(chatID int64, user *User) locale {
	if chat, ok := a.store.ChatByID(chatID); ok {
		if loc, valid := normalizeLocaleCode(chat.Locale); valid {
			return loc
		}
	}
	return detectLocale(user)
}

func (a *App) canChangeLanguage(ctx context.Context, chat Chat, user *User) (bool, error) {
	if chat.Type == "private" {
		return true, nil
	}
	return a.isAdmin(ctx, chat, user)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func (a *App) createPollsForDate(ctx context.Context, chatID int64, day time.Time, loc locale) error {
	fixtures, err := a.fixtures.FixturesForArmeniaDay(ctx, day)
	if err != nil {
		return err
	}
	start, end := ArmeniaDayWindow(day, a.cfg.TimeZone)
	if len(fixtures) == 0 {
		return a.client.SendMessage(ctx, chatID, textNoFixtures(loc, start, end))
	}

	allExist := true
	firstExisting := MatchRecord{}
	firstExistingSet := false
	for _, fixture := range fixtures {
		existing, ok := a.store.MatchByFixtureAndChat(fixture.ID, chatID)
		if ok && existing.PollID != "" {
			if !firstExistingSet || existing.PollMessageID < firstExisting.PollMessageID {
				firstExisting = existing
				firstExistingSet = true
			}
			continue
		}
		allExist = false

		question := fmt.Sprintf("%s vs %s", fixture.HomeTeam, fixture.AwayTeam)
		msg, err := a.client.SendPoll(ctx, chatID, question, []string{
			fmt.Sprintf("%s win", fixture.HomeTeam),
			"Draw",
			fmt.Sprintf("%s win", fixture.AwayTeam),
		})
		if err != nil {
			return err
		}

		record := MatchRecord{
			Fixture:        fixture,
			FixtureID:      fixture.ID,
			PollID:         msg.Poll.ID,
			PollMessageID:  msg.MessageID,
			PollChatID:     msg.Chat.ID,
			PollsCreatedAt: time.Now().In(a.cfg.TimeZone).Format(time.RFC3339),
		}
		a.store.UpsertMatch(record)
		if !firstExistingSet || record.PollMessageID < firstExisting.PollMessageID {
			firstExisting = record
			firstExistingSet = true
		}
	}

	if err := a.store.Save(); err != nil {
		return err
	}

	if allExist && firstExistingSet {
		if err := a.client.SendReply(ctx, chatID, firstExisting.PollMessageID, textPollsAlreadyCreated(loc)); err == nil {
			return nil
		}
		return a.client.SendMessage(ctx, chatID, textPollsAlreadyCreatedFallback(loc))
	}

	return a.client.SendMessage(ctx, chatID, textCreatedPolls(loc, start, end))
}

func (a *App) handleSettleMatch(ctx context.Context, chatID int64, args []string, loc locale) error {
	switch len(args) {
	case 0:
		return a.autoSettleMatches(ctx, chatID, "", loc)
	case 1:
		return a.autoSettleMatches(ctx, chatID, args[0], loc)
	case 3:
		return a.manualSettleMatch(ctx, chatID, args, loc)
	default:
		return a.client.SendMessage(ctx, chatID, textSettleUsage(loc))
	}
}

func (a *App) manualSettleMatch(ctx context.Context, chatID int64, args []string, loc locale) error {
	match, ok := a.store.MatchByFixtureAndChat(args[0], chatID)
	if !ok {
		return a.client.SendMessage(ctx, chatID, textUnknownMatch(loc))
	}

	home, err := strconv.Atoi(args[1])
	if err != nil {
		return a.client.SendMessage(ctx, chatID, textHomeScoreMustBeNumber(loc))
	}
	away, err := strconv.Atoi(args[2])
	if err != nil {
		return a.client.SendMessage(ctx, chatID, textAwayScoreMustBeNumber(loc))
	}

	if match.PollID == "" {
		return a.client.SendMessage(ctx, chatID, textMatchHasNoPoll(loc))
	}

	if err := a.applySettledResult(ctx, match, home, away); err != nil {
		return err
	}
	if err := a.store.Save(); err != nil {
		return err
	}

	return a.client.SendMessage(ctx, chatID, textSettledMatch(loc, match.ID, match.HomeTeam, home, away, match.AwayTeam, a.formatLeaderboard(chatID, loc)))
}

func (a *App) autoSettleMatches(ctx context.Context, chatID int64, fixtureID string, loc locale) error {
	fixtures, err := a.fixtures.AllFixtures(ctx)
	if err != nil {
		return err
	}

	fixtureByID := make(map[string]Fixture, len(fixtures))
	for _, fixture := range fixtures {
		fixtureByID[fixture.ID] = fixture
	}

	settledCount := 0
	matches := a.store.MatchesForChat(chatID)
	for _, match := range matches {
		if match.PollID == "" || match.Settled {
			continue
		}
		if fixtureID != "" && match.FixtureIdentifier() != fixtureID {
			continue
		}

		fixture, ok := fixtureByID[match.FixtureIdentifier()]
		if !ok || !fixture.HasResult {
			continue
		}
		if err := a.applySettledResult(ctx, match, fixture.ScoreHome, fixture.ScoreAway); err != nil {
			return err
		}
		settledCount++
		if fixtureID != "" {
			break
		}
	}

	if settledCount == 0 {
		return a.client.SendMessage(ctx, chatID, textNoFinishedMatchesToSettle(loc))
	}
	if err := a.store.Save(); err != nil {
		return err
	}
	return a.client.SendMessage(ctx, chatID, textAutoSettledSummary(loc, settledCount, a.formatLeaderboard(chatID, loc)))
}

func (a *App) applySettledResult(ctx context.Context, match MatchRecord, home, away int) error {
	match.ScoreHome = home
	match.ScoreAway = away
	match.Result = resolveOutcome(home, away)
	match.Settled = true
	a.store.UpsertMatch(match)

	if err := a.client.StopPoll(ctx, match.PollChatID, match.PollMessageID); err != nil {
		log.Printf("stop poll failed for %s: %v", match.ID, err)
	}
	return nil
}

func (a *App) isAdmin(ctx context.Context, chat Chat, user *User) (bool, error) {
	if user == nil {
		return false, nil
	}

	if _, ok := a.cfg.Admins[user.ID]; ok {
		return true, nil
	}

	if chat.Type != "group" && chat.Type != "supergroup" {
		return len(a.cfg.Admins) == 0, nil
	}

	admins, err := a.client.GetChatAdministrators(ctx, chat.ID)
	if err != nil {
		return false, err
	}
	for _, admin := range admins {
		if admin.User.ID != user.ID {
			continue
		}
		if admin.Status == "creator" || admin.Status == "administrator" {
			return true, nil
		}
	}
	return false, nil
}

func resolveOutcome(home, away int) string {
	switch {
	case home > away:
		return "home"
	case away > home:
		return "away"
	default:
		return "draw"
	}
}

func optionToOutcome(option int) string {
	switch option {
	case 0:
		return "home"
	case 1:
		return "draw"
	case 2:
		return "away"
	default:
		return ""
	}
}

func (a *App) formatMatches(ctx context.Context, day time.Time, loc locale) (string, error) {
	fixtures, err := a.fixtures.FixturesForArmeniaDay(ctx, day)
	if err != nil {
		return "", err
	}

	start, end := ArmeniaDayWindow(day, a.cfg.TimeZone)
	if len(fixtures) == 0 {
		return textNoMatchesInWindow(loc, start, end), nil
	}

	lines := []string{textMatchesHeader(loc, start, end)}
	for _, fixture := range fixtures {
		lines = append(lines, fmt.Sprintf(
			"%s vs %s at %s%s",
			fixture.HomeTeam,
			fixture.AwayTeam,
			fixture.Kickoff.In(a.cfg.TimeZone).Format("2006-01-02 15:04"),
			formatVenueSuffix(fixture.Venue),
		))
	}
	return strings.Join(lines, "\n"), nil
}

func (a *App) formatLeaderboard(chatID int64, loc locale) string {
	stats := a.computeLeaderboard(chatID)
	if len(stats) == 0 {
		return textNoSettledPredictions(loc)
	}

	lines := []string{textLeaderboardHeader(loc)}
	for i, row := range stats {
		total := row.Correct + row.Wrong
		accuracy := 0.0
		if total > 0 {
			accuracy = float64(row.Correct) / float64(total) * 100
		}
		name := formatUserLabel(row.Username)
		lines = append(lines, fmt.Sprintf(
			"%d. %s | matches: %d | correct: %d | wrong: %d | accuracy: %.1f%%",
			i+1,
			name,
			total,
			row.Correct,
			row.Wrong,
			accuracy,
		))
	}
	return strings.Join(lines, "\n")
}

func (a *App) formatUpcomingGuesses(now time.Time, chatID int64, loc locale) string {
	type row struct {
		match      MatchRecord
		prediction Prediction
		updatedAt  time.Time
	}

	var rows []row
	for _, match := range a.store.MatchesForChat(chatID) {
		if match.Settled || match.Kickoff.Before(now) || match.PollID == "" {
			continue
		}
		for _, prediction := range a.store.PredictionsForPoll(match.PollID) {
			updatedAt, _ := time.Parse(time.RFC3339, prediction.UpdatedAt)
			rows = append(rows, row{
				match:      match,
				prediction: prediction,
				updatedAt:  updatedAt,
			})
		}
	}

	if len(rows) == 0 {
		return textNoUpcomingGuesses(loc)
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].match.Kickoff.Equal(rows[j].match.Kickoff) {
			return rows[i].updatedAt.After(rows[j].updatedAt)
		}
		return rows[i].match.Kickoff.Before(rows[j].match.Kickoff)
	})

	lines := []string{textUpcomingGuessesHeader(loc)}
	for _, item := range rows {
		lines = append(lines, fmt.Sprintf(
			"%s | %s vs %s | %s picked %s",
			item.match.Kickoff.In(a.cfg.TimeZone).Format("2006-01-02 15:04"),
			item.match.HomeTeam,
			item.match.AwayTeam,
			formatUserLabel(item.prediction.Username),
			localizedOutcomeLabel(loc, item.match, item.prediction.Outcome),
		))
	}
	return strings.Join(lines, "\n")
}

func (a *App) formatNextMatch(ctx context.Context, now time.Time, loc locale) (string, error) {
	fixtures, err := a.fixtures.AllFixtures(ctx)
	if err != nil {
		return "", err
	}

	for _, fixture := range fixtures {
		if fixture.Kickoff.After(now) {
			return textNextMatch(loc, fixture, formatCountdown(now, fixture.Kickoff)), nil
		}
	}

	return textNoUpcomingMatches(loc), nil
}

func (a *App) computeLeaderboard(chatID int64) []UserStats {
	rows := make(map[string]UserStats)

	for _, match := range a.store.MatchesForChat(chatID) {
		if !match.Settled || match.PollID == "" {
			continue
		}
		for _, prediction := range a.store.PredictionsForPoll(match.PollID) {
			key := leaderboardUserKey(prediction)
			row := rows[key]
			if row.UserID == 0 {
				row = UserStats{
					UserID:   prediction.UserID,
					Username: prediction.Username,
				}
			}
			if prediction.Outcome == match.Result {
				row.Correct++
			} else {
				row.Wrong++
			}
			rows[key] = row
		}
	}

	result := make([]UserStats, 0, len(rows))
	for _, row := range rows {
		result = append(result, row)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Correct == result[j].Correct {
			leftTotal := result[i].Correct + result[i].Wrong
			rightTotal := result[j].Correct + result[j].Wrong
			if leftTotal == rightTotal {
				return strings.ToLower(result[i].Username) < strings.ToLower(result[j].Username)
			}
			return leftTotal < rightTotal
		}
		return result[i].Correct > result[j].Correct
	})
	return result
}

func leaderboardUserKey(prediction Prediction) string {
	username := strings.ToLower(strings.TrimSpace(prediction.Username))
	if username != "" {
		return "username:" + username
	}
	return "user_id:" + strconv.FormatInt(prediction.UserID, 10)
}

func formatUserLabel(name string) string {
	if name == "" {
		return "unknown"
	}
	if strings.Contains(name, " ") {
		return name
	}
	if strings.HasPrefix(name, "@") {
		return name
	}
	return "@" + name
}

func formatVenueSuffix(venue string) string {
	if venue == "" {
		return ""
	}
	return " | " + venue
}

func formatCountdown(from, to time.Time) string {
	if !to.After(from) {
		return "0m"
	}

	diff := to.Sub(from).Round(time.Minute)
	hours := diff / time.Hour
	minutes := (diff % time.Hour) / time.Minute

	if hours == 0 {
		return fmt.Sprintf("%dm", minutes)
	}
	if minutes == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh %dm", hours, minutes)
}

func (a *App) registerCommandMenus(ctx context.Context) error {
	locales := []locale{localeEN, localeRU, localeHY}
	scopes := []*BotCommandScope{
		{Type: "default"},
		{Type: "all_private_chats"},
		{Type: "all_group_chats"},
		{Type: "all_chat_administrators"},
	}
	for _, loc := range locales {
		for _, scope := range scopes {
			if err := a.client.SetMyCommands(ctx, commandsForLocale(loc), languageCodeForCommands(loc), scope); err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *App) Validate() error {
	return nil
}
