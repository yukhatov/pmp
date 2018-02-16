package auth

import (
	"html/template"
	"net/http"

	"bitbucket.org/tapgerine/pmp/control/database"

	"bitbucket.org/tapgerine/pmp/control/models"

	"golang.org/x/crypto/bcrypt"
)

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var error string
	if r.Method == "POST" {
		r.ParseForm()
		username := r.Form.Get("username")
		password := r.Form.Get("password")
		//hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		var user models.User
		database.Postgres.
			Where(&models.User{
				UserName: username,
			}).First(&user)

		if user.UserName == "" {
			error = "Your username or password is incorrect"
		} else {
			err := bcrypt.CompareHashAndPassword(user.Password, []byte(password))
			if err != nil {
				error = "Your username or password is incorrect"
			} else {
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
	}

	t, _ := template.ParseFiles(
		"control/templates/auth/login.html",
	)
	t.ExecuteTemplate(w, "login", error)
}
