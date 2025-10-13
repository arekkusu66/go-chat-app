package utils

import (
	crrand "crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"gochat/models"
	"gochat/types"
	"math/big"
	"net/http"
	"net/smtp"
	"os"
	"regexp"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)


func NewJWT(id string) (string, error) {
	var claims = &models.Claims{
		ID: id,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 5)),
		},
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(os.Getenv("JWT_SECRET")))
}


func refreshToken(jwtToken string) (string, error) {
	var claims = &models.Claims{}

	token, err := jwt.ParseWithClaims(jwtToken, claims, func(t *jwt.Token) (any, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})

	if err != nil {
		if err.Error() != "Token is expired" {
			return NewJWT(claims.ID)
		}
	}

	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.After(time.Now()) {
		return NewJWT(claims.ID)
	}

	if token != nil && token.Valid {
		return jwtToken, nil
	}

	return "", errors.New("invalid token")
}


func ParseCookie(r *http.Request) (*models.Claims, error) {
	tokenData, err := r.Cookie("token")

	if err != nil {
		return nil, err
	}

	tokenString, err := refreshToken(tokenData.Value)

	if err != nil {
		return nil, err
	}

	token, err := jwt.ParseWithClaims(tokenString, &models.Claims{}, func(t *jwt.Token) (any, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})

	if err != nil {
		return nil, err
	}

	return token.Claims.(*models.Claims), nil
}


func IsReply(user models.User, replyID string) bool {
	if replyID == "" || replyID == "0" {
		return false
	}

	var message models.Message
	if err := models.DB.First(&message, replyID).Error; err != nil && err == gorm.ErrRecordNotFound {
		return false
	}

	if message.UserID == user.ID {
		return true
	}

	return false
}


func GenerateToken() string {
	const chars string = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"

	var ret = make([]byte, 8)

	for i := range 8 {
		num, _ := crrand.Int(crrand.Reader, big.NewInt(int64(len(chars))))

		ret[i] = chars[num.Int64()]
	}

	return string(ret)
}


func GenerateOauthState() string {
	var b = make([]byte, 16)

	crrand.Read(b)

	return base64.StdEncoding.EncodeToString(b)
}


func SendVerificationEmail(user models.User, verifType, origin string) error {

	var verifLink, subject, body string

	switch verifType {
		case types.EMAIL_VERIFY:

			verifLink = fmt.Sprintf("http://%s/email/verification?token=%s", origin, user.EmailVerification.Token)
			subject = "Subject: Email verification link" 
			body = fmt.Sprintf("<a href=\"%s\">Verify email</a>", verifLink)

		case types.PASSWORD_RESET:

			verifLink = fmt.Sprintf("http://%s/password/new?token=%s", origin, user.PasswordVerification.Token)
			subject = "Subject: Password reset link"
			body = fmt.Sprintf("<a href=\"%s\">Reset password</a>", verifLink)
	}

	var (
		auth = smtp.PlainAuth("", os.Getenv("EMAIL_FROM"), os.Getenv("EMAIL_APP_PASSWORD"), "smtp.gmail.com")
		headers = `Content-Type:text/html;charset="UTF-8";`
		message = subject + "\n" + headers + "\n\n" + body
	)

	return smtp.SendMail("smtp.gmail.com:587", auth, os.Getenv("EMAIL_FROM"), []string{user.Email}, []byte(message))
}


func Validate(credType, cred string) bool {
	switch credType {
		case "email":
			return regexp.MustCompile("[a-z0-9!#$%&'*+/=?^_`{|}~-]+(?:\\.[a-z0-9!#$%&'*+/=?^_`{|}~-]+)*@(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\\.)+[a-z0-9](?:[a-z0-9-]*[a-z0-9])?").MatchString(cred)
		
		case "password":
			var (
				c1 = len(regexp.MustCompile(`(\W)`).FindAllString(cred, -1)) >= 4
				c2 = !regexp.MustCompile(`(\s)`).MatchString(cred)
				c3 = len(regexp.MustCompile(`([a-z])`).FindAllString(cred, -1)) >= 4
				c4 = len(regexp.MustCompile(`([A-Z])`).FindAllString(cred, -1)) >= 4
				c5 = len(regexp.MustCompile(`([0-9])`).FindAllString(cred, -1)) >= 4
			)
			
			return c1 && c2 && c3 && c4 && c5

		case "username":
			return regexp.MustCompile(`^[\w_]{4,16}$`).MatchString(cred)

		default:
			return false
	}
}


func SettingMod(r *http.Request, user models.User, settingType, dbField string) error {
	switch r.PostFormValue(settingType) {
		case "yes":
			models.DB.Model(&models.Setting{}).Where("user_id = ?", user.ID).Update(dbField, true)
			return nil
		case "no":
			models.DB.Model(&models.Setting{}).Where("user_id = ?", user.ID).Update(dbField, false)
			return nil
		default:
			return errors.New("invalid action")
	}
}