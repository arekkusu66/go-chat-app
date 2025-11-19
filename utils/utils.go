package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	crrand "crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"gochat/types"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gopkg.in/gomail.v2"
)

type Claims struct {
	ID	uuid.UUID
	jwt.RegisteredClaims
}


func NewJWT(id uuid.UUID) (string, error) {
	var claims = &Claims{
		ID: id,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 5)),
		},
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(os.Getenv("JWT_SECRET")))
}


func refreshToken(jwtToken string) (string, error) {
	var claims = &Claims{}

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


func parseCookie(r *http.Request) (*Claims, error) {
	tokenData, err := r.Cookie("token")

	if err != nil {
		log.Println("no token in the cookie", err)
		return nil, err
	}

	tokenString, err := refreshToken(tokenData.Value)

	if err != nil {
		log.Println("couldnt refresh token", err)
		return nil, err
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (any, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})

	if err != nil {
		log.Println("couldnt parse the token", err)
		return nil, err
	}

	return token.Claims.(*Claims), nil
}


func GetUserID(r *http.Request) (uuid.UUID, int, error) {
	userDatas, err := parseCookie(r)

	if err != nil || userDatas.ID == uuid.Nil {
		return uuid.UUID{}, http.StatusInternalServerError, errors.New("couldnt retrieve the user datas")
	}

	return userDatas.ID, 0, nil
}

func ServeDir(dirname string, mux *http.ServeMux) {
	mux.Handle(
		fmt.Sprintf("%s/", dirname), 
		http.StripPrefix(fmt.Sprintf("%s/", dirname), 
		http.FileServer(
			http.Dir(fmt.Sprintf(".%s", dirname)))))
}


func EncryptAES(plainText string) (string, error) {
	block, err := aes.NewCipher([]byte(os.Getenv("AES_SECRET")))

	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)

	if err != nil {
		return "", err
	}

	var nonce = make([]byte, gcm.NonceSize())

	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	var cipherText = gcm.Seal(nonce, nonce, []byte(plainText), nil)

	return base64.StdEncoding.EncodeToString(cipherText), nil
}


func DecryptAES(cipherText string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(cipherText)

	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher([]byte(os.Getenv("AES_SECRET")))

	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)

	if err != nil {
		return "", err
	}

	var nonceSize = gcm.NonceSize()

	nonce, text := data[:nonceSize], data[nonceSize:]

	plainText, err := gcm.Open(nil, nonce, text, nil)

	if err != nil {
		return "", err
	}

	return string(plainText), nil
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

	return base64.URLEncoding.EncodeToString(b)
}


func SendVerificationEmail(verifType types.Verification, token string, email, origin string) error {
	var m = gomail.NewMessage()

	m.SetHeader("From", os.Getenv("GMAIL_EMAIL_FROM"))
	m.SetHeader("To", email)

	switch verifType {
		case types.EMAIL_VERIFICATION:
			var verifLink = fmt.Sprintf("http://%s/email/verification?token=%s", origin, token)

			m.SetHeader("Subject", "Email verification link")
			m.SetBody(`text/html;charset="UTF-8"`, fmt.Sprintf("<a href=\"%s\">Verify email</a>", verifLink))

		case types.PASSWORD_RESET:
			var verifLink = fmt.Sprintf("http://%s/password/new?token=%s", origin, token)

			m.SetHeader("Subject", "Password reset link")
			m.SetBody(`text/html;charset="UTF-8"`, fmt.Sprintf("<a href=\"%s\">Reset password</a>", verifLink))
	}

	var d = gomail.NewDialer("smtp.gmail.com", 587, os.Getenv("GMAIL_EMAIL_FROM"), os.Getenv("GMAIL_APP_PASSWORD"))

	return d.DialAndSend(m)
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


func GetFuncInfo() string {
	pc, _, line, _ := runtime.Caller(1)
	
	var fnInfo = runtime.FuncForPC(pc)

	return "Line " + fmt.Sprint(line) + " " + fnInfo.Name()
}