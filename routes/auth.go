package routes

import (
	"database/sql"
	"fmt"
	"gochat/db"
	"gochat/pages"
	"gochat/types"
	"gochat/utils"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)


var providers = []string{"google", "discord"}


func SignUpH(w http.ResponseWriter, r *http.Request) {	
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "couldnt get the datas from the form", http.StatusInternalServerError)
			return
		}

		var (
			username = r.PostFormValue("username")
			password = r.PostFormValue("password")
			email 	 = r.PostFormValue("email")
		)

		if !utils.Validate("username", username) {
			http.Error(w, "the username is invalid, it should have at least 4 letters and a maximum of 16, without any special characters", http.StatusBadRequest)
			return
		} else if !utils.Validate("email", email) {
			http.Error(w, "invalid email", http.StatusBadRequest)
			return
		} else if !utils.Validate("password", password) {
			http.Error(w, "the password is invalid, it should have at least 4 lowercase letter, 4 uppercase, 4 digits and 4 special characters", http.StatusBadRequest)
			return
		}

		bytePassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

		if err != nil {
			http.Error(w, "error saving the datas", http.StatusInternalServerError)
			return
		}
	
		var id = uuid.New()
		
		if err := db.CreateUser(r.Context(), &db.CreateUserParams{
			ID: id,
			Username: username,
			Email: email,
			Password: sql.NullString{String: string(bytePassword), Valid: true},
		}); err != nil {
			http.Error(w, "error saving the user datas", http.StatusInternalServerError)
			return
		}

		jwtToken, err := utils.NewJWT(id)

		if err != nil {
			http.Error(w, "error saving the user datas", http.StatusInternalServerError)
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

		http.Redirect(w, r, "/", http.StatusFound)
	}
	

	if _, _, err := utils.GetUserID(r); err != nil {
		pages.Login(false, "signup", providers).Render(r.Context(), w)
		return
	}

	pages.Login(true, "", nil).Render(r.Context(), w)
}


func LoginH(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "couldnt get the datas from the form", http.StatusInternalServerError)
			return
		}

		var (
			username = r.PostFormValue("username")
			password = r.PostFormValue("password")
		)

		user, _ := db.Query.GetUserByName(r.Context(), username)

		if err := bcrypt.CompareHashAndPassword([]byte(user.Password.String), []byte(password)); err != nil {

			http.Error(w, "wrong password", http.StatusBadRequest)
			return

		} else {

			jwtToken, err := utils.NewJWT(user.ID)

			if err != nil {
				http.Error(w, "couldnt save the user datas", http.StatusInternalServerError)
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

			http.Redirect(w, r, "/", http.StatusFound)
		}
	}

	if _, _, err := utils.GetUserID(r); err != nil {
		pages.Login(false, "login", providers).Render(r.Context(), w)
		return
	}

	pages.Login(true, "", nil).Render(r.Context(), w)
}


func LogOffH(w http.ResponseWriter, r *http.Request) {
	var cookie = &http.Cookie{
		Name: "token",
		Value: "",
		Path: "/",
		Expires: time.Unix(0, 0),
		MaxAge: -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}

	http.SetCookie(w, cookie)
}


func EmailVerificationSendH(w http.ResponseWriter, r *http.Request) {
	id, status, err := utils.GetUserID(r)

	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}

	user, _ := db.Query.GetUserById(r.Context(), id)
	
	if r.Method == http.MethodPost {
		if user.Verified {
			http.Error(w, "you are already verified!", http.StatusBadRequest)
			return
		}

		var token = utils.GenerateToken()

		db.Query.DeleteAuthVerification(r.Context(), db.DeleteAuthVerificationParams{
			UserID: user.ID,
			Type: types.EMAIL_VERIFICATION,
		})

		db.Query.CreateAuthVerification(r.Context(), db.CreateAuthVerificationParams{
			UserID: user.ID,
			Type: types.EMAIL_VERIFICATION,
			Token: token,
			Expiry: time.Now().Add(time.Hour),
		})

		if err := utils.SendVerificationEmail(types.EMAIL_VERIFICATION, token, user.Email, r.Host); err != nil {
			log.Println(utils.GetFuncInfo(), err)
			http.Error(w, "couldnt send the email", http.StatusInternalServerError)
			return
		} else {
			fmt.Fprint(w, "email sent")
			return
		}
	}
	

	pages.EmailVerificationSend(user).Render(r.Context(), w)
}


