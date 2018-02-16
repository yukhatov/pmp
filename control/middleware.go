package control

import (
	"context"
	"net/http"
	"time"

	"strings"

	"bitbucket.org/tapgerine/pmp/control/models"
)

var routesWithoutAuth = map[string]bool{
	"/login":                         true,
	"/sync":                          true,
	"/jwt":                           true,
	"/merge_statistics":              true,
	"/merge_statistics_events":       true,
	"/merge_statistics_rtb_events":   true,
	"/merge_statistics_rtb_requests": true,
}

func authMiddleware(inner http.Handler) http.Handler {
	mw := func(w http.ResponseWriter, r *http.Request) {
		uri := r.RequestURI

		// has query params
		if strings.Contains(uri, "/jwt_") {
			uri = "/jwt"
		}

		sessionCookie, err := r.Cookie("Session")
		session := &models.Session{}

		if authDontNeeded := routesWithoutAuth[uri]; authDontNeeded == false {
			if err != nil || sessionCookie.Value == "" {
				http.Redirect(w, r, "/login", 302)
				return
			} else {
				session.GetByID(sessionCookie.Value)
				if session.IsExpired() {
					http.Redirect(w, r, "/login", 302)
					return
				}
				if session.UserID != 0 {
					if session.User.Role == "publisher" && !strings.HasPrefix(uri, "/publisher_admin/") {
						http.Redirect(w, r, "/login", 302)
						return
					}
					session.LastLogin = time.Now()
					session.Save()
				} else {
					http.Redirect(w, r, "/login", 302)
					return
				}
			}
		}

		if (uri == "/action_log/" || strings.HasPrefix(uri, "/admin/")) && session.User.Role != "admin" {
			http.Redirect(w, r, "/login", 302)
			return
		}

		ctx := context.WithValue(r.Context(), "session", session)
		ctx = context.WithValue(ctx, "userID", session.UserID)
		r = r.WithContext(ctx)

		inner.ServeHTTP(w, r)
	}
	return http.HandlerFunc(mw)
}
