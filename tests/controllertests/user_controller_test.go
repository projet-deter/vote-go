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

func TestCreateUser(t *testing.T) {

	err := refreshUserTable()
	if err != nil {
		log.Fatal(err)
	}
	samples := []struct {
		inputJSON    string
		statusCode   int
		lastname     string
		firstname    string
		email        string
		errorMessage string
	}{
		{
			inputJSON:    `{"lastname":"Pet", "fistname":"Karl", "email": "pet@gmail.com", "password": "password"}`,
			statusCode:   201,
			lastname:     "Pet",
			firstname:    "Karl",
			email:        "pet@gmail.com",
			errorMessage: "",
		},
		{
			inputJSON:    `{"lastname":"Frank", "fistname":"Karl", "email": "pet@gmail.com", "password": "password"}`,
			statusCode:   500,
			errorMessage: "Email Already Taken",
		},
		{
			inputJSON:    `{"lastname":"Pet", "fistname":"Karl","email": "grand@gmail.com", "password": "password"}`,
			statusCode:   500,
			errorMessage: "lastname Already Taken",
		},
		{
			inputJSON:    `{"lastname":"Kan","fistname":"Karl", "email": "kangmail.com", "password": "password"}`,
			statusCode:   422,
			errorMessage: "Invalid Email",
		},
		{
			inputJSON:    `{"lastname": "", "fistname":"Karl", "email": "kan@gmail.com", "password": "password"}`,
			statusCode:   422,
			errorMessage: "Required lastname",
		},
		{
			inputJSON:    `{"lastname": "Kan","fistname":"Karl", "email": "", "password": "password"}`,
			statusCode:   422,
			errorMessage: "Required Email",
		},
		{
			inputJSON:    `{"lastname": "Kan","fistname":"Karl", "email": "kan@gmail.com", "password": ""}`,
			statusCode:   422,
			errorMessage: "Required Password",
		},
	}

	for _, v := range samples {

		req, err := http.NewRequest("Vote", "/users", bytes.NewBufferString(v.inputJSON))
		if err != nil {
			t.Errorf("this is the error: %v", err)
		}
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(server.CreateUser)
		handler.ServeHTTP(rr, req)

		responseMap := make(map[string]interface{})
		err = json.Unmarshal([]byte(rr.Body.String()), &responseMap)
		if err != nil {
			fmt.Printf("Cannot convert to json: %v", err)
		}
		assert.Equal(t, rr.Code, v.statusCode)
		if v.statusCode == 201 {
			assert.Equal(t, responseMap["lastname"], v.lastname)
			assert.Equal(t, responseMap["firstname"], v.firstname)
			assert.Equal(t, responseMap["email"], v.email)
		}
		if v.statusCode == 422 || v.statusCode == 500 && v.errorMessage != "" {
			assert.Equal(t, responseMap["error"], v.errorMessage)
		}
	}
}

