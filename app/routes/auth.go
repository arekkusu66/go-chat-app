package routes

import (
	"database/sql"
	"fmt"
	"gochat/app"
	"gochat/app/db"
	"gochat/app/pages"
	"gochat/app/types"
	"gochat/app/utils"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var providers = []string{"google", "discord"}


func SignUpH(app *app.App) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			
			if err := db.CreateUser(r.Context(), app.DB, app.Query, &db.CreateUserParams{
				ID: id,
				Username: username,
				Email: email,
				Password: sql.NullString{String: string(bytePassword), Valid: true},
			}); err != nil {
				http.Error(w, "error saving the user datas", http.StatusInternalServerError)
				return
			}

			jwtToken, err := app.NewJWT(id)

			if err != nil {
				http.Error(w, "error saving the user datas", http.StatusInternalServerError)
				return
			}

			var cookie = &http.Cookie{
				Name: "token",
				Value: jwtToken,
				Path: "/",
				HttpOnly: true,
			}

			http.SetCookie(w, cookie)

			http.Redirect(w, r, "/", http.StatusFound)
		}
		

		if _, _, err := app.GetUserID(r, w); err != nil {
			pages.Login(app, false, "signup", providers).Render(r.Context(), w)
			return
		}

		pages.Login(app, true, "", nil).Render(r.Context(), w)
	})
}


func LoginH(app *app.App) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if err := r.ParseForm(); err != nil {
				http.Error(w, "couldnt get the datas from the form", http.StatusInternalServerError)
				return
			}

			var (
				username = r.PostFormValue("username")
				password = r.PostFormValue("password")
			)

			user, _ := app.Query.GetUserByName(r.Context(), username)

			if err := bcrypt.CompareHashAndPassword([]byte(user.Password.String), []byte(password)); err != nil {

				http.Error(w, "wrong password", http.StatusBadRequest)
				return

			} else {

				jwtToken, err := app.NewJWT(user.ID)

				if err != nil {
					http.Error(w, "couldnt save the user datas", http.StatusInternalServerError)
					return
				}

				var cookie = &http.Cookie{
					Name: "token",
					Value: jwtToken,
					Path: "/",
					HttpOnly: true,

				}
			
				http.SetCookie(w, cookie)

				http.Redirect(w, r, "/", http.StatusFound)
			}
		}

		if _, _, err := app.GetUserID(r, w); err != nil {
			pages.Login(app, false, "login", providers).Render(r.Context(), w)
			return
		}

		pages.Login(app, true, "", nil).Render(r.Context(), w)
	})
}


func LogOffH(app *app.App) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var cookie = &http.Cookie{
			Name: "token",
			Value: "",
			Path: "/",
			Expires: time.Unix(0, 0),
			MaxAge: -1,
			HttpOnly: true,
		}

		http.SetCookie(w, cookie)
	})
}


func EmailVerificationSendH(app *app.App) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, status, err := app.GetUserID(r, w)

		if err != nil {
			http.Error(w, err.Error(), status)
			return
		}

		user, _ := app.Query.GetUserById(r.Context(), id)
		
		if r.Method == http.MethodPost {
			if user.Verified {
				http.Error(w, "you are already verified!", http.StatusBadRequest)
				return
			}

			var token = utils.GenerateToken()

			app.Query.DeleteAuthVerification(r.Context(), db.DeleteAuthVerificationParams{
				UserID: user.ID,
				Kind: types.EMAIL_VERIFICATION,
			})

			app.Query.CreateAuthVerification(r.Context(), db.CreateAuthVerificationParams{
				UserID: user.ID,
				Kind: types.EMAIL_VERIFICATION,
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
		

		pages.EmailVerificationSend(app, user).Render(r.Context(), w)
	})
}


func EmailVerificationH(app *app.App) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, status, err := app.GetUserID(r, w)

		if err != nil {
			http.Error(w, err.Error(), status)
			return
		}

		user, _ := app.Query.GetUserById(r.Context(), id)

		if user.Verified {
			fmt.Fprint(w, "you are already verified")
			return
		}

		emailVerif, err := app.Query.GetAuthVerification(r.Context(), db.GetAuthVerificationParams{
			UserID: user.ID,
			Kind: types.EMAIL_VERIFICATION,
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
			app.Query.UpdateUser(r.Context(), db.UpdateUserParams{ID: user.ID, Verified: true})

			app.Query.CreateNotification(r.Context(), db.CreateNotificationParams{
				UserID: user.ID,
				NotifFrom: "app",
				Message: "Your account has been verified",
				Link: "/",
				Kind: string(types.ACC_VERIFIED),
			})

			app.Query.DeleteAuthVerification(r.Context(), db.DeleteAuthVerificationParams{
				UserID: user.ID,
				Kind: types.EMAIL_VERIFICATION,
			})

			app.Query.DeleteAllNotificationsByKind(r.Context(), db.DeleteAllNotificationsByKindParams{
				UserID: user.ID,
				Kind: string(types.ACC_VERIFY_NEED),
			})
		}
	})
}


func PasswordResetH(app *app.App) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, status, err := app.GetUserID(r, w)

		if err != nil {
			http.Error(w, err.Error(), status)
			return
		}

		user, _ := app.Query.GetUserById(r.Context(), id)

		if r.Method == http.MethodPost {
			var token = utils.GenerateToken()

			app.Query.CreateAuthVerification(r.Context(), db.CreateAuthVerificationParams{
				UserID: user.ID,
				Kind: types.PASSWORD_RESET,
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
	})
}


func PasswordNewH(app *app.App) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, status, err := app.GetUserID(r, w)

		if err != nil {
			http.Error(w, err.Error(), status)
			return
		}

		user, _ := app.Query.GetUserById(r.Context(), id)

		if r.Method == http.MethodPost {
			r.ParseForm()

			passwordVerif, err := app.Query.GetAuthVerification(r.Context(), db.GetAuthVerificationParams{
				UserID: id,
				Kind: types.PASSWORD_NEW,
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

			app.Query.UpdateUser(r.Context(), db.UpdateUserParams{ID: user.ID, Password: string(bytePassword)})

			app.Query.DeleteAuthVerification(r.Context(), db.DeleteAuthVerificationParams{
				UserID: user.ID,
				Kind: types.PASSWORD_NEW,
			})
		}


		pages.PasswordForgot(app, user).Render(r.Context(), w)
	})
}