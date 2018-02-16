package auth

import (
	"net/http"

	"bitbucket.org/tapgerine/pmp/control/database"

	"bitbucket.org/tapgerine/pmp/control/models"

	"encoding/json"
	"fmt"

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
)

func JWTCheckTokenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		httpCode := 200
		isValid := UserJWTValidate(r.URL.Query().Get("token"))

		if !isValid {
			httpCode = 403
		}

		var response = map[string]bool{
			"success": isValid,
		}

		json_, _ := json.Marshal(response)

		w.WriteHeader(httpCode)
		w.Header().Set("Content-Type", "application/json")
		w.Write(json_)
	}
}

func UserJWTValidate(tokenString string) bool {
	claims := parseClaims(tokenString)

	if claims == nil {
		return false
	}

	return isUserValid(claims["username"], claims["password"])
}

func parseClaims(tokenString string) map[string]string {
	rsaSecret := `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAuoBgprX9+L16R0A1BTR+
ugq2rv14Z5bkdJ2jqZ0Y+5lISKn38PSevvFp/o3hXrtyNiZefnxKfHbfwii23oVK
UrH1IVRcNT+ceolBT5+P7T9/rrEKjPsUYPXx0CTcbkWvB5dXv13jegDuZFckD7wP
z2NGP0UXa3Ptve1lO1PLxk/55pf3+IaForNeb0vm7tY1XTOavs9SCJxd+2rxARdu
Cv7rusmyLZsnEbz7OL33hiT0ewOvCMT1QMqwLUjLx4HZ8WG//QXLFbhgMUf+pjG7
KyQVqaK0UMCZX0olHqzNr3CBpNmXpi8/WwEOrrxSusbpt8o7MpYgwvYiv47r58CY
CwIDAQAB
-----END PUBLIC KEY-----`
	claims := jwt.MapClaims{}

	_, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return jwt.ParseRSAPublicKeyFromPEM([]byte(rsaSecret))
	})

	if err != nil {
		return nil
	}

	result := make(map[string]string)
	result["username"] = claims["username"].(string)
	result["password"] = claims["password"].(string)

	return result
}

func isUserValid(username string, password string) bool {
	var user models.User
	database.Postgres.
		Where(&models.User{
			UserName: username,
		}).First(&user)

	if user.UserName == "" {
		return false
	} else {
		err := bcrypt.CompareHashAndPassword(user.Password, []byte(password))

		if err != nil {
			return false
		} else {
			return true
		}
	}
}

func JWTLoginHandler(w http.ResponseWriter, r *http.Request) {
	claims := parseClaims(r.URL.Query().Get("token"))

	if claims == nil {
		http.Redirect(w, r, "/", 302)
	}

	var user models.User

	database.Postgres.
		Where(&models.User{
			UserName: claims["username"],
		}).First(&user)

	if user.UserName != "" {
		err := bcrypt.CompareHashAndPassword(user.Password, []byte(claims["password"]))

		if err == nil {
			session := &models.Session{}
			session.Create(user.ID)

			sessionCookie := &http.Cookie{
				Name:  "Session",
				Value: session.Key,
			}

			http.SetCookie(w, sessionCookie)

			if user.Role == "publisher" {
				http.Redirect(w, r, "/publisher_admin/", 302)
			} else {
				http.Redirect(w, r, "/", 302)
			}

			return
		}
	}

	http.Redirect(w, r, "/", 302)
}
