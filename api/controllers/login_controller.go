package controllers

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"log"
	"strconv"
	"net/http"

	"github.com/m2fof/vote/api/auth"
	"github.com/m2fof/vote/api/models"
	"github.com/m2fof/vote/api/responses"
	"github.com/m2fof/vote/api/utils/formaterror"
	"golang.org/x/crypto/bcrypt"
)

func (server *Server) Login(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		responses.ERROR(w, http.StatusUnprocessableEntity, err)
		return
	}
	user := models.User{}
	err = json.Unmarshal(body, &user)
	if err != nil {
		responses.ERROR(w, http.StatusUnprocessableEntity, err)
		return
	}

	user.Prepare()
	err = user.Validate("login")
	if err != nil {
		responses.ERROR(w, http.StatusUnprocessableEntity, err)
		return
	}
	token, err := server.SignIn(user.Email, user.Password)
	if err != nil {
		formattedError := formaterror.FormatError(err.Error())
		responses.ERROR(w, http.StatusUnprocessableEntity, formattedError)
		return
	}

	responses.JSON(w, http.StatusOK, token)
}

func (server *Server) SignIn(email, password string) (string, error) {

	var err error

	user := models.User{}

	err = server.DB.Debug().Model(models.User{}).Where("email = ?", email).Take(&user).Error
	if err != nil {
		return "", err
	}
	err = models.VerifyPassword(user.Password, password)
	if err != nil && err == bcrypt.ErrMismatchedHashAndPassword {
		return "", err
	}

	os.Setenv("currentUserFirst_name", user.First_name)
	os.Setenv("currentUserAccessLevel", strconv.FormatInt(int64(user.AccessLevel), 10))
	log.Println("New Login:")
	log.Println(os.Getenv("currentUserFirst_name"))
	log.Println(os.Getenv("currentUserAccessLevel"))


	return auth.CreateToken(user.ID)
}
