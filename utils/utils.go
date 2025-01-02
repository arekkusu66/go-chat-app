package utils

import (
	crrand "crypto/rand"
	"fmt"
	"gochat/models"
	"gochat/types"
	"math/big"
	"net/http"
	"net/smtp"
	"os"
	"regexp"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)


var Wupg = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}


func ParseCookie(r *http.Request) (*models.Claims, error) {
	cookie, err := r.Cookie("token")

	if err != nil {
		return nil, err
	}

	token, err := jwt.ParseWithClaims(cookie.Value, &models.Claims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})

	if err != nil {
		return nil, err
	}

	return token.Claims.(*models.Claims), nil
}


func IsReply(user models.User, replyID uint) bool {
	if replyID == 0 {
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

	for i := 0; i < 8; i++ {
		num, _ := crrand.Int(crrand.Reader, big.NewInt(int64(len(chars))))

		ret[i] = chars[num.Int64()]
	}

	return string(ret)
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