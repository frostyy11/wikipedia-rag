package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const (
	ollamaURL    = "http://localhost:11434/api/generate"
	wikipediaURL = "https://en.wikipedia.org/w/api.php"
)

type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type OllamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

type WikipediaResponse struct {
	Query struct {
		Search []struct {
			Title   string `json:"title"`
			Snippet string `json:"snippet"`
		} `json:"search"`
	} `json:"query"`
}

type WikipediaPageResponse struct {
	Query struct {
		Pages map[string]struct {
			Extract string `json:"extract"`
		} `json:"pages"`
	} `json:"query"`
}

func searchWikipedia(query string) ([]string, error) {
	params := url.Values{}
	params.Add("action", "query")
	params.Add("list", "search")
	params.Add("srsearch", query)
	params.Add("format", "json")
	params.Add("srlimit", "3")
	params.Add("utf8", "1")

	fullURL := wikipediaURL + "?" + params.Encode()
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "RAG-CLI-App/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Wikipedia API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var wikiResp WikipediaResponse
	if err := json.Unmarshal(body, &wikiResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %v. Body: %s", err, string(body[:min(200, len(body))]))
	}

	var titles []string
	for _, result := range wikiResp.Query.Search {
		titles = append(titles, result.Title)
	}

	return titles, nil
}

func getWikipediaContent(title string) (string, error) {
	params := url.Values{}
	params.Add("action", "query")
	params.Add("titles", title)
	params.Add("prop", "extracts")
	params.Add("explaintext", "true")
	params.Add("exintro", "true")
	params.Add("format", "json")
	params.Add("utf8", "1")

	fullURL := wikipediaURL + "?" + params.Encode()
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "RAG-CLI-App/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Wikipedia API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var wikiResp WikipediaPageResponse
	if err := json.Unmarshal(body, &wikiResp); err != nil {
		return "", fmt.Errorf("failed to parse JSON: %v", err)
	}

	for _, page := range wikiResp.Query.Pages {
		return page.Extract, nil
	}

	return "", fmt.Errorf("no content found")
}

func queryOllama(model, prompt string) (string, error) {
	reqBody := OllamaRequest{
		Model:  model,
		Prompt: prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(ollamaURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var ollamaResp OllamaResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return "", err
	}

	return ollamaResp.Response, nil
}

func performRAG(question, model string) error {
	fmt.Println("\n🔍 Searching Wikipedia...")
	titles, err := searchWikipedia(question)
	if err != nil {
		return fmt.Errorf("Wikipedia search error: %v", err)
	}

	if len(titles) == 0 {
		return fmt.Errorf("no Wikipedia results found")
	}

	fmt.Printf("📚 Found %d relevant articles\n", len(titles))

	var context strings.Builder
	for i, title := range titles {
		fmt.Printf("📖 Retrieving: %s\n", title)
		content, err := getWikipediaContent(title)
		if err != nil {
			fmt.Printf("⚠️  Error retrieving %s: %v\n", title, err)
			continue
		}

		// Limit content length
		if len(content) > 1000 {
			content = content[:1000] + "..."
		}

		context.WriteString(fmt.Sprintf("\n--- Article %d: %s ---\n%s\n", i+1, title, content))
	}

	prompt := fmt.Sprintf(`Based on the following Wikipedia articles, answer this question: %s

Context:
%s

Answer the question based on the context provided. If the context doesn't contain enough information, say so.

Answer:`, question, context.String())

	fmt.Println("\n🤖 Generating answer with Ollama...")
	answer, err := queryOllama(model, prompt)
	if err != nil {
		return fmt.Errorf("Ollama error: %v", err)
	}

	fmt.Println("\n📝 Answer:")
	fmt.Println(strings.TrimSpace(answer))

	return nil
}

func main() {
	fmt.Println("=================================")
	fmt.Println("RAG System: Ollama + Wikipedia")
	fmt.Println("=================================")

	model := "llama2"
	if len(os.Args) > 1 && os.Args[1] == "--model" && len(os.Args) > 2 {
		model = os.Args[2]
	}

	fmt.Printf("\nUsing Ollama model: %s\n", model)
	fmt.Println("\nType your questions (or 'exit' to quit)")
	fmt.Println("Example: What is quantum computing?\n")

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("\n> ")
		if !scanner.Scan() {
			break
		}

		question := strings.TrimSpace(scanner.Text())

		if question == "" {
			continue
		}

		if strings.ToLower(question) == "exit" {
			fmt.Println("Goodbye!")
			break
		}

		if err := performRAG(question, model); err != nil {
			fmt.Printf("\n❌ Error: %v\n", err)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
