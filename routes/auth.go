package routes

import (
	"database/sql"
	"gochat/models"
	"gochat/types"
	"gochat/utils"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)


func SignUpH(w http.ResponseWriter, r *http.Request) {	
	var id = uuid.New().String()

	var claims = &models.Claims{
		ID: id,
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(os.Getenv("JWT_SECRET")))

	if err != nil {
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		return
	}


	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "couldnt get the datas from the form", http.StatusInternalServerError)
			return
		}

		var (
			username = r.PostFormValue("username")
			password = r.PostFormValue("password")
			email = r.PostFormValue("email")
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
			http.Error(w, "couldnt save password", http.StatusInternalServerError)
			return
		}


		var newUser = models.User{
			ID: id,
			Username: username,
			Password: sql.NullString{String: string(bytePassword), Valid: true},
			Email: email,
			Joined: time.Now(),
			EmailVerification: &models.AuthVerification{
				UserID: id,
				Type: types.EMAIL_VERIFY,
				Token: utils.GenerateToken(),
				Expiry: time.Now().Add(time.Hour),
			},
			Notifications: []models.Notification{
				{
					Message: "You have succesfully created your account, but you need to verify your email",
					Date: time.Now(),
					Link: "/email/verification/send",
					Type: types.ACC_VERIFY_NEED,
					NotifFrom: "app",
				},
			},
			Settings: models.Setting{
				AcceptsFriendReqs: true,
				AcceptsDMReqs: true,
			},
		}

		if err := models.DB.Create(&newUser).Error; err != nil {
			http.Error(w, "user already exists!", http.StatusBadRequest)
			return
		}


		var cookie = &http.Cookie{
			Name: "token",
			Value: token,
			Path: "/",
			SameSite: http.SameSiteLaxMode,
			HttpOnly: true,
			Secure: true,
		}

		http.SetCookie(w, cookie)

		http.Redirect(w, r, "/", http.StatusFound)
	}
	
	userData, err := utils.ParseCookie(r)

	if err != nil || userData.ID == "" {
		log(false, "signup").Render(r.Context(), w)
		return
	}

	var user models.User

	if err := models.DB.First(&user, "id = ?", userData.ID).Error; err != nil {
		log(false, "signup").Render(r.Context(), w)
		return
	}

	log(true, "").Render(r.Context(), w)
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

		var user models.User
		if err := models.DB.First(&user, "username = ?", username).Error; err != nil {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.Password.String), []byte(password)); err != nil {
			http.Error(w, "wrong password", http.StatusBadRequest)
			return

		} else {

			var claims = &models.Claims{
				ID: user.ID,
			}

			token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(os.Getenv("JWT_SECRET")))

			if err != nil {
				http.Error(w, "couldnt authenticate you", http.StatusInternalServerError)
				return
			}

			var cookie = &http.Cookie{
				Name: "token",
				Value: token,
				Path: "/",
				SameSite: http.SameSiteLaxMode,
				HttpOnly: true,
				Secure: true,
			}
		
			http.SetCookie(w, cookie)
			http.Redirect(w, r, "/", http.StatusFound)
		}
	}
	

	userData, err := utils.ParseCookie(r)

	if err != nil || userData.ID == "" {
		log(false, "login").Render(r.Context(), w)
		return
	}

	var user models.User
	if err := models.DB.First(&user, "id = ?", userData.ID).Error; err != nil {
		log(false, "login").Render(r.Context(), w)
		return
	}

	log(true, "").Render(r.Context(), w)
}


func LogOffH(w http.ResponseWriter, r *http.Request) {
	var cookie = &http.Cookie{
		Name: "token",
		Value: "",
		Path: "/",
		SameSite: http.SameSiteLaxMode,
	}

	http.SetCookie(w, cookie)
}


