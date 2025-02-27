package sdk

type Response struct {
	Choices []Choice `json:"choices"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Object  string   `json:"object"`
}

type Choice struct {
	FinishReason string  `json:"finish_reason"`
	Index        int     `json:"index"`
	Message      Message `json:"message"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type RequestPayload struct {
	Model             string    `json:"model"`
	Messages          []Message `json:"messages"`
	Stream            bool      `json:"stream"`
	RepetitionPenalty float64   `json:"repetition_penalty"`
}
