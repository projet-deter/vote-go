package modeltests

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/joho/godotenv"
	"github.com/m2fof/vote/api/controllers"
	"github.com/m2fof/vote/api/models"
)

var server = controllers.Server{}
var userInstance = models.User{}
var voteInstance = models.Vote{}

func TestMain(m *testing.M) {
	var err error
	err = godotenv.Load(os.ExpandEnv("../../.env"))
	if err != nil {
		log.Fatalf("Error getting env %v\n", err)
	}
	Database()

	os.Exit(m.Run())
}

func Database() {

	var err error

	TestDbDriver := os.Getenv("TestDbDriver")

	if TestDbDriver == "postgres" {
		DBURL := fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=disable password=%s", os.Getenv("TestDbHost"), os.Getenv("TestDbPort"), os.Getenv("TestDbUser"), os.Getenv("TestDbName"), os.Getenv("TestDbPassword"))
		server.DB, err = gorm.Open(TestDbDriver, DBURL)
		if err != nil {
			fmt.Printf("Cannot connect to %s database\n", TestDbDriver)
			log.Fatal("This is the error:", err)
		} else {
			fmt.Printf("We are connected to the %s database\n", TestDbDriver)
		}
	}
}

func refreshUserTable() error {
	err := server.DB.DropTableIfExists(&models.User{}).Error
	if err != nil {
		return err
	}
	err = server.DB.AutoMigrate(&models.User{}).Error
	if err != nil {
		return err
	}
	log.Printf("Successfully refreshed table")
	return nil
}

func seedOneUser() (models.User, error) {

	refreshUserTable()

	user := models.User{
		First_name: "Pet ",
		Last_name:  " victor",
		Email:      "pet@gmail.com",
		Password:   "password",
	}

	err := server.DB.Model(&models.User{}).Create(&user).Error
	if err != nil {
		log.Fatalf("cannot seed users table: %v", err)
	}
	return user, nil
}

func seedUsers() error {

	users := []models.User{
		models.User{
			First_name: "Steven ",
			Last_name:  " victor",
			Email:      "steven@gmail.com",
			Password:   "password",
			Birth_date: "11/05/2000",
		},
		models.User{
			First_name: "Martin ",
			Last_name:  "Luther",
			Email:      "luther@gmail.com",
			Password:   "password",
			Birth_date: "24/04/2005",
		},
	}

	for i, _ := range users {
		err := server.DB.Model(&models.User{}).Create(&users[i]).Error
		if err != nil {
			return err
		}
	}
	return nil
}

func refreshUserAndVoteTable() error {

	err := server.DB.DropTableIfExists(&models.User{}, &models.Vote{}).Error
	if err != nil {
		return err
	}
	err = server.DB.AutoMigrate(&models.User{}, &models.Vote{}).Error
	if err != nil {
		return err
	}
	log.Printf("Successfully refreshed tables")
	return nil
}

func seedOneUserAndOneVote() (models.Vote, error) {

	err := refreshUserAndVoteTable()
	if err != nil {
		return models.Vote{}, err
	}
	user := models.User{
		First_name: "Sam ",
		Last_name:  " Phil",
		Email:      "sam@gmail.com",
		Password:   "password",
	}
	err = server.DB.Model(&models.User{}).Create(&user).Error
	if err != nil {
		return models.Vote{}, err
	}
	vote := models.Vote{
		Title:    "This is the title sam",
		Desc:     "This is the content sam",
		AuthorID: user.ID,
	}
	err = server.DB.Model(&models.Vote{}).Create(&vote).Error
	if err != nil {
		return models.Vote{}, err
	}
	return vote, nil
}

func seedUsersAndVotes() ([]models.User, []models.Vote, error) {

	var err error

	if err != nil {
		return []models.User{}, []models.Vote{}, err
	}
	var users = []models.User{
		models.User{
			First_name: "Steven ",
			Last_name:  " victor",
			Email:      "steven@gmail.com",
			Password:   "password",
			Birth_date: "11/05/2000",
		},
		models.User{
			First_name: "Martin ",
			Last_name:  "Luther",
			Email:      "luther@gmail.com",
			Password:   "password",
			Birth_date: "24/04/2005",
		},
	}
	var votes = []models.Vote{
		models.Vote{
			Title: "Title 1",
			Desc:  "Vote 1",
		},
		models.Vote{
			Title: "Title 2",
			Desc:  "Vote 2",
		},
	}

	for i, _ := range users {
		err = server.DB.Model(&models.User{}).Create(&users[i]).Error
		if err != nil {
			log.Fatalf("cannot seed users table: %v", err)
		}
		votes[i].AuthorID = users[i].ID

		err = server.DB.Model(&models.Vote{}).Create(&votes[i]).Error
		if err != nil {
			log.Fatalf("cannot seed votes table: %v", err)
		}
	}
	return users, votes, nil
}
