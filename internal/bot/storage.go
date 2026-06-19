package bot

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
)

type Store struct {
	mu   sync.RWMutex
	path string
	data State
}

func NewStore(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, err
	}

	store := &Store{
		path: filepath.Join(dataDir, "state.json"),
		data: State{
			Matches:     make(map[string]MatchRecord),
			Predictions: make(map[string]map[int64]Prediction),
			Chats:       make(map[string]RegisteredChat),
		},
	}

	if err := store.load(); err != nil {
		return nil, err
	}
	if store.data.Matches == nil {
		store.data.Matches = make(map[string]MatchRecord)
	}
	if store.data.Predictions == nil {
		store.data.Predictions = make(map[string]map[int64]Prediction)
	}
	if store.data.Chats == nil {
		store.data.Chats = make(map[string]RegisteredChat)
	}

	return store, nil
}

func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	bytes, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(bytes) == 0 {
		return nil
	}
	return json.Unmarshal(bytes, &s.data)
}

func (s *Store) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	payload, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, payload, 0o644)
}

func (s *Store) LastUpdateID() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.LastUpdateID
}

func (s *Store) SetLastUpdateID(id int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.LastUpdateID = id
}

func (s *Store) UpsertMatch(match MatchRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Matches[matchRecordKey(match.PollChatID, match.FixtureIdentifier())] = match
}

func (s *Store) MatchByPollID(pollID string) (MatchRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, match := range s.data.Matches {
		if match.PollID == pollID {
			return match, true
		}
	}
	return MatchRecord{}, false
}

func (s *Store) MatchByFixtureAndChat(fixtureID string, chatID int64) (MatchRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	match, ok := s.data.Matches[matchRecordKey(chatID, fixtureID)]
	return match, ok
}

func (s *Store) MatchesForDate(date string, chatID int64) []MatchRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var matches []MatchRecord
	for _, match := range s.data.Matches {
		if match.MatchDate == date && match.PollChatID == chatID {
			matches = append(matches, match)
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Kickoff.Before(matches[j].Kickoff)
	})
	return matches
}

func (s *Store) SavePrediction(pollID string, prediction Prediction) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.Predictions[pollID] == nil {
		s.data.Predictions[pollID] = make(map[int64]Prediction)
	}
	s.data.Predictions[pollID][prediction.UserID] = prediction
}

func (s *Store) RegisterChat(chat RegisteredChat) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := stringKey(chat.ChatID)
	existing, ok := s.data.Chats[key]
	if ok {
		if chat.Title == "" {
			chat.Title = existing.Title
		}
		if chat.Type == "" {
			chat.Type = existing.Type
		}
		if chat.Locale == "" {
			chat.Locale = existing.Locale
		}
		if chat.RegisteredAt == "" {
			chat.RegisteredAt = existing.RegisteredAt
		}
	}
	if chat.RegisteredAt == "" {
		chat.RegisteredAt = chat.LastSeenAt
	}
	s.data.Chats[key] = chat
}

func (s *Store) ChatByID(chatID int64) (RegisteredChat, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	chat, ok := s.data.Chats[stringKey(chatID)]
	return chat, ok
}

func (s *Store) RegisteredChatIDs() []int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]int64, 0, len(s.data.Chats))
	for _, chat := range s.data.Chats {
		ids = append(ids, chat.ChatID)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func (s *Store) PredictionsForPoll(pollID string) []Prediction {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []Prediction
	for _, prediction := range s.data.Predictions[pollID] {
		result = append(result, prediction)
	}
	return result
}

func (s *Store) AllMatches() []MatchRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]MatchRecord, 0, len(s.data.Matches))
	for _, match := range s.data.Matches {
		result = append(result, match)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].MatchDate == result[j].MatchDate {
			return result[i].Kickoff.Before(result[j].Kickoff)
		}
		return result[i].MatchDate < result[j].MatchDate
	})
	return result
}

func (s *Store) MatchesForChat(chatID int64) []MatchRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]MatchRecord, 0)
	for _, match := range s.data.Matches {
		if match.PollChatID == chatID {
			result = append(result, match)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].MatchDate == result[j].MatchDate {
			return result[i].Kickoff.Before(result[j].Kickoff)
		}
		return result[i].MatchDate < result[j].MatchDate
	})
	return result
}

func matchRecordKey(chatID int64, fixtureID string) string {
	return fixtureID + "|" + stringKey(chatID)
}

func stringKey(id int64) string {
	return strconv.FormatInt(id, 10)
}
