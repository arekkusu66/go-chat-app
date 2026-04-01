package app

import (
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/time/rate"
)


type Claims struct {
	ID	uuid.UUID
	jwt.RegisteredClaims
}


func (app *App) NewJWT(id uuid.UUID) (string, error) {
	var claims = &Claims{
		ID: id,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 5)),
		},
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(os.Getenv("JWT_SECRET")))
}


func (app *App) refreshToken(r *http.Request, w http.ResponseWriter) (*Claims, error) {
	tokenData, err := r.Cookie("token")

	if err != nil {
		log.Println("no token in the cookie", err)
		return nil, err
	}

	var claims = &Claims{}

	token, err := jwt.ParseWithClaims(tokenData.Value, claims, func(t *jwt.Token) (any, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			newToken, err := app.NewJWT(claims.ID)

			if err != nil {
				return nil, err
			}

			http.SetCookie(w, &http.Cookie{
				Name: "token",
				Value: newToken,
				Path: "/",
			})

			return claims, nil
		}

		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}


func (app *App) GetUserID(r *http.Request, w http.ResponseWriter) (uuid.UUID, int, error) {
	userDatas, err := app.refreshToken(r, w)

	if err != nil || userDatas.ID == uuid.Nil {
		return uuid.UUID{}, http.StatusInternalServerError, errors.New("couldnt retrieve the user datas")
	}

	return userDatas.ID, 0, nil
}


func (app *App) RateLimiter(id string, t time.Duration) *rate.Limiter {
	if limiter, ok := app.RateLimitClients.Load(id); ok {
		return limiter.(*rate.Limiter)
	}

	var limiter = rate.NewLimiter(rate.Every(t), 1)

	actual, _ := app.RateLimitClients.LoadOrStore(id, limiter)

	return actual.(*rate.Limiter)
}