package admin

import (
	"html/template"
	"net/http"
	"time"

	"bitbucket.org/tapgerine/pmp/control/database"
	"bitbucket.org/tapgerine/pmp/control/models"
)

type responseActionLogList struct {
	Items     []*models.ActionLog
	TodayDate string
}

func ActionLogListHandler(w http.ResponseWriter, r *http.Request) {
	var list []*models.ActionLog
	database.Postgres.Preload("User").Where("created_at >= (now() - interval '1 day')").Order("created_at desc").Find(&list)

	t, _ := template.ParseFiles(
		"control/templates/main.html",
		"control/templates/action_log.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)
	t.ExecuteTemplate(w, "main", responseActionLogList{
		Items: list, TodayDate: time.Now().Format("2006-01-02"),
	})
}