func EmailVerificationSendH(w http.ResponseWriter, r *http.Request) {
	userData, err := utils.ParseCookie(r)

	if err != nil {
		http.Error(w, "couldnt retrieve user data", http.StatusInternalServerError)
		return
	}

	var user models.User
	models.DB.Preload("EmailVerification").First(&user, "id = ?", userData.ID)

	
	if r.Method == http.MethodPost {
		if user.Verified {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("you are already verified!"))
			return
		}


		if r.URL.Query().Get("send-new") == "true" {
			models.DB.Where("user_id = ? AND type = ?", user.ID, types.EMAIL_VERIFY).Delete(&models.AuthVerification{})
			models.DB.Model(&user).Association("EmailVerification").Append(&models.AuthVerification{
				UserID: user.ID,
				Type: types.EMAIL_VERIFY,
				Token: utils.GenerateToken(),
				Expiry: time.Now().Add(time.Hour),
			})
		}


		if err := utils.SendVerificationEmail(user, types.EMAIL_VERIFY, r.Host); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("couldnt send the email"))
			return
		} else {
			w.Write([]byte("email sent"))
			return
		}
	}
	

	email_verification_send(user).Render(r.Context(), w)
}


func EmailVerificationH(w http.ResponseWriter, r *http.Request) {
	userData, err := utils.ParseCookie(r)

	if err != nil {
		http.Error(w, "couldnt retrieve user data", http.StatusInternalServerError)
		return
	}

	var user models.User
	models.DB.Preload("EmailVerification").First(&user, "id = ?", userData.ID)


	if user.Verified {
		w.Write([]byte("You are already verified!"))
		return
	}

	if time.Now().After(user.EmailVerification.Expiry) {
		http.Error(w, "your token expired!", http.StatusBadRequest)
		return
	}


	var token = r.URL.Query().Get("token")

	if token != user.EmailVerification.Token {

		http.Error(w, "this isnt the right token!", http.StatusBadRequest)
		return

	} else {

		user.Verified = true
		user.Notifications = []models.Notification{
			{
				Message: "Your account has been verified",
				Date: time.Now(),
				Link: "/",
				Type: types.ACC_VERIFIED,
				NotifFrom: "app",
			},
		}
		models.DB.Save(&user)

		models.DB.Where("user_id = ? AND type = ?", user.ID, types.EMAIL_VERIFY).Delete(&models.AuthVerification{})
		models.DB.Where("user_id = ? AND notif_from = ? AND type = ?", user.ID, "app", types.ACC_VERIFY_NEED).Delete(&models.Notification{})
	}
}


func PasswordResetH(w http.ResponseWriter, r *http.Request) {
	userData, err := utils.ParseCookie(r)

	if err != nil {
		http.Error(w, "couldnt retrieve user data", http.StatusInternalServerError)
		return
	}

	var user models.User
	models.DB.Preload("PasswordVerification").First(&user, "id = ?", userData.ID)


	if r.Method == http.MethodPost {
		user.PasswordVerification = &models.AuthVerification{
			UserID: user.ID,
			Type: types.PASSWORD_RESET,
			Token: utils.GenerateToken(),
			Expiry: time.Now().Add(time.Hour),
		}
	
		models.DB.Save(&user)
	
		if err := utils.SendVerificationEmail(user, types.PASSWORD_RESET, r.Host); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("couldnt send the email"))
			return
		} else {
			w.Write([]byte("email sent"))
			return
		}
	}


	password_reset_send().Render(r.Context(), w)
}


func PasswordNewH(w http.ResponseWriter, r *http.Request) {
	userData, err := utils.ParseCookie(r)

	if err != nil {
		http.Error(w, "couldnt retrieve user data", http.StatusInternalServerError)
		return
	}

	var user models.User
	models.DB.Preload("PasswordVerification").First(&user, "id = ?", userData.ID)


	if r.Method == http.MethodPost {

		r.ParseForm()


		if user.PasswordVerification == nil {
			http.Error(w, "you didnt send a request to reset your password", http.StatusBadRequest)
			return
		}

		if time.Now().After(user.PasswordVerification.Expiry) {
			http.Error(w, "the token expired!", http.StatusBadRequest)
			return
		}

		var token = r.URL.Query().Get("token")

		if token != user.PasswordVerification.Token {
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
			http.Error(w, "couldnt save your password", http.StatusInternalServerError)
			return
		}
	
		user.Password = sql.NullString{String: string(bytePassword), Valid: true}
		models.DB.Save(&user)

		models.DB.Where("user_id = ? AND type = ?", user.ID, types.PASSWORD_RESET).Delete(&models.AuthVerification{})
	}


	password_forgot(user).Render(r.Context(), w)
}