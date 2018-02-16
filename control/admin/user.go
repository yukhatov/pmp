package admin

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"bitbucket.org/tapgerine/pmp/control/models"
	"github.com/asaskevich/govalidator"
)

type responseAdminUserEdit struct {
	Item      models.User
	IsEditing bool
	Success   bool
	Errors    []string
}

func AdminUserCreateHandler(w http.ResponseWriter, r *http.Request) {
	var item models.User
	t, _ := template.ParseFiles(
		"control/templates/main.html",
		"control/templates/user/edit.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)
	t.ExecuteTemplate(w, "main", responseAdminUserEdit{Item: item})
}

func AdminUserEditHandler(w http.ResponseWriter, r *http.Request) {
	success := len(r.URL.Query().Get("success")) > 0

	userID, userIDErr := getUintIDFromRequest(r, "user_id")
	isNewRecord := userIDErr != nil || userID == 0
	isEditing := !isNewRecord

	item := &models.User{}

	if r.Method == "POST" {
		r.ParseForm()

		if isNewRecord {
			item.PopulateData(r)
			item.Role = r.Form.Get("role")
		} else {
			item.GetByID(userID)
			item.Role = r.Form.Get("role")
			item.UpdateData(r)
		}

		if _, err := govalidator.ValidateStruct(item); err != nil {
			errorReplaced := strings.Replace(err.Error(), ";", "|", -1)
			http.Redirect(w, r, fmt.Sprintf("/admin/user/%d/edit/?error=%s", userID, errorReplaced), 302)
			return
		} else {
			success = true
			if isNewRecord {
				item.Create()
			} else {
				item.Save()
			}
		}
		http.Redirect(w, r, fmt.Sprintf("/admin/user/%d/edit/?success=true", userID), 302)
		return
	}
	if !isNewRecord {
		item.GetByID(userID)
	}

	t, _ := template.ParseFiles(
		"control/templates/main.html",
		"control/templates/user/edit.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)
	t.ExecuteTemplate(w, "main", responseAdminUserEdit{
		Item:      *item,
		IsEditing: isEditing,
		Success:   success,
	})
}
