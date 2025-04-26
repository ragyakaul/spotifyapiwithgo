package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"bearer"`
	ExpiresIn   int    `json:"expires_in"`
}

type SearchTracksResponse struct {
	Tracks struct {
		Items []struct {
			Name    string `json:"name"`
			ID      string `json:"id"`
			Artists []struct {
				Name string `json:"name"`
			} `json:"artists"`
		} `json:"items"`
	} `json:"tracks"`
}

func getToken(client *http.Client) string {

	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}
	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")
	// Encode credentials
	authHeader := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))

	// Setting the body for the upcoming request
	requestBody := "grant_type=client_credentials"

	// Initializing the required parameters for getting the token
	tokenMethod := "POST"
	tokenUrl := "https://accounts.spotify.com/api/token"
	bodyReader := strings.NewReader(requestBody)

	// Creating the token object
	req, err := http.NewRequest(tokenMethod, tokenUrl, bodyReader)
	if err != nil {
		log.Fatal("Error creating request:", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Basic "+authHeader)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	//  Make a request to the Spotify API
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	// Unmarshal the response body into a Go struct
	var tokenResponse TokenResponse
	err = json.Unmarshal(responseBody, &tokenResponse)
	if err != nil {
		log.Fatal("Error unmarshalling response:", err)
	}

	// Print the access token
	fmt.Println("Access Token:", tokenResponse.AccessToken)
	return tokenResponse.AccessToken
}

func searchForTracks(client *http.Client, accessToken string) {

	// Ask user for search input
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter track to search: ")
	userInput, _ := reader.ReadString('\n')
	userInput = strings.TrimSpace(userInput)

	// Create the search URL
	searchParams := url.Values{}
	searchParams.Set("q", userInput)
	searchParams.Set("type", "track")

	// Initializing the required parameters to perform a search
	searchMethod := "GET"
	searchUrl := "https://api.spotify.com/v1/search?" + searchParams.Encode()
	fmt.Println("searchUrl: ", searchUrl)

	// Creating the request object
	searchReq, err := http.NewRequest(searchMethod, searchUrl, nil)

	// Set headers
	searchReq.Header.Set("Authorization", "Bearer "+accessToken)

	// Make a search request to the Spotify API
	searchResp, err := client.Do(searchReq)
	if err != nil {
		panic(err)
	}
	defer searchResp.Body.Close()

	// Read the search response body
	searchResponseBody, err := io.ReadAll(searchResp.Body)
	if err != nil {
		log.Fatal(err)
	}

	// Unmarshall the body into the struct you created
	var searchTracksResponse SearchTracksResponse
	err = json.Unmarshal(searchResponseBody, &searchTracksResponse)
	if err != nil {
		log.Fatal("Error unmarshalling search tracks response:", err)
	}

	// Print the results
	fmt.Println("Gets here")
	fmt.Println("Is slice empty? ", searchTracksResponse.Tracks.Items)
	for _, track := range searchTracksResponse.Tracks.Items {
		fmt.Println("Gets here")
		fmt.Printf("Track: %s by %s\n", track.Name, track.Artists[0].Name)
	}

}

func main() {
	client := http.Client{}
	accessToken := getToken(&client)
	searchForTracks(&client, accessToken)
}