func EmailVerificationH(w http.ResponseWriter, r *http.Request) {
	id, status, err := utils.GetUserID(r)

	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}

	user, _ := db.Query.GetUserById(r.Context(), id)

	if user.Verified {
		fmt.Fprint(w, "you are already verified")
		return
	}

	emailVerif, err := db.Query.GetAuthVerification(r.Context(), db.GetAuthVerificationParams{
		UserID: user.ID,
		Type: types.EMAIL_VERIFICATION,
	})

	if err != nil {
		http.Error(w, "couldnt get the email verification datas", http.StatusInternalServerError)
		return
	}

	if time.Now().After(emailVerif.Expiry) {
		http.Error(w, "your token expired!", http.StatusBadRequest)
		return
	}

	var token = r.URL.Query().Get("token")

	if token != emailVerif.Token {

		http.Error(w, "this isnt the right token!", http.StatusBadRequest)
		return

	} else {
		db.Query.UpdateUser(r.Context(), db.UpdateUserParams{ID: user.ID, Verified: true})

		db.Query.CreateNotification(r.Context(), db.CreateNotificationParams{
			UserID: user.ID,
			NotifFrom: "app",
			Message: "Your account has been verified",
			Link: "/",
			Type: string(types.ACC_VERIFIED),
		})

		db.Query.DeleteAuthVerification(r.Context(), db.DeleteAuthVerificationParams{
			UserID: user.ID,
			Type: types.EMAIL_VERIFICATION,
		})

		db.Query.DeleteAllNotificationsByType(r.Context(), db.DeleteAllNotificationsByTypeParams{
			UserID: user.ID,
			Type: string(types.ACC_VERIFY_NEED),
		})
	}
}


func PasswordResetH(w http.ResponseWriter, r *http.Request) {
	id, status, err := utils.GetUserID(r)

	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}

	user, _ := db.Query.GetUserById(r.Context(), id)

	if r.Method == http.MethodPost {
		var token = utils.GenerateToken()

		db.Query.CreateAuthVerification(r.Context(), db.CreateAuthVerificationParams{
			UserID: user.ID,
			Type: types.PASSWORD_RESET,
			Token: token,
			Expiry: time.Now().Add(time.Hour),
		})
	
		if err := utils.SendVerificationEmail(types.PASSWORD_RESET, token, user.Email, r.Host); err != nil {
			http.Error(w, "couldnt send the verification email", http.StatusInternalServerError)
			return
		} else {
			fmt.Fprint(w, "verification email sent")
			return
		}
	}

	
	pages.PasswordResetSend().Render(r.Context(), w)
}


func PasswordNewH(w http.ResponseWriter, r *http.Request) {
	id, status, err := utils.GetUserID(r)

	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}

	user, _ := db.Query.GetUserById(r.Context(), id)

	if r.Method == http.MethodPost {

		r.ParseForm()

		passwordVerif, err := db.Query.GetAuthVerification(r.Context(), db.GetAuthVerificationParams{
			UserID: id,
			Type: types.PASSWORD_NEW,
		})

		if err != nil {
			http.Error(w, "you didnt send a request to reset your password", http.StatusBadRequest)
			return
		}

		if time.Now().After(passwordVerif.Expiry) {
			http.Error(w, "the token expired!", http.StatusBadRequest)
			return
		}

		var token = r.URL.Query().Get("token")

		if token != passwordVerif.Token {
			http.Error(w, "this isnt the right token!", http.StatusBadRequest)
			return
		}

		var password = r.PostFormValue("password")
	
		if !utils.Validate("password", password) {
			http.Error(w, "the password is invalid", http.StatusBadRequest)
			return
		}
	
		bytePassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	
		if err != nil {
			http.Error(w, "couldnt save the user datas", http.StatusInternalServerError)
			return
		}

		db.Query.UpdateUser(r.Context(), db.UpdateUserParams{ID: user.ID, Password: string(bytePassword)})

		db.Query.DeleteAuthVerification(r.Context(), db.DeleteAuthVerificationParams{
			UserID: user.ID,
			Type: types.PASSWORD_NEW,
		})
	}


	pages.PasswordForgot(user).Render(r.Context(), w)
}