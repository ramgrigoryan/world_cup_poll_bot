package bot

import (
	"testing"
	"time"
)

func TestComputeLeaderboard(t *testing.T) {
	store := &Store{
		data: State{
			Matches: map[string]MatchRecord{
				"m1|100": {Fixture: Fixture{ID: "m1"}, FixtureID: "m1", PollID: "p1", PollChatID: 100, Result: "home", Settled: true},
				"m2|100": {Fixture: Fixture{ID: "m2"}, FixtureID: "m2", PollID: "p2", PollChatID: 100, Result: "draw", Settled: true},
				"m3|200": {Fixture: Fixture{ID: "m3"}, FixtureID: "m3", PollID: "p3", PollChatID: 200, Result: "away", Settled: true},
			},
			Predictions: map[string]map[int64]Prediction{
				"p1": {
					1: {UserID: 1, Username: "alice", Outcome: "home"},
					2: {UserID: 2, Username: "bob", Outcome: "away"},
				},
				"p2": {
					1: {UserID: 1, Username: "alice", Outcome: "draw"},
					2: {UserID: 2, Username: "bob", Outcome: "draw"},
				},
				"p3": {
					3: {UserID: 3, Username: "charlie", Outcome: "away"},
				},
			},
		},
	}

	app := &App{store: store}
	stats := app.computeLeaderboard(100)
	if len(stats) != 2 {
		t.Fatalf("expected 2 users, got %d", len(stats))
	}
	if stats[0].Username != "alice" || stats[0].Correct != 2 || stats[0].Wrong != 0 {
		t.Fatalf("unexpected leader: %+v", stats[0])
	}
	if stats[1].Username != "bob" || stats[1].Correct != 1 || stats[1].Wrong != 1 {
		t.Fatalf("unexpected second row: %+v", stats[1])
	}
}

func TestArmeniaDayWindow(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Yerevan")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}

	start, end := ArmeniaDayWindow(time.Date(2026, 6, 18, 9, 0, 0, 0, loc), loc)
	if got := start.Format(time.RFC3339); got != "2026-06-18T19:00:00+04:00" {
		t.Fatalf("unexpected start: %s", got)
	}
	if got := end.Format(time.RFC3339); got != "2026-06-19T12:00:00+04:00" {
		t.Fatalf("unexpected end: %s", got)
	}
}

func TestParseWikipediaFixtures(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Yerevan")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}

	page := wikipediaMatchMarker + `
<div class="fleft"><time><div class="fdate">June&#160;18,&#160;2026<span style="display: none;">&#160;(<span class="bday dtstart published updated itvstart">2026-06-18</span>)</span></div><div class="ftime">7:00&#160;p.m. <a href="/wiki/UTC%E2%88%9206:00" title="UTC−06:00">UTC−6</a></div></time></div><table class="fevent"><tbody><tr itemprop="name">
<th class="fhome" itemprop="homeTeam" itemscope="" itemtype="http&#58;//schema.org/SportsTeam"><span itemprop="name"><a href="/wiki/Mexico_national_football_team" title="Mexico national football team">Mexico</a></span></th><th class="fscore"><a href="/wiki/2026_FIFA_World_Cup_Group_A#Mexico_vs_South_Korea" title="2026 FIFA World Cup Group A">Match 28</a></th><th class="faway" itemprop="awayTeam" itemscope="" itemtype="http&#58;//schema.org/SportsTeam"><span itemprop="name"><a href="/wiki/South_Korea_national_football_team" title="South Korea national football team">South Korea</a></span></th></tr></tbody></table><div class="fright"><div itemprop="location" itemscope="" itemtype="http&#58;//schema.org/Place"><span itemprop="name address"><a href="/wiki/Estadio_Akron" title="Estadio Akron">Estadio Akron</a>, <a href="/wiki/Zapopan" title="Zapopan">Zapopan</a></span></div></div></div>`

	fixtures, err := parseWikipediaFixtures(page, loc)
	if err != nil {
		t.Fatalf("parse fixtures: %v", err)
	}
	if len(fixtures) != 1 {
		t.Fatalf("expected 1 fixture, got %d", len(fixtures))
	}

	fixture := fixtures[0]
	if fixture.HomeTeam != "Mexico" || fixture.AwayTeam != "South Korea" {
		t.Fatalf("unexpected teams: %+v", fixture)
	}
	if got := fixture.Kickoff.Format(time.RFC3339); got != "2026-06-19T05:00:00+04:00" {
		t.Fatalf("unexpected kickoff: %s", got)
	}
	if fixture.Venue != "Estadio Akron, Zapopan" {
		t.Fatalf("unexpected venue: %q", fixture.Venue)
	}
	if fixture.HasResult {
		t.Fatalf("did not expect result for unfinished match")
	}
}

func TestParseWikipediaFinishedFixtureScore(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Yerevan")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}

	page := wikipediaMatchMarker + `
<div class="fleft"><time><div class="fdate">June&#160;18,&#160;2026<span style="display: none;">&#160;(<span class="bday dtstart published updated itvstart">2026-06-18</span>)</span></div><div class="ftime">12:00&#160;p.m. <a href="/wiki/UTC%E2%88%9204:00" title="UTC−04:00">UTC−4</a></div></time></div><table class="fevent"><tbody><tr itemprop="name">
<th class="fhome" itemprop="homeTeam" itemscope="" itemtype="http&#58;//schema.org/SportsTeam"><span itemprop="name"><a href="/wiki/Canada_national_soccer_team" title="Canada national soccer team">Canada</a></span></th><th class="fscore"><a href="/wiki/2026_FIFA_World_Cup_Group_B#Canada_vs_Qatar" title="2026 FIFA World Cup Group B">2–1</a></th><th class="faway" itemprop="awayTeam" itemscope="" itemtype="http&#58;//schema.org/SportsTeam"><span itemprop="name"><a href="/wiki/Qatar_national_football_team" title="Qatar national football team">Qatar</a></span></th></tr></tbody></table><div class="fright"><div itemprop="location" itemscope="" itemtype="http&#58;//schema.org/Place"><span itemprop="name address"><a href="/wiki/BC_Place" title="BC Place">BC Place</a>, <a href="/wiki/Vancouver" title="Vancouver">Vancouver</a></span></div></div></div>`

	fixtures, err := parseWikipediaFixtures(page, loc)
	if err != nil {
		t.Fatalf("parse fixtures: %v", err)
	}
	if len(fixtures) != 1 {
		t.Fatalf("expected 1 fixture, got %d", len(fixtures))
	}
	if !fixtures[0].HasResult || fixtures[0].ScoreHome != 2 || fixtures[0].ScoreAway != 1 {
		t.Fatalf("unexpected parsed result: %+v", fixtures[0])
	}
}

func TestFormatCountdown(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Yerevan")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}

	from := time.Date(2026, 6, 18, 10, 20, 0, 0, loc)
	to := time.Date(2026, 6, 19, 2, 0, 0, 0, loc)
	if got := formatCountdown(from, to); got != "15h 40m" {
		t.Fatalf("unexpected countdown: %s", got)
	}
}
