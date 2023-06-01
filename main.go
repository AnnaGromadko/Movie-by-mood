package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	openai "github.com/sashabaranov/go-openai"
)

type Mood string

const (
	Angry Mood = "angry"
	Happy Mood = "happy"
)

type Movie struct {
	Title string `json:"title"`
}

func main() {
	router := gin.Default()

	router.GET("/movie/:mood", getMovieByMood)

	router.Run(":8080")
}

func getMovieByMood(c *gin.Context) {
	mood := c.Param("mood")
	if mood != string(Angry) && mood != string(Happy) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid mood",
		})
		return
	}

	movie := getMovieFromLLM(mood)
	if movie == "" {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "no movie found for the given mood",
		})
		return
	}

	extendedInfo := getExtendedMovieInfoFromAPI(movie)
	c.JSON(http.StatusOK, gin.H{
		"mood":          mood,
		"recommended":   movie,
		"extended_info": extendedInfo,
	})
}

func getExtendedMovieInfoFromAPI(movie string) string {
	url := "http://www.omdbapi.com/?i=tt3896198&apikey=&plot=full&t=" + movie
	res, err := http.Get(url)
	if err != nil {
		fmt.Printf("error making http request: %s\n", err)
		return ""
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		fmt.Printf("unexpected response status code: %d\n", res.StatusCode)
		return ""
	}

	var response struct {
		Title       string `json:"Title"`
		Plot        string `json:"Plot"`
		ReleaseYear string `json:"Year"`
		// Add more fields as needed to capture the extended movie information
	}

	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		fmt.Printf("error decoding response: %s\n", err)
		return ""
	}

	extendedInfo := fmt.Sprintf("Title: %s\nPlot: %s\nRelease Year: %s", response.Title, response.Plot, response.ReleaseYear)

	return extendedInfo
}

func getMovieFromLLM(mood string) string {
	client := openai.NewClient("apikey")
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role: openai.ChatMessageRoleUser,
					Content: fmt.Sprintf(`Send me one movie title based on this mood: %s
					Provide it in JSON format in an array with the following key: 
					"title".`, mood),
				},
			},
		},
	)
	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		return ""
	}

	var movies []Movie
	err = json.Unmarshal([]byte(resp.Choices[0].Message.Content), &movies)
	if err != nil {
		fmt.Printf("Error parsing movie JSON: %v\n", err)
		return ""
	}

	if len(movies) > 0 {
		return movies[0].Title
	}

	return ""
}
