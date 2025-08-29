package data

type PromptResponse struct {
	Title    string
	Message  string
	Callback func(bool)
}
