package bot

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

const wikipediaMatchMarker = `<div itemscope="" itemtype="http&#58;//schema.org/SportsEvent" class="footballbox" style="color:inherit">`

var (
	datePattern  = regexp.MustCompile(`<span class="bday dtstart published updated itvstart">([^<]+)</span>`)
	timePattern  = regexp.MustCompile(`(?s)<div class="ftime">(.*?)</div>`)
	homePattern  = regexp.MustCompile(`(?s)<th class="fhome".*?<a [^>]*>([^<]+)</a>`)
	awayPattern  = regexp.MustCompile(`(?s)<th class="faway".*?<a [^>]*>([^<]+)</a>`)
	scorePattern = regexp.MustCompile(`(?s)<th class="fscore">.*?>(\d+)\s*[–-]\s*(\d+)<`)
	venuePattern = regexp.MustCompile(`(?s)<span itemprop="name address">(.*?)</span>`)
	tagPattern   = regexp.MustCompile(`<[^>]+>`)
	slugPattern  = regexp.MustCompile(`[^a-z0-9]+`)
)

type FixtureProvider struct {
	sourceURL string
	loc       *time.Location
	client    *http.Client
}

func NewFixtureProvider(sourceURL string, loc *time.Location) *FixtureProvider {
	return &FixtureProvider{
		sourceURL: sourceURL,
		loc:       loc,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (p *FixtureProvider) FixturesForArmeniaDay(ctx context.Context, day time.Time) ([]Fixture, error) {
	fixtures, err := p.AllFixtures(ctx)
	if err != nil {
		return nil, err
	}

	start, end := ArmeniaDayWindow(day, p.loc)
	var filtered []Fixture
	for _, fixture := range fixtures {
		if !fixture.Kickoff.Before(start) && fixture.Kickoff.Before(end) {
			fixture.MatchDate = start.Format("2006-01-02")
			filtered = append(filtered, fixture)
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Kickoff.Before(filtered[j].Kickoff)
	})
	return filtered, nil
}

func (p *FixtureProvider) AllFixtures(ctx context.Context) ([]Fixture, error) {
	body, err := p.fetchPage(ctx)
	if err != nil {
		return nil, err
	}

	fixtures, err := parseWikipediaFixtures(body, p.loc)
	if err != nil {
		return nil, err
	}

	sort.Slice(fixtures, func(i, j int) bool {
		return fixtures[i].Kickoff.Before(fixtures[j].Kickoff)
	})
	return fixtures, nil
}

func (p *FixtureProvider) fetchPage(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.sourceURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "world-cup-poll-bot/1.0 (https://telegram.org)")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("fixture source returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func ArmeniaDayWindow(day time.Time, loc *time.Location) (time.Time, time.Time) {
	local := day.In(loc)
	start := time.Date(local.Year(), local.Month(), local.Day(), 19, 0, 0, 0, loc)
	end := start.Add(17 * time.Hour)
	return start, end
}

func parseWikipediaFixtures(page string, loc *time.Location) ([]Fixture, error) {
	parts := strings.Split(page, wikipediaMatchMarker)
	if len(parts) < 2 {
		return nil, fmt.Errorf("could not find match blocks in fixture source")
	}

	fixtures := make([]Fixture, 0, len(parts)-1)
	for _, part := range parts[1:] {
		fixture, ok := parseWikipediaFixtureBlock(part, loc)
		if !ok {
			continue
		}
		fixtures = append(fixtures, fixture)
	}

	return fixtures, nil
}

func parseWikipediaFixtureBlock(block string, loc *time.Location) (Fixture, bool) {
	date := extractMatch(block, datePattern)
	timeText := normalizeText(extractMatch(block, timePattern))
	home := normalizeText(extractMatch(block, homePattern))
	away := normalizeText(extractMatch(block, awayPattern))
	venue := normalizeText(stripTags(extractMatch(block, venuePattern)))
	scoreHome, scoreAway, hasResult := parseScore(block)

	if date == "" || timeText == "" || home == "" || away == "" {
		return Fixture{}, false
	}

	kickoff, err := parseWikipediaKickoff(date, timeText, loc)
	if err != nil {
		return Fixture{}, false
	}

	return Fixture{
		ID:        buildFixtureID(date, home, away),
		HomeTeam:  home,
		AwayTeam:  away,
		Kickoff:   kickoff,
		MatchDate: kickoff.In(loc).Format("2006-01-02"),
		Stage:     "FIFA World Cup 2026",
		Venue:     venue,
		ScoreHome: scoreHome,
		ScoreAway: scoreAway,
		HasResult: hasResult,
	}, true
}

func parseScore(block string) (int, int, bool) {
	matches := scorePattern.FindStringSubmatch(block)
	if len(matches) != 3 {
		return 0, 0, false
	}
	var home, away int
	if _, err := fmt.Sscanf(matches[1], "%d", &home); err != nil {
		return 0, 0, false
	}
	if _, err := fmt.Sscanf(matches[2], "%d", &away); err != nil {
		return 0, 0, false
	}
	return home, away, true
}

func parseWikipediaKickoff(date, timeText string, loc *time.Location) (time.Time, error) {
	clean := strings.ReplaceAll(timeText, "\u00a0", " ")
	clean = strings.ReplaceAll(clean, "−", "-")
	parts := strings.Fields(clean)
	if len(parts) < 3 {
		return time.Time{}, fmt.Errorf("unexpected time format: %q", clean)
	}

	clock := parts[0]
	meridiem := strings.ToUpper(strings.ReplaceAll(parts[1], ".", ""))
	offset := parts[2]
	if !strings.HasPrefix(offset, "UTC") {
		return time.Time{}, fmt.Errorf("unexpected timezone format: %q", clean)
	}

	offsetValue := strings.TrimPrefix(offset, "UTC")
	sign := 1
	if strings.HasPrefix(offsetValue, "-") {
		sign = -1
		offsetValue = strings.TrimPrefix(offsetValue, "-")
	} else if strings.HasPrefix(offsetValue, "+") {
		offsetValue = strings.TrimPrefix(offsetValue, "+")
	}

	hours := 0
	minutes := 0
	if strings.Contains(offsetValue, ":") {
		fmt.Sscanf(offsetValue, "%d:%d", &hours, &minutes)
	} else {
		fmt.Sscanf(offsetValue, "%d", &hours)
	}

	sourceZone := time.FixedZone(offset, sign*(hours*3600+minutes*60))
	sourceTime, err := time.ParseInLocation("2006-01-02 3:04 PM", fmt.Sprintf("%s %s %s", date, clock, meridiem), sourceZone)
	if err != nil {
		return time.Time{}, err
	}
	return sourceTime.In(loc), nil
}

func extractMatch(input string, pattern *regexp.Regexp) string {
	matches := pattern.FindStringSubmatch(input)
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}

func stripTags(raw string) string {
	return tagPattern.ReplaceAllString(raw, " ")
}

func normalizeText(raw string) string {
	if raw == "" {
		return ""
	}
	text := html.UnescapeString(raw)
	text = strings.ReplaceAll(text, "\u00a0", " ")
	text = stripTags(text)
	text = strings.Join(strings.Fields(text), " ")
	text = strings.ReplaceAll(text, " ,", ",")
	return text
}

func buildFixtureID(date, home, away string) string {
	return fmt.Sprintf("wc-%s-%s-%s", date, slugify(home), slugify(away))
}

func slugify(value string) string {
	value = strings.ToLower(value)
	value = slugPattern.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	return value
}
