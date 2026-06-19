package bot

import "time"

type UpdateResponse struct {
	OK     bool     `json:"ok"`
	Result []Update `json:"result"`
}

type SendMessageResponse struct {
	OK     bool    `json:"ok"`
	Result Message `json:"result"`
}

type SendPollResponse struct {
	OK     bool    `json:"ok"`
	Result Message `json:"result"`
}

type BaseResponse struct {
	OK bool `json:"ok"`
}

type ChatAdministratorsResponse struct {
	OK     bool         `json:"ok"`
	Result []ChatMember `json:"result"`
}

type Update struct {
	UpdateID   int64       `json:"update_id"`
	Message    *Message    `json:"message,omitempty"`
	PollAnswer *PollAnswer `json:"poll_answer,omitempty"`
}

type Message struct {
	MessageID int64     `json:"message_id"`
	Chat      Chat      `json:"chat"`
	From      *User     `json:"from,omitempty"`
	Text      string    `json:"text,omitempty"`
	Date      int64     `json:"date"`
	Poll      *PollInfo `json:"poll,omitempty"`
}

type Chat struct {
	ID    int64  `json:"id"`
	Title string `json:"title,omitempty"`
	Type  string `json:"type"`
}

type ChatMember struct {
	User   User   `json:"user"`
	Status string `json:"status"`
}

type User struct {
	ID           int64  `json:"id"`
	IsBot        bool   `json:"is_bot"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name,omitempty"`
	Username     string `json:"username,omitempty"`
	LanguageCode string `json:"language_code,omitempty"`
}

type PollInfo struct {
	ID       string       `json:"id"`
	Question string       `json:"question"`
	Options  []PollOption `json:"options"`
}

type PollOption struct {
	Text       string `json:"text"`
	VoterCount int    `json:"voter_count"`
}

type PollAnswer struct {
	PollID    string `json:"poll_id"`
	User      User   `json:"user"`
	OptionIDs []int  `json:"option_ids"`
}

type Fixture struct {
	ID        string    `json:"id"`
	HomeTeam  string    `json:"home_team"`
	AwayTeam  string    `json:"away_team"`
	Kickoff   time.Time `json:"kickoff"`
	MatchDate string    `json:"match_date"`
	Stage     string    `json:"stage,omitempty"`
	Venue     string    `json:"venue,omitempty"`
	ScoreHome int       `json:"score_home,omitempty"`
	ScoreAway int       `json:"score_away,omitempty"`
	HasResult bool      `json:"has_result,omitempty"`
}

type MatchRecord struct {
	Fixture
	FixtureID      string `json:"fixture_id,omitempty"`
	PollID         string `json:"poll_id"`
	PollMessageID  int64  `json:"poll_message_id"`
	PollChatID     int64  `json:"poll_chat_id"`
	PollsCreatedAt string `json:"polls_created_at,omitempty"`
	Result         string `json:"result,omitempty"`
	ScoreHome      int    `json:"score_home,omitempty"`
	ScoreAway      int    `json:"score_away,omitempty"`
	Settled        bool   `json:"settled"`
}

func (m MatchRecord) FixtureIdentifier() string {
	if m.FixtureID != "" {
		return m.FixtureID
	}
	return m.Fixture.ID
}

type Prediction struct {
	UserID    int64  `json:"user_id"`
	Username  string `json:"username"`
	Option    int    `json:"option"`
	Outcome   string `json:"outcome,omitempty"`
	UpdatedAt string `json:"updated_at"`
}

type UserStats struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Correct  int    `json:"correct"`
	Wrong    int    `json:"wrong"`
}

type RegisteredChat struct {
	ChatID       int64  `json:"chat_id"`
	Title        string `json:"title,omitempty"`
	Type         string `json:"type,omitempty"`
	Locale       string `json:"locale,omitempty"`
	RegisteredAt string `json:"registered_at,omitempty"`
	LastSeenAt   string `json:"last_seen_at,omitempty"`
}

type State struct {
	LastUpdateID int64                           `json:"last_update_id"`
	Matches      map[string]MatchRecord          `json:"matches"`
	Predictions  map[string]map[int64]Prediction `json:"predictions"`
	Chats        map[string]RegisteredChat       `json:"chats"`
}

type BotCommand struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

type BotCommandScope struct {
	Type string `json:"type"`
}
