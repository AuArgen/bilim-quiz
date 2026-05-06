package repository

import "time"

type Teacher struct {
	ID        int
	GoogleID  string
	Email     string
	Name      string
	AvatarURL string
	Language  string
	GeminiKey string
	CreatedAt time.Time
}

type Game struct {
	ID            int
	TeacherID     int
	Title         string
	Description   string
	QuestionCount int
	ShareToken    string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Question struct {
	ID           int
	GameID       int
	Position     int
	Content      string
	ImageURL     string
	YoutubeURL   string
	YoutubeStart int
	YoutubeEnd   int
	TimeLimit    int
	ScoreType    string
	StaticScore  int
	Answers      []Answer
	CreatedAt    time.Time
}

type Answer struct {
	ID         int
	QuestionID int
	Text       string
	IsCorrect  bool
}

type GameSession struct {
	ID           int
	GameID       int
	TeacherID    int
	PinCode      string
	Status       string
	TotalPlayers int
	StartedAt    *time.Time
	FinishedAt   *time.Time
	CreatedAt    time.Time
}

type SessionPlayer struct {
	ID         int
	SessionID  int
	Nickname   string
	Avatar     string
	FinalScore int
	FinishedAt *time.Time
}

type SnapshotQuestion struct {
	ID           int
	SessionID    int
	OriginalID   int
	Position     int
	Content      string
	ImageURL     string
	YoutubeURL   string
	YoutubeStart int
	YoutubeEnd   int
	TimeLimit    int
	ScoreType    string
	StaticScore  int
	Answers      []SnapshotAnswer
}

type SnapshotAnswer struct {
	ID                 int
	SnapshotQuestionID int
	Text               string
	IsCorrect          bool
}

type PlayerAnswer struct {
	ID                 int
	PlayerID           int
	SnapshotQuestionID int
	SelectedAnswerText string
	IsCorrect          bool
	EarnedPoints       int
	TimeTakenMs        int
	AnsweredAt         time.Time
}

type TeacherStats struct {
	TotalGames    int
	TotalSessions int
	TotalPlayers  int
}

type SessionRating struct {
	ID        int
	SessionID int
	PlayerID  int
	Stars     int
	Comment   string
	Nickname  string
	Avatar    string
	CreatedAt time.Time
}

type RatingStats struct {
	AvgStars float64
	Count    int
}
