package auth

import "net/http"

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	sessionCookie := &http.Cookie{
		Name:  "Session",
		Value: "",
	}
	http.SetCookie(w, sessionCookie)
	http.Redirect(w, r, "/login", 302)
}
