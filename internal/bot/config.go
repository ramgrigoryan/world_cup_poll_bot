package bot

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Token             string
	Admins            map[int64]struct{}
	DataDir           string
	ListenTimeout     time.Duration
	GroupChatIDs      []int64
	DailyPollHour     int
	DailyPollMinute   int
	TimeZone          *time.Location
	FixturesSourceURL string
}

func LoadConfig() (Config, error) {
	token := strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if token == "" {
		return Config{}, errors.New("TELEGRAM_BOT_TOKEN is required")
	}

	dataDir := strings.TrimSpace(os.Getenv("BOT_DATA_DIR"))
	if dataDir == "" {
		dataDir = "data"
	}

	fixturesSourceURL := strings.TrimSpace(os.Getenv("FIXTURES_SOURCE_URL"))
	if fixturesSourceURL == "" {
		fixturesSourceURL = "https://en.wikipedia.org/wiki/2026_FIFA_World_Cup"
	}

	groupChatIDs, err := parseChatIDs(os.Getenv("GROUP_CHAT_IDS"))
	if err != nil {
		return Config{}, err
	}
	if len(groupChatIDs) == 0 {
		groupChatID, err := parseInt64Env("GROUP_CHAT_ID", 0)
		if err != nil {
			return Config{}, err
		}
		if groupChatID != 0 {
			groupChatIDs = []int64{groupChatID}
		}
	}

	timeoutSeconds, err := parseIntEnv("LISTEN_TIMEOUT_SECONDS", 25)
	if err != nil {
		return Config{}, err
	}

	hour, err := parseIntEnv("DAILY_POLL_HOUR", 19)
	if err != nil {
		return Config{}, err
	}
	minute, err := parseIntEnv("DAILY_POLL_MINUTE", 0)
	if err != nil {
		return Config{}, err
	}

	location, err := time.LoadLocation("Asia/Yerevan")
	if err != nil {
		return Config{}, err
	}
	tz := strings.TrimSpace(os.Getenv("BOT_TIMEZONE"))
	if tz != "" {
		location, err = time.LoadLocation(tz)
		if err != nil {
			return Config{}, err
		}
	}

	admins, err := parseAdmins(os.Getenv("ADMIN_USER_IDS"))
	if err != nil {
		return Config{}, err
	}

	return Config{
		Token:             token,
		Admins:            admins,
		DataDir:           dataDir,
		ListenTimeout:     time.Duration(timeoutSeconds) * time.Second,
		GroupChatIDs:      groupChatIDs,
		DailyPollHour:     hour,
		DailyPollMinute:   minute,
		TimeZone:          location,
		FixturesSourceURL: fixturesSourceURL,
	}, nil
}

func parseChatIDs(raw string) ([]int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	ids := make([]int64, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func parseAdmins(raw string) (map[int64]struct{}, error) {
	admins := make(map[int64]struct{})
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return admins, nil
	}

	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return nil, err
		}
		admins[id] = struct{}{}
	}

	return admins, nil
}

func parseIntEnv(name string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}
	return strconv.Atoi(raw)
}

func parseInt64Env(name string, fallback int64) (int64, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}
	return strconv.ParseInt(raw, 10, 64)
}
