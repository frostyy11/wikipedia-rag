### Ollama
Make sure Ollama is running:
`ollama serve`

Pull a model (if you haven't already):
`ollama pull mistral`

### Compilation
1. Create a directory and save the code:
```
mkdir rag-system
cd rag-system
```
2. Save the code to a file named `main.go`
3. Initialize the Go module: `go mod init rag-system`
4. Compile the application: `go build -o rag`
5. Run the compiled executable: `./rag`
