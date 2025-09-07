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

	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)


var (
	key = []byte(os.Getenv("SESSION_SECRET"))
	store = sessions.NewCookieStore(key)
)

var oauth = &models.Oauth {
	Config: &oauth2.Config{
		ClientID: os.Getenv("CLIENT_ID"),
		ClientSecret: os.Getenv("CLIENT_SECRET"),
		RedirectURL: "http://localhost:5173/oauth/creds",
		Endpoint: google.Endpoint,
		Scopes: []string{"email", "profile"},
	},
}


func saveToken(r *http.Request, w http.ResponseWriter, oa *models.Oauth, token *oauth2.Token, ctx context.Context) (*http.Client, error) {

	session, _ := store.Get(r, "oauth")

	var src = oauth2.ReuseTokenSource(token, oa.Config.TokenSource(ctx, token))

	t, err := src.Token()
	
	if err != nil {
		return nil, err
	}

	var id = uuid.NewString()

	var tokenData = &models.SessionData{
		ID: id,
		Token: t,
	}

	models.DB.Create(tokenData)

	session.Values["token_id"] = id
	session.Save(r, w)

	return oauth2.NewClient(ctx, src), nil
}


func newClient(w http.ResponseWriter, r *http.Request, code string, oa *models.Oauth) (*http.Client, error) {
	var ctx = r.Context()

	session, _ := store.Get(r, "oauth")
	token_id, ok := session.Values["token_id"]

	if ok {
		var sessionData = &models.SessionData{}

		if err := models.DB.First(sessionData, token_id).Error; err == nil {
			if sessionData.Token.Valid() {
				return saveToken(r, w, oa, sessionData.Token, ctx)
			}
		}
	}

	token, err := oa.Config.Exchange(ctx, code)

	if err != nil {
		return nil, err
	}
	
	return saveToken(r, w, oa, token, ctx)
}


func OauthSignUpH(w http.ResponseWriter, r *http.Request) {
	var state = utils.GenerateOauthState()

	session, _ := store.Get(r, "oauth")

	session.Values["state"] = state
	session.Save(r, w)

	var url = oauth.Config.AuthCodeURL(state, oauth2.AccessTypeOffline)

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}


func OauthCredsH(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	var (
		state	  =		r.URL.Query().Get("state")
		code 	  = 	r.URL.Query().Get("code")
	)

	session, _ := store.Get(r, "oauth")

	if session.IsNew || session.Values["state"] != state {
		http.Error(w, "invalid oauth state", http.StatusBadRequest)
		return
	}

	delete(session.Values, "state")
	session.Save(r, w)

	client, err := newClient(w, r, code, oauth)

	if err != nil {
		http.Error(w, "error trying to fetch datas: " + err.Error(), http.StatusInternalServerError)
		return
	}

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


	var id = uuid.NewString()

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
		Settings: models.Setting{
			AcceptsFriendReqs: true,
			AcceptsDMReqs: true,
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


	if err := models.DB.Create(&newUser); err != nil {
		http.Error(w, "user already exists!", http.StatusBadRequest)
		return
	}


	jwtToken, err := utils.NewJWT(id)

	if err != nil {
		http.Error(w, "error creating the token", http.StatusInternalServerError)
		return
	}

	var cookie = &http.Cookie{
		Name: "token",
		Value: jwtToken,
		Path: "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}

	http.SetCookie(w, cookie)

	
	oauth_creds().Render(r.Context(), w)
}