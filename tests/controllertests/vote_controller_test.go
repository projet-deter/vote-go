package controllertests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gorilla/mux"
	"github.com/m2fof/vote/api/models"
	"gopkg.in/go-playground/assert.v1"
)

func TestCreateVote(t *testing.T) {

	err := refreshUserAndVoteTable()
	if err != nil {
		log.Fatal(err)
	}
	user, err := seedOneUser()
	if err != nil {
		log.Fatalf("Cannot seed user %v\n", err)
	}
	token, err := server.SignIn(user.Email, "password") //Note the password in the database is already hashed, we want unhashed
	if err != nil {
		log.Fatalf("cannot login: %v\n", err)
	}
	tokenString := fmt.Sprintf("Bearer %v", token)

	samples := []struct {
		inputJSON    string
		statusCode   int
		title        string
		description  string
		author_id    uint32
		tokenGiven   string
		errorMessage string
	}{
		{
			inputJSON:    `{"title":"The title", "Desc": "Description", "author_id": 1}`,
			statusCode:   201,
			tokenGiven:   tokenString,
			title:        "The title",
			description:  "the vote number one",
			author_id:    user.ID,
			errorMessage: "",
		},
		{
			inputJSON:    `{"title":"The title", "Desc": "Description", "author_id": 1}`,
			statusCode:   500,
			tokenGiven:   tokenString,
			errorMessage: "Title Already Taken",
		},
		{
			// When no token is passed
			inputJSON:    `{"title":"When no token is passed", "Desc": "Description", "author_id": 1}`,
			statusCode:   401,
			tokenGiven:   "",
			errorMessage: "Unauthorized",
		},
		{
			// When incorrect token is passed
			inputJSON:    `{"title":"When incorrect token is passed", "Desc": "Description", "author_id": 1}`,
			statusCode:   401,
			tokenGiven:   "This is an incorrect token",
			errorMessage: "Unauthorized",
		},
		{
			inputJSON:    `{"title": "", "Desc": "Description", "author_id": 1}`,
			statusCode:   422,
			tokenGiven:   tokenString,
			errorMessage: "Required Title",
		},
		{
			inputJSON:    `{"title": "This is a title", "Desc": "", "author_id": 1}`,
			statusCode:   422,
			tokenGiven:   tokenString,
			errorMessage: "Required Decription",
		},
		{
			inputJSON:    `{"title": "This is an awesome title", "Desc": "Description"}`,
			statusCode:   422,
			tokenGiven:   tokenString,
			errorMessage: "Required Author",
		},
		{
			// When user 2 uses user 1 token
			inputJSON:    `{"title": "This is an awesome title", "Desc": "Description", "author_id": 2}`,
			statusCode:   401,
			tokenGiven:   tokenString,
			errorMessage: "Unauthorized",
		},
	}
	for _, v := range samples {

		req, err := http.NewRequest("Vote", "/votes", bytes.NewBufferString(v.inputJSON))
		if err != nil {
			t.Errorf("this is the error: %v\n", err)
		}
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(server.CreateVote)

		req.Header.Set("Authorization", v.tokenGiven)
		handler.ServeHTTP(rr, req)

		responseMap := make(map[string]interface{})
		err = json.Unmarshal([]byte(rr.Body.String()), &responseMap)
		if err != nil {
			fmt.Printf("Cannot convert to json: %v", err)
		}
		assert.Equal(t, rr.Code, v.statusCode)
		if v.statusCode == 201 {
			assert.Equal(t, responseMap["title"], v.title)
			assert.Equal(t, responseMap["Desc"], v.description)
			assert.Equal(t, responseMap["author_id"], float64(v.author_id)) //just for both ids to have the same type
		}
		if v.statusCode == 401 || v.statusCode == 422 || v.statusCode == 500 && v.errorMessage != "" {
			assert.Equal(t, responseMap["error"], v.errorMessage)
		}
	}
}

func TestGetVotes(t *testing.T) {

	err := refreshUserAndVoteTable()
	if err != nil {
		log.Fatal(err)
	}
	_, _, err = seedUsersAndVotes()
	if err != nil {
		log.Fatal(err)
	}

	req, err := http.NewRequest("GET", "/votes", nil)
	if err != nil {
		t.Errorf("this is the error: %v\n", err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.GetVotes)
	handler.ServeHTTP(rr, req)

	var votes []models.Vote
	err = json.Unmarshal([]byte(rr.Body.String()), &votes)

	assert.Equal(t, rr.Code, http.StatusOK)
	assert.Equal(t, len(votes), 2)
}
func TestGetVoteByID(t *testing.T) {

	err := refreshUserAndVoteTable()
	if err != nil {
		log.Fatal(err)
	}
	vote, err := seedOneUserAndOneVote()
	if err != nil {
		log.Fatal(err)
	}
	voteSample := []struct {
		id           string
		statusCode   int
		title        string
		desc         string
		author_id    uint32
		errorMessage string
	}{
		{
			id:         strconv.Itoa(int(vote.ID)),
			statusCode: 200,
			title:      vote.Title,
			desc:       vote.Desc,
			author_id:  vote.AuthorID,
		},
		{
			id:         "unknwon",
			statusCode: 400,
		},
	}
	for _, v := range voteSample {

		req, err := http.NewRequest("GET", "/votes", nil)
		if err != nil {
			t.Errorf("this is the error: %v\n", err)
		}
		req = mux.SetURLVars(req, map[string]string{"id": v.id})

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(server.GetVote)
		handler.ServeHTTP(rr, req)

		responseMap := make(map[string]interface{})
		err = json.Unmarshal([]byte(rr.Body.String()), &responseMap)
		if err != nil {
			log.Fatalf("Cannot convert to json: %v", err)
		}
		assert.Equal(t, rr.Code, v.statusCode)

		if v.statusCode == 200 {
			assert.Equal(t, vote.Title, responseMap["title"])
			assert.Equal(t, vote.Desc, responseMap["description"])
			assert.Equal(t, float64(vote.AuthorID), responseMap["author_id"]) //the response author id is float64
		}
	}
}

func TestUpdateVote(t *testing.T) {

	var VoteUserEmail, VoteUserPassword string
	var AuthVoteAuthorID uint32
	var AuthVoteID uint64

	err := refreshUserAndVoteTable()
	if err != nil {
		log.Fatal(err)
	}
	users, votes, err := seedUsersAndVotes()
	if err != nil {
		log.Fatal(err)
	}
	// Get only the first user
	for _, user := range users {
		if user.ID == 2 {
			continue
		}
		VoteUserEmail = user.Email
		VoteUserPassword = "password" //Note the password in the database is already hashed, we want unhashed
	}
	//Login the user and get the authentication token
	token, err := server.SignIn(VoteUserEmail, VoteUserPassword)
	if err != nil {
		log.Fatalf("cannot login: %v\n", err)
	}
	tokenString := fmt.Sprintf("Bearer %v", token)

	// Get only the first vote
	for _, vote := range votes {
		if vote.ID == 2 {
			continue
		}
		AuthVoteID = vote.ID
		AuthVoteAuthorID = vote.AuthorID
	}
	// fmt.Printf("this is the auth vote: %v\n", AuthVoteID)

	samples := []struct {
		id           string
		updateJSON   string
		statusCode   int
		title        string
		desc         string
		author_id    uint32
		tokenGiven   string
		errorMessage string
	}{
		{
			// Convert int64 to int first before converting to string
			id:           strconv.Itoa(int(AuthVoteID)),
			updateJSON:   `{"title":"The updated vote", "desc": "This is the updated desc", "author_id": 1}`,
			statusCode:   200,
			title:        "The updated vote",
			desc:         "This is the updated desc",
			author_id:    AuthVoteAuthorID,
			tokenGiven:   tokenString,
			errorMessage: "",
		},
		{
			// When no token is provided
			id:           strconv.Itoa(int(AuthVoteID)),
			updateJSON:   `{"title":"This is still another title", "desc": "This is the updated desc", "author_id": 1}`,
			tokenGiven:   "",
			statusCode:   401,
			errorMessage: "Unauthorized",
		},
		{
			// When incorrect token is provided
			id:           strconv.Itoa(int(AuthVoteID)),
			updateJSON:   `{"title":"This is still another title", "desc": "This is the updated desc", "author_id": 1}`,
			tokenGiven:   "this is an incorrect token",
			statusCode:   401,
			errorMessage: "Unauthorized",
		},
		{
			//Note: "Title 2" belongs to vote 2, and title must be unique
			id:           strconv.Itoa(int(AuthVoteID)),
			updateJSON:   `{"title":"Title 2", "desc": "This is the updated desc", "author_id": 1}`,
			statusCode:   500,
			tokenGiven:   tokenString,
			errorMessage: "Title Already Taken",
		},
		{
			id:           strconv.Itoa(int(AuthVoteID)),
			updateJSON:   `{"title":"", "desc": "This is the updated desc", "author_id": 1}`,
			statusCode:   422,
			tokenGiven:   tokenString,
			errorMessage: "Required Title",
		},
		{
			id:           strconv.Itoa(int(AuthVoteID)),
			updateJSON:   `{"title":"Awesome title", "desc": "", "author_id": 1}`,
			statusCode:   422,
			tokenGiven:   tokenString,
			errorMessage: "Required desc",
		},
		{
			id:           strconv.Itoa(int(AuthVoteID)),
			updateJSON:   `{"title":"This is another title", "desc": "This is the updated desc"}`,
			statusCode:   401,
			tokenGiven:   tokenString,
			errorMessage: "Unauthorized",
		},
		{
			id:         "unknwon",
			statusCode: 400,
		},
		{
			id:           strconv.Itoa(int(AuthVoteID)),
			updateJSON:   `{"title":"This is still another title", "desc": "This is the updated desc", "author_id": 2}`,
			tokenGiven:   tokenString,
			statusCode:   401,
			errorMessage: "Unauthorized",
		},
	}

	for _, v := range samples {

		req, err := http.NewRequest("VOTE", "/votes", bytes.NewBufferString(v.updateJSON))
		if err != nil {
			t.Errorf("this is the error: %v\n", err)
		}
		req = mux.SetURLVars(req, map[string]string{"id": v.id})
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(server.UpdateVote)

		req.Header.Set("Authorization", v.tokenGiven)

		handler.ServeHTTP(rr, req)

		responseMap := make(map[string]interface{})
		err = json.Unmarshal([]byte(rr.Body.String()), &responseMap)
		if err != nil {
			t.Errorf("Cannot convert to json: %v", err)
		}
		assert.Equal(t, rr.Code, v.statusCode)
		if v.statusCode == 200 {
			assert.Equal(t, responseMap["title"], v.title)
			assert.Equal(t, responseMap["description"], v.desc)
			assert.Equal(t, responseMap["author_id"], float64(v.author_id)) //just to match the type of the json we receive thats why we used float64
		}
		if v.statusCode == 401 || v.statusCode == 422 || v.statusCode == 500 && v.errorMessage != "" {
			assert.Equal(t, responseMap["error"], v.errorMessage)
		}
	}
}

func TestDeleteVote(t *testing.T) {

	var VoteUserEmail, VoteUserPassword string
	var VoteUserID uint32
	var AuthVoteID uint64

	err := refreshUserAndVoteTable()
	if err != nil {
		log.Fatal(err)
	}
	users, votes, err := seedUsersAndVotes()
	if err != nil {
		log.Fatal(err)
	}
	//Let's get only the Second user
	for _, user := range users {
		if user.ID == 1 {
			continue
		}
		VoteUserEmail = user.Email
		VoteUserPassword = "password" //Note the password in the database is already hashed, we want unhashed
	}
	//Login the user and get the authentication token
	token, err := server.SignIn(VoteUserEmail, VoteUserPassword)
	if err != nil {
		log.Fatalf("cannot login: %v\n", err)
	}
	tokenString := fmt.Sprintf("Bearer %v", token)

	// Get only the second votes
	for _, vote := range votes {
		if vote.ID == 1 {
			continue
		}
		AuthVoteID = vote.ID
		VoteUserID = vote.AuthorID
	}
	voteSample := []struct {
		id           string
		author_id    uint32
		tokenGiven   string
		statusCode   int
		errorMessage string
	}{
		{
			// Convert int64 to int first before converting to string
			id:           strconv.Itoa(int(AuthVoteID)),
			author_id:    VoteUserID,
			tokenGiven:   tokenString,
			statusCode:   204,
			errorMessage: "",
		},
		{
			// When empty token is passed
			id:           strconv.Itoa(int(AuthVoteID)),
			author_id:    VoteUserID,
			tokenGiven:   "",
			statusCode:   401,
			errorMessage: "Unauthorized",
		},
		{
			// When incorrect token is passed
			id:           strconv.Itoa(int(AuthVoteID)),
			author_id:    VoteUserID,
			tokenGiven:   "This is an incorrect token",
			statusCode:   401,
			errorMessage: "Unauthorized",
		},
		{
			id:         "unknwon",
			tokenGiven: tokenString,
			statusCode: 400,
		},
		{
			id:           strconv.Itoa(int(1)),
			author_id:    1,
			statusCode:   401,
			errorMessage: "Unauthorized",
		},
	}
	for _, v := range voteSample {

		req, _ := http.NewRequest("GET", "/votes", nil)
		req = mux.SetURLVars(req, map[string]string{"id": v.id})

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(server.DeleteVote)

		req.Header.Set("Authorization", v.tokenGiven)

		handler.ServeHTTP(rr, req)

		assert.Equal(t, rr.Code, v.statusCode)

		if v.statusCode == 401 && v.errorMessage != "" {

			responseMap := make(map[string]interface{})
			err = json.Unmarshal([]byte(rr.Body.String()), &responseMap)
			if err != nil {
				t.Errorf("Cannot convert to json: %v", err)
			}
			assert.Equal(t, responseMap["error"], v.errorMessage)
		}
	}
}
