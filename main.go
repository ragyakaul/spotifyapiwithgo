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
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"bearer"`
	ExpiresIn   int    `json:"expires_in"`
}

type GetAllTracksResponse struct {
	Tracks struct {
		Items []struct {
			Name    string `json:"name"`
			URI     string `json:"uri"`
			Artists []struct {
				Name string `json:"name"`
			} `json:"artists"`
		} `json:"items"`
	} `json:"tracks"`
}

type Playlist struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type GetAllPlaylistsResponse struct {
	Items []Playlist `json:"items"`
}

type GetTracksFromPlaylistResponse struct {
	Tracks struct {
		Items []struct {
			Track struct {
				Name    string `json:"name"`
				URI     string `json:"uri"`
				Type    string `json:"type"`
				Artists []struct {
					Name string `json:"name"`
				} `json:"artists"`
			} `json:"track"`
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
	scopeToAlterPlaylist := os.Getenv("SPOTIFY_ALTER_PLAYLIST")
	// Encode credentials
	authHeader := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))

	// Setting the body for the upcoming request
	// requestBody := "grant_type=client_credentials"
	reqBody := url.Values{}
	reqBody.Set("grant_type", "authorization_code")
	reqBody.Set("code", scopeToAlterPlaylist)
	reqBody.Set("redirect_uri", "http://localhost:8000")
	requestBody := reqBody.Encode()

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

func searchForTracks(client *http.Client, accessToken string) string {

	// Ask user for search input
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter track to search: ")
	userInput, _ := reader.ReadString('\n')
	userInput = strings.TrimSpace(userInput)

	// Create the search URL
	params := url.Values{}
	params.Set("q", userInput)
	params.Set("type", "track")

	// Initializing the required parameters to perform a search
	method := "GET"
	url := "https://api.spotify.com/v1/search?" + params.Encode()

	// Creating the request object
	req, err := http.NewRequest(method, url, nil)

	// Set headers
	req.Header.Set("Authorization", "Bearer "+accessToken)

	// Make a search request to the Spotify API
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// Read the search response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	// Unmarshall the body into the struct you created
	var searchTracksResponse GetAllTracksResponse
	err = json.Unmarshal(responseBody, &searchTracksResponse)
	if err != nil {
		log.Fatal("Error unmarshalling search tracks response:", err)
	}

	// Print the results
	for i, track := range searchTracksResponse.Tracks.Items {
		fmt.Printf("[%d] Track: %s by %s\n", i, track.Name, track.Artists[0].Name)
	}

	// Get index of track from search results - select which track you want
	fmt.Print("Enter the index of the playlist you want: ")
	trackIDInput, _ := reader.ReadString('\n')
	trackIDInput = strings.TrimSpace(trackIDInput)

	trackIndex, err := strconv.Atoi(trackIDInput)
	if err != nil {
		log.Fatal(err)
	}

	if trackIndex >= len(searchTracksResponse.Tracks.Items) || trackIndex < 0 {
		log.Fatal("Index out of bounds")
	}
	requestedTrack := searchTracksResponse.Tracks.Items[trackIndex]
	return requestedTrack.URI
}

// Get an existing playlist I created in Spotify
func getAllPlaylists(client *http.Client, accessToken string) []Playlist {

	// Initializing the required parameters to get all my playlists
	method := "GET"
	url := "https://api.spotify.com/v1/users/ragyakaul/playlists"

	// Creating the request object
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		log.Fatal("Error creating request:", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+accessToken)

	// Make a request to get your playlists from the Spotify API
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// Read the search response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	// Unmarshall the body into the struct you created
	var getAllPlaylistsResponse GetAllPlaylistsResponse
	err = json.Unmarshal(responseBody, &getAllPlaylistsResponse)
	if err != nil {
		log.Fatal("Error unmarshalling get all playlists response:", err)
	}

	return getAllPlaylistsResponse.Items
}

func selectPlaylist(playlists []Playlist) string {

	// Print the results
	for i, item := range playlists {
		fmt.Printf("[%d] Playlist Name: %s Playlist ID: %s\n", i, item.Name, item.ID)
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter the index of the playlist you want: ")
	userInput, _ := reader.ReadString('\n')
	userInput = strings.TrimSpace(userInput)

	playlistIndex, err := strconv.Atoi(userInput)
	if err != nil {
		log.Fatal(err)
	}

	if playlistIndex >= len(playlists) || playlistIndex < 0 {
		log.Fatal("Index out of bounds")
	}
	requestedPlaylist := playlists[playlistIndex]
	return requestedPlaylist.ID
}

func addTrackToPlaylist(client *http.Client, accessToken string, trackURI string, playlistID string) {

	// Initializing the required parameters to add track to my playlist
	method := "POST"
	url := fmt.Sprintf("https://api.spotify.com/v1/playlists/%s/tracks", playlistID)

	reqBody := fmt.Sprintf(`{"uris": ["%s"]}`, trackURI)
	bodyReader := strings.NewReader(reqBody)

	// Creating the request object
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		log.Fatal("Error creating request:", err)
	}
	fmt.Println("URL: ", url)

	// Set headers
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	// Make a req to add track to your playlist using the Spotify API
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	statusResponse := resp.StatusCode
	if statusResponse != 201 {
		respBody, _ := io.ReadAll(resp.Body)
		fmt.Println(string(respBody))
		log.Fatal("Failed to add track to playlists")

	}
}

func getTrackFromPlaylist(client *http.Client, accessToken string, playlistID string) string {

	// Initializing the required parameters to get a track from my playlist
	method := "GET"
	url := fmt.Sprintf("https://api.spotify.com/v1/playlists/%s", playlistID)

	// Creating the request object
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		log.Fatal("Error creating request:", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// Read the search response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	statusResponse := resp.StatusCode
	if statusResponse != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		fmt.Println(string(respBody))
		log.Fatal("Failed to get track from playlist")
	}

	var getTracksFromPlaylistResponse GetTracksFromPlaylistResponse
	err = json.Unmarshal(responseBody, &getTracksFromPlaylistResponse)
	if err != nil {
		log.Fatal("Error unmarshalling get track from playlist response")
	}

	for i, item := range getTracksFromPlaylistResponse.Tracks.Items {
		fmt.Printf("[%d] Track Name: %s Artist: %s\n", i, item.Track.Name, item.Track.Artists[0].Name)
	}

	fmt.Print("Enter the index of the track you want: ")
	reader := bufio.NewReader(os.Stdin)
	index, _ := reader.ReadString('\n')
	index = strings.TrimSpace(index)
	indexInt, err := strconv.Atoi(index)
	if err != nil {
		log.Fatal(err)
	}

	if indexInt >= len(getTracksFromPlaylistResponse.Tracks.Items) || indexInt < 0 {
		log.Fatal("Index out of bounds")
	}
	requestedTrack := getTracksFromPlaylistResponse.Tracks.Items[indexInt]
	return requestedTrack.Track.URI

}

func removeTrackFromPlaylist(client *http.Client, accessToken string, playlistID string, trackURI string) {

	// Initializing the required parameters to add track to my playlist
	method := "DELETE"
	url := fmt.Sprintf("https://api.spotify.com/v1/playlists/%s/tracks", playlistID)

	reqBody := fmt.Sprintf(`{"tracks":[{"uri": "%s"}]}`, trackURI)
	bodyReader := strings.NewReader(reqBody)

	// Creating the request object
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		log.Fatal("Error creating request:", err)
	}
	fmt.Println("URL: ", url)

	// Set headers
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	// Make a req to add track to your playlist using the Spotify API
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Println(trackURI)
	statusResponse := resp.StatusCode
	if statusResponse != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		fmt.Println(string(respBody))
		log.Fatal("Failed to delete track from playlist")
	}

}

func main() {
	client := http.Client{}

	accessToken := getToken(&client)

	playlists := getAllPlaylists(&client, accessToken)

	playlistID := selectPlaylist(playlists)

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Do you want to add or remove a track from your playlist?")
	actionInput, _ := reader.ReadString('\n')
	actionInput = strings.TrimSpace(actionInput)

	if actionInput == "add" {
		trackURI := searchForTracks(&client, accessToken)
		addTrackToPlaylist(&client, accessToken, trackURI, playlistID)

	} else if actionInput == "remove" {
		trackURI := getTrackFromPlaylist(&client, accessToken, playlistID)
		removeTrackFromPlaylist(&client, accessToken, playlistID, trackURI)
	}

}
