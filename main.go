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

type SpotifyClient struct {
	client      *http.Client
	AccessToken string
}

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

func NewSpotifyClient(accessToken string) *SpotifyClient {
	return &SpotifyClient{
		client:      &http.Client{},
		AccessToken: accessToken,
	}
}

func (c *SpotifyClient) makeRequest(method, endpoint string, body io.Reader) (*http.Response, error) {
	url := "https://api.spotify.com/v1" + endpoint

	// Create the HTTP request
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	// Set the headers
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	// Make the request using HTTP client
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}
	return resp, nil
}

func (c *SpotifyClient) callMakeRequestAndUnmarshal(method, endpoint string, body io.Reader, result any) error {

	// Make the request
	resp, err := c.makeRequest(method, endpoint, body)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	// Read the response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	// Unmarshal JSON into the result struct
	err = json.Unmarshal(responseBody, &result)
	if err != nil {
		log.Fatal("Error unmarshalling response %v", err)
	}
	return nil
}

func getUserInput(prompt string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(prompt)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
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

func (c *SpotifyClient) SearchForTracks() (string, error) {

	userInput, err := getUserInput("Enter track to search: ")
	if err != nil {
		return "", err
	}

	params := url.Values{}
	params.Set("q", userInput)
	params.Set("type", "track")
	method := "GET"
	endpoint := "/search?" + params.Encode()

	// Unmarshall the body into the struct you created
	var searchTracksResponse GetAllTracksResponse
	err = c.callMakeRequestAndUnmarshal(method, endpoint, nil, &searchTracksResponse)
	if err != nil {
		return "", err
	}

	for i, track := range searchTracksResponse.Tracks.Items {
		fmt.Printf("[%d] Track: %s by %s\n", i, track.Name, track.Artists[0].Name)
	}

	// Get index of track from search results - select which track you want
	trackIDInput, err := getUserInput("Enter the index of the playlist you want: ")
	if err != nil {
		return "", err // Which format of error is better?
	}

	trackIndex, err := strconv.Atoi(trackIDInput)
	if err != nil {
		log.Fatal(err)
	}

	if trackIndex >= len(searchTracksResponse.Tracks.Items) || trackIndex < 0 {
		log.Fatal("Index out of bounds")
	}
	requestedTrack := searchTracksResponse.Tracks.Items[trackIndex]
	return requestedTrack.URI, nil
}

// Get an existing playlist I created in Spotify
func (c *SpotifyClient) GetAllPlaylists() ([]Playlist, error) {

	// Initializing the required parameters to get all my playlists
	method := "GET"
	endpoint := "/users/ragyakaul/playlists"

	// Unmarshall the body into the struct you created
	var getAllPlaylistsResponse GetAllPlaylistsResponse
	err := c.callMakeRequestAndUnmarshal(method, endpoint, nil, &getAllPlaylistsResponse)
	if err != nil {
		return nil, err
	}
	return getAllPlaylistsResponse.Items, nil
}

func selectPlaylist(playlists []Playlist) string {

	// Print the results
	for i, item := range playlists {
		fmt.Printf("[%d] Playlist Name: %s Playlist ID: %s\n", i, item.Name, item.ID)
	}

	userInput, err := getUserInput("Enter the index of the playlist you want: ")

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

func (c *SpotifyClient) AddTrackToPlaylist(trackURI string, playlistID string) {

	// Initializing the required parameters to add track to my playlist
	method := "POST"
	endpoint := fmt.Sprintf("/playlists/%s/tracks", playlistID)

	reqBody := fmt.Sprintf(`{"uris": ["%s"]}`, trackURI)
	bodyReader := strings.NewReader(reqBody)

	// Creating the request object
	c.makeRequest(method, endpoint, bodyReader)
}

func (c *SpotifyClient) GetTrackFromPlaylist(playlistID string) string {

	method := "GET"
	endpoint := fmt.Sprintf("/playlists/%s", playlistID)

	var getTracksFromPlaylistResponse GetTracksFromPlaylistResponse
	err := c.callMakeRequestAndUnmarshal(method, endpoint, nil, &getTracksFromPlaylistResponse)

	for i, item := range getTracksFromPlaylistResponse.Tracks.Items {
		fmt.Printf("[%d] Track Name: %s Artist: %s\n", i, item.Track.Name, item.Track.Artists[0].Name)
	}

	index, err := getUserInput("Enter the index of the track you want: ")
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

func (c *SpotifyClient) RemoveTrackFromPlaylist(playlistID string, trackURI string) {

	// Initializing the required parameters to add track to my playlist
	method := "DELETE"
	endpoint := fmt.Sprintf("/playlists/%s/tracks", playlistID)

	reqBody := fmt.Sprintf(`{"tracks":[{"uri": "%s"}]}`, trackURI)
	bodyReader := strings.NewReader(reqBody)

	c.makeRequest(method, endpoint, bodyReader)
}

func main() {
	client := http.Client{}

	accessToken := getToken(&client)

	spotifyClient := NewSpotifyClient(accessToken)

	playlists, err := spotifyClient.GetAllPlaylists()
	if err != nil {
		return
	}

	playlistID := selectPlaylist(playlists)

	actionInput, err := getUserInput("Do you want to add or remove a track from your playlist?")
	if err != nil {
		return
	}

	if actionInput == "add" {
		trackURI, err := spotifyClient.SearchForTracks()
		if err != nil {
			return
		}
		spotifyClient.AddTrackToPlaylist(trackURI, playlistID)

	} else if actionInput == "remove" {
		trackURI := spotifyClient.GetTrackFromPlaylist(playlistID)
		spotifyClient.RemoveTrackFromPlaylist(playlistID, trackURI)
	}

}
