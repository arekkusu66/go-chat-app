package routes

import (
	"context"
	"encoding/json"
	"gochat/models"
	"gochat/types"
	"gochat/utils"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)


func oauth() models.Oauth {
	return models.Oauth{
		Config: &oauth2.Config{
			ClientID: os.Getenv("CLIENT_ID"),
			ClientSecret: os.Getenv("CLIENT_SECRET"),
			RedirectURL: "http://localhost:5173/oauth/creds",
			Endpoint: google.Endpoint,
			Scopes: []string{"email", "profile"},
		},
	}
}


func OauthSignUpH(w http.ResponseWriter, r *http.Request) {
	var url = oauth().Config.AuthCodeURL("idk", oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}


func OauthCredsH(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	var code = r.URL.Query().Get("code")

	token, err := oauth().Config.Exchange(context.Background(), code)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if !token.Valid() {
		http.Error(w, "token is invalid, please try to sign up again", http.StatusBadRequest)
		return
	}

	var client = oauth().Config.Client(context.Background(), token)

	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer resp.Body.Close()

	var datas models.ProviderDatas

	if err := json.NewDecoder(resp.Body).Decode(&datas); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}


	var id = uuid.New().String()

	var newUser = models.User{
		ID: id,
		Username: "user_" + id,
		Email: datas.Email,
		Verified: datas.Verified,
		Joined: time.Now(),
		Notifications: []models.Notification{
			{
				Message: "You successfully created an account",
				Date: time.Now(),
				Link: "/",
				Type: types.ACC_CREATED,
				NotifFrom: "app",
			},
		},
	}


	if !datas.Verified {
		newUser.EmailVerification = &models.AuthVerification{
			UserID: id,
			Type: types.EMAIL_VERIFY,
			Token: utils.GenerateToken(),
			Expiry: time.Now().Add(time.Hour),
		}

		newUser.Notifications = []models.Notification{
			{
				Message: "You have succesfully created your account, but you need to verify your email",
				Date: time.Now(),
				Link: "/email/verification/send",
				Type: types.ACC_VERIFY_NEED,
				NotifFrom: "app",
			},
		}
	}


	models.DB.Create(&newUser)


	var claims = &models.Claims{
		ID: id,
	}

	jwtToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(os.Getenv("JWT_SECRET")))

	if err != nil {
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	var cookie = &http.Cookie{
		Name: "token",
		Value: jwtToken,
		Path: "/",
		SameSite: http.SameSiteLaxMode,
		HttpOnly: true,
		Secure: true,
	}

	http.SetCookie(w, cookie)

	
	oauth_creds().Render(r.Context(), w)
}