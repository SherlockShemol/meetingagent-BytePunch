package models

// Meeting represents a meeting entity
type Meeting struct {
	ID        string      `json:"id"`
	Content   interface{} `json:"content"`
	Summary   string      `json:"summary"`
	CreatedAt string      `json:"created_at"`
}

// PostMeetingResponse represents the response for creating a meeting
type PostMeetingResponse struct {
	ID string `json:"id"`
}

// GetMeetingsResponse represents the response for listing meetings
type GetMeetingsResponse struct {
	Meetings []Meeting `json:"meetings"`
}

// ChatMessage represents a chat message in the SSE stream
type ChatMessage struct {
	Data string `json:"data"`
}
