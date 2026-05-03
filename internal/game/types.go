package game

// GameState represents the current phase of a room.
type GameState int

const (
	StateWaiting  GameState = iota // lobby open
	StatePlaying                   // game in progress
	StateFinished                  // game over
)

// Message is a JSON-serializable WebSocket message.
type Message struct {
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
}

// SnapshotQuestion mirrors repository.SnapshotQuestion but lives in game layer.
type SnapshotQuestion struct {
	ID           int              `json:"snapshot_id"`
	OriginalID   int              `json:"original_id"`
	Position     int              `json:"position"`
	Content      string           `json:"content"`
	ImageURL     string           `json:"image_url"`
	YoutubeURL   string           `json:"youtube_url"`
	YoutubeStart int              `json:"youtube_start"`
	YoutubeEnd   int              `json:"youtube_end"`
	TimeLimit    int              `json:"time_limit"`
	ScoreType    string           `json:"score_type"`
	StaticScore  int              `json:"static_score"`
	Answers      []SnapshotAnswer `json:"answers"`
}

type SnapshotAnswer struct {
	ID        int    `json:"id"`
	Text      string `json:"text"`
	IsCorrect bool   `json:"is_correct,omitempty"` // hidden from students until answered
}

// PlayerState tracks a player's progress during a session.
type PlayerState struct {
	PlayerID    int
	Client      *Client
	TotalScore  int
	QuestionsAnswered int
}

// AnswerMsg is sent by a student over WebSocket.
type AnswerMsg struct {
	Type        string `json:"type"`
	Answer      string `json:"answer"`
	TimeTakenMs int    `json:"time_taken_ms"`
	QuestionID  int    `json:"question_id"` // snapshot question id
}