func TestGetUsers(t *testing.T) {

	err := refreshUserTable()
	if err != nil {
		log.Fatal(err)
	}
	_, err = seedUsers()
	if err != nil {
		log.Fatal(err)
	}
	req, err := http.NewRequest("GET", "/users", nil)
	if err != nil {
		t.Errorf("this is the error: %v\n", err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.GetUsers)
	handler.ServeHTTP(rr, req)

	var users []models.User
	err = json.Unmarshal([]byte(rr.Body.String()), &users)
	if err != nil {
		log.Fatalf("Cannot convert to json: %v\n", err)
	}
	assert.Equal(t, rr.Code, http.StatusOK)
	assert.Equal(t, len(users), 2)
}

func TestGetUserByID(t *testing.T) {

	err := refreshUserTable()
	if err != nil {
		log.Fatal(err)
	}
	user, err := seedOneUser()
	if err != nil {
		log.Fatal(err)
	}
	userSample := []struct {
		id           string
		statusCode   int
		lastname     string
		firstname    string
		email        string
		errorMessage string
	}{
		{
			id:         strconv.Itoa(int(user.ID)),
			statusCode: 200,
			lastname:   user.Last_name,
			firstname:  user.First_name,
			email:      user.Email,
		},
		{
			id:         "unknwon",
			statusCode: 400,
		},
	}
	for _, v := range userSample {

		req, err := http.NewRequest("GET", "/users", nil)
		if err != nil {
			t.Errorf("This is the error: %v\n", err)
		}
		req = mux.SetURLVars(req, map[string]string{"id": v.id})
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(server.GetUser)
		handler.ServeHTTP(rr, req)

		responseMap := make(map[string]interface{})
		err = json.Unmarshal([]byte(rr.Body.String()), &responseMap)
		if err != nil {
			log.Fatalf("Cannot convert to json: %v", err)
		}

		assert.Equal(t, rr.Code, v.statusCode)

		if v.statusCode == 200 {
			assert.Equal(t, user.Last_name, responseMap["LastName"])
			assert.Equal(t, user.First_name, responseMap["Firstname"])
			assert.Equal(t, user.Email, responseMap["email"])
		}
	}
}

func TestUpdateUser(t *testing.T) {

	var AuthEmail, AuthPassword string
	var AuthID uint32

	err := refreshUserTable()
	if err != nil {
		log.Fatal(err)
	}
	users, err := seedUsers() //we need atleast two users to properly check the update
	if err != nil {
		log.Fatalf("Error seeding user: %v\n", err)
	}
	// Get only the first user
	for _, user := range users {
		if user.ID == 2 {
			continue
		}
		AuthID = user.ID
		AuthEmail = user.Email
		AuthPassword = "password" //Note the password in the database is already hashed, we want unhashed
	}
	//Login the user and get the authentication token
	token, err := server.SignIn(AuthEmail, AuthPassword)
	if err != nil {
		log.Fatalf("cannot login: %v\n", err)
	}
	tokenString := fmt.Sprintf("Bearer %v", token)

	samples := []struct {
		id              string
		updateJSON      string
		statusCode      int
		updatelastname  string
		updatefirstname string
		updateEmail     string
		tokenGiven      string
		errorMessage    string
	}{
		{
			// Convert int32 to int first before converting to string
			id:              strconv.Itoa(int(AuthID)),
			updateJSON:      `{"Lastname":"Grand", "Firstname":"April","email": "grand@gmail.com", "password": "password"}`,
			statusCode:      200,
			updatelastname:  "Grand",
			updatefirstname: "April",
			updateEmail:     "grand@gmail.com",
			tokenGiven:      tokenString,
			errorMessage:    "",
		},
		{
			// When password field is empty
			id:           strconv.Itoa(int(AuthID)),
			updateJSON:   `{"Lastname":"woman", "Firstname":"Sandra","email": "woman@gmail.com", "password": ""}`,
			statusCode:   422,
			tokenGiven:   tokenString,
			errorMessage: "Required Password",
		},
		{
			// When no token was passed
			id:           strconv.Itoa(int(AuthID)),
			updateJSON:   `{"Lastname":"Man", "Firstname":"Pedro", "email": "man@gmail.com", "password": "password"}`,
			statusCode:   401,
			tokenGiven:   "",
			errorMessage: "Unauthorized",
		},
		{
			// When incorrect token was passed
			id:           strconv.Itoa(int(AuthID)),
			updateJSON:   `{"Lastname":"woman", "Firstname":"Sandra","email": "woman@gmail.com", "password": "password"}`,
			statusCode:   401,
			tokenGiven:   "This is incorrect token",
			errorMessage: "Unauthorized",
		},
		{
			// Remember "kenny@gmail.com" belongs to user 2
			id:           strconv.Itoa(int(AuthID)),
			updateJSON:   `{"Lastname":"Frank", "Firstname":"Kenny","email": "kenny@gmail.com", "password": "password"}`,
			statusCode:   500,
			tokenGiven:   tokenString,
			errorMessage: "Email Already Taken",
		},
		{
			// Remember "Kenny Morris" belongs to user 2
			id:           strconv.Itoa(int(AuthID)),
			updateJSON:   `{"Lastname":"Morris", "Firstname":"Kenny", "email": "grand@gmail.com", "password": "password"}`,
			statusCode:   500,
			tokenGiven:   tokenString,
			errorMessage: "Lastname Already Taken",
		},
		{
			id:           strconv.Itoa(int(AuthID)),
			updateJSON:   `{"Lastname":"Kan", "Firstname":"Liyo", "email": "kangmail.com", "password": "password"}`,
			statusCode:   422,
			tokenGiven:   tokenString,
			errorMessage: "Invalid Email",
		},
		{
			id:           strconv.Itoa(int(AuthID)),
			updateJSON:   `{"Lastname":"", "Firstname":"","email": "kan@gmail.com", "password": "password"}`,
			statusCode:   422,
			tokenGiven:   tokenString,
			errorMessage: "Required Lastname",
		},
		{
			id:           strconv.Itoa(int(AuthID)),
			updateJSON:   `{"Lastname":"Kan", "Firstname":"Liyo", "email": "", "password": "password"}`,
			statusCode:   422,
			tokenGiven:   tokenString,
			errorMessage: "Required Email",
		},
		{
			id:         "unknwon",
			tokenGiven: tokenString,
			statusCode: 400,
		},
		{
			// When user 2 is using user 1 token
			id:           strconv.Itoa(int(2)),
			updateJSON:   `{"Lastname":"Moke", "Firstname":"Kayin", "email": "mike@gmail.com", "password": "password"}`,
			tokenGiven:   tokenString,
			statusCode:   401,
			errorMessage: "Unauthorized",
		},
	}

	for _, v := range samples {

		req, err := http.NewRequest("Vote", "/users", bytes.NewBufferString(v.updateJSON))
		if err != nil {
			t.Errorf("This is the error: %v\n", err)
		}
		req = mux.SetURLVars(req, map[string]string{"id": v.id})

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(server.UpdateUser)

		req.Header.Set("Authorization", v.tokenGiven)

		handler.ServeHTTP(rr, req)

		responseMap := make(map[string]interface{})
		err = json.Unmarshal([]byte(rr.Body.String()), &responseMap)
		if err != nil {
			t.Errorf("Cannot convert to json: %v", err)
		}
		assert.Equal(t, rr.Code, v.statusCode)
		if v.statusCode == 200 {
			assert.Equal(t, responseMap["Lastname"], v.updatelastname)
			assert.Equal(t, responseMap["Fiestname"], v.updatefirstname)
			assert.Equal(t, responseMap["email"], v.updateEmail)
		}
		if v.statusCode == 401 || v.statusCode == 422 || v.statusCode == 500 && v.errorMessage != "" {
			assert.Equal(t, responseMap["error"], v.errorMessage)
		}
	}
}

func TestDeleteUser(t *testing.T) {

	var AuthEmail, AuthPassword string
	var AuthID uint32

	err := refreshUserTable()
	if err != nil {
		log.Fatal(err)
	}

	users, err := seedUsers() //we need atleast two users to properly check the update
	if err != nil {
		log.Fatalf("Error seeding user: %v\n", err)
	}
	// Get only the first and log him in
	for _, user := range users {
		if user.ID == 2 {
			continue
		}
		AuthID = user.ID
		AuthEmail = user.Email
		AuthPassword = "password" ////Note the password in the database is already hashed, we want unhashed
	}
	//Login the user and get the authentication token
	token, err := server.SignIn(AuthEmail, AuthPassword)
	if err != nil {
		log.Fatalf("cannot login: %v\n", err)
	}
	tokenString := fmt.Sprintf("Bearer %v", token)

	userSample := []struct {
		id           string
		tokenGiven   string
		statusCode   int
		errorMessage string
	}{
		{
			// Convert int32 to int first before converting to string
			id:           strconv.Itoa(int(AuthID)),
			tokenGiven:   tokenString,
			statusCode:   204,
			errorMessage: "",
		},
		{
			// When no token is given
			id:           strconv.Itoa(int(AuthID)),
			tokenGiven:   "",
			statusCode:   401,
			errorMessage: "Unauthorized",
		},
		{
			// When incorrect token is given
			id:           strconv.Itoa(int(AuthID)),
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
			// User 2 trying to use User 1 token
			id:           strconv.Itoa(int(2)),
			tokenGiven:   tokenString,
			statusCode:   401,
			errorMessage: "Unauthorized",
		},
	}
	for _, v := range userSample {

		req, err := http.NewRequest("GET", "/users", nil)
		if err != nil {
			t.Errorf("This is the error: %v\n", err)
		}
		req = mux.SetURLVars(req, map[string]string{"id": v.id})
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(server.DeleteUser)

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
