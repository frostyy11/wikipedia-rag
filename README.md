Install dependencies:
`go mod init rag-system`

Make sure Ollama is running:
`ollama serve`

Pull a model (if you haven't already):
`ollama pull mistral`

Run the app:
`go run main.go` or `go run main.go --model mistral`
