package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

type WikiSearchResponse struct {
	Query struct {
		Search []struct {
			Title   string `json:"title"`
			Snippet string `json:"snippet"`
		} `json:"search"`
	} `json:"query"`
}

type WikiPageResponse struct {
	Query struct {
		Pages map[string]struct {
			Title   string `json:"title"`
			Extract string `json:"extract"`
		} `json:"pages"`
	} `json:"query"`
}

func searchWikipedia(query string) ([]string, error) {
	baseURL := "https://en.wikipedia.org/w/api.php"
	params := url.Values{}
	params.Add("action", "query")
	params.Add("list", "search")
	params.Add("srsearch", query)
	params.Add("format", "json")
	params.Add("srlimit", "3")

	client := &http.Client{}
	req, err := http.NewRequest("GET", baseURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "RAG-CLI/1.0 (Educational Project)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result WikiSearchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	var titles []string
	for _, page := range result.Query.Search {
		titles = append(titles, page.Title)
	}

	return titles, nil
}

func getWikipediaContent(title string) (string, error) {
	baseURL := "https://en.wikipedia.org/w/api.php"
	params := url.Values{}
	params.Add("action", "query")
	params.Add("titles", title)
	params.Add("prop", "extracts")
	params.Add("explaintext", "true")
	params.Add("exsectionformat", "plain")
	params.Add("format", "json")

	client := &http.Client{}
	req, err := http.NewRequest("GET", baseURL+"?"+params.Encode(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "RAG-CLI/1.0 (Educational Project)")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result WikiPageResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	for _, page := range result.Query.Pages {
		if len(page.Extract) > 2000 {
			return page.Extract[:2000], nil
		}
		return page.Extract, nil
	}

	return "", fmt.Errorf("no content found")
}

func queryTGPT(prompt string) error {
	cmd := exec.Command("tgpt", prompt)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: rag-cli <your question>")
		fmt.Println("Example: rag-cli \"What is quantum computing?\"")
		os.Exit(1)
	}

	question := strings.Join(os.Args[1:], " ")

	fmt.Printf("🔍 Searching Wikipedia for: %s\n\n", question)

	titles, err := searchWikipedia(question)
	if err != nil {
		fmt.Printf("Error searching Wikipedia: %v\n", err)
		os.Exit(1)
	}

	if len(titles) == 0 {
		fmt.Println("No Wikipedia articles found. Asking tgpt directly...")
		if err := queryTGPT(question); err != nil {
			fmt.Printf("Error querying tgpt: %v\n", err)
			os.Exit(1)
		}
		return
	}

	fmt.Printf("📚 Found %d relevant articles\n", len(titles))
	var context strings.Builder
	context.WriteString("Context from Wikipedia:\n\n")

	for i, title := range titles {
		fmt.Printf("   %d. %s\n", i+1, title)
		content, err := getWikipediaContent(title)
		if err != nil {
			fmt.Printf("      (couldn't fetch content)\n")
			continue
		}
		context.WriteString(fmt.Sprintf("Article: %s\n%s\n\n", title, content))
	}

	fmt.Println("\n🤖 Generating answer with tgpt...\n")

	prompt := fmt.Sprintf("%s\n\nBased on the above context, please answer this question: %s", 
		context.String(), question)

	if err := queryTGPT(prompt); err != nil {
		fmt.Printf("\nError querying tgpt: %v\n", err)
		os.Exit(1)
	}
}
