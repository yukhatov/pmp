package admin

import (
	"html/template"
	"net/http"

	"strings"

	"fmt"

	"time"

	"os"

	"os/exec"

	"io"

	"bytes"

	"io/ioutil"

	"bitbucket.org/tapgerine/pmp/control/database"
	"bitbucket.org/tapgerine/pmp/control/models"
	log "github.com/Sirupsen/logrus"
	"github.com/asaskevich/govalidator"
)

type responsePublisherInvoiceEdit struct {
	Item       models.PublisherInvoice
	Publishers []*models.Publisher
	IsEditing  bool
	Success    bool
	Errors     []string
}

type responseAdvertiserInvoiceEdit struct {
	Item        models.AdvertiserInvoice
	Advertisers []*models.Advertiser
	IsEditing   bool
	Success     bool
	Errors      []string
}

func PublisherInvoiceListHandler(w http.ResponseWriter, r *http.Request) {
	var list []*models.PublisherInvoice
	database.Postgres.Preload("Publisher").Find(&list)

	t, _ := template.ParseFiles(
		"control/templates/main.html",
		"control/templates/invoice/publisher_invoice_list.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)
	t.ExecuteTemplate(w, "main", list)
}

func PublisherInvoiceCreateHandler(w http.ResponseWriter, r *http.Request) {
	item := &models.PublisherInvoice{}

	var publisherList []*models.Publisher
	database.Postgres.Find(&publisherList)

	t, _ := template.ParseFiles(
		"control/templates/main.html",
		"control/templates/invoice/publisher_invoice_edit.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)
	t.ExecuteTemplate(w, "main", responsePublisherInvoiceEdit{
		Item:       *item,
		Publishers: publisherList,
		IsEditing:  false,
		Errors:     strings.Split(r.URL.Query().Get("error"), "|"),
	})
}

func PublisherInvoiceEditHandler(w http.ResponseWriter, r *http.Request) {
	var success bool
	id, err := getUintIDFromRequest(r, "id")
	isNewRecord := err != nil || id == 0

	item := &models.PublisherInvoice{}

	var publisherList []*models.Publisher
	database.Postgres.Find(&publisherList)

	if r.Method == "POST" {
		r.ParseMultipartForm(32 << 20)

		if isNewRecord {
			item.PopulateData(r)
		} else {
			item.GetByID(id)
			item.UpdateData(r)
		}

		// TODO: handle error from r.FormFile
		file, header, _ := r.FormFile("invoice_file")
		if file != nil {
			// TODO: delete previous file
			defer file.Close()
			uniqueID := fmt.Sprintf("pub_%d", item.PublisherID)
			filePath, err := database.FilesManager.UploadFile(file, header, uniqueID)
			if err != nil {
				log.WithError(err).Warn("Can't save publisher invoice")
				return
			}
			item.FileName = header.Filename
			item.FilePath = filePath
		}

		if _, err := govalidator.ValidateStruct(item); err != nil {
			errorReplaced := strings.Replace(err.Error(), ";", "|", -1)
			if isNewRecord {
				http.Redirect(w, r, fmt.Sprintf("/invoice/publisher/create/?error=%s", errorReplaced), 302)
			} else {
				http.Redirect(w, r, fmt.Sprintf("/invoice/publisher/%d/edit/?error=%s", item.ID, errorReplaced), 302)
			}
			return
		} else {
			if isNewRecord {
				item.Create()
			} else {
				item.Save()
			}
		}

		http.Redirect(w, r, fmt.Sprintf("/invoice/publisher/%d/edit/?success=true", item.ID), 302)
		return

	} else {
		if isNewRecord {
			// TODO: error handling
			panic(err)
		}

		success = len(r.URL.Query().Get("success")) > 0

		item.GetByID(id)
	}

	t, _ := template.ParseFiles(
		"control/templates/main.html",
		"control/templates/invoice/publisher_invoice_edit.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)
	t.ExecuteTemplate(w, "main", responsePublisherInvoiceEdit{
		Item:       *item,
		Publishers: publisherList,
		IsEditing:  true,
		Success:    success,
		Errors:     strings.Split(r.URL.Query().Get("error"), "|"),
	})
}

func PublisherInvoiceStatusChangeHandler(w http.ResponseWriter, r *http.Request) {
	id, err := getUintIDFromRequest(r, "id")
	if err != nil {
		// TODO: add proper error handling if needed
		w.WriteHeader(400)
		w.Write([]byte("error"))
		return
	}
	item := &models.PublisherInvoice{}
	item.GetByID(id)

	item.Status, _ = getStringFromRequest(r, "status")

	if item.Status == "Paid" {
		item.DatePaid = time.Now()
	}

	item.Save()
	w.Write([]byte("success"))
}

func PublisherInvoiceViewHandler(w http.ResponseWriter, r *http.Request) {
	id, err := getUintIDFromRequest(r, "id")
	if err != nil {
		// TODO: add proper error handling if needed
		panic(err)
	}
	item := &models.PublisherInvoice{}
	item.GetByID(id)

	file, err := database.FilesManager.ReadFile(item.FilePath)
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-type", "application/pdf")

	if _, err := file.WriteTo(w); err != nil {
		fmt.Fprintf(w, "%s", err)
	}
}

func subtractTwoFloats(first, second float64) float64 {
	return first - second
}

func PublisherInvoiceDetailsHandler(w http.ResponseWriter, r *http.Request) {
	fm1 := template.FuncMap{"subtractTwoFloats": subtractTwoFloats}

	id, err := getUintIDFromRequest(r, "id")
	if err != nil {
		// TODO: add proper error handling if needed
		panic(err)
	}
	item := &models.PublisherInvoice{}
	item.GetByID(id)

	t, _ := template.New("main").Funcs(fm1).ParseFiles(
		"control/templates/main.html",
		"control/templates/invoice/publisher_invoice_details.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)
	t.ExecuteTemplate(w, "main", item)
}

func AdvertiserInvoiceListHandler(w http.ResponseWriter, r *http.Request) {
	var list []*models.AdvertiserInvoice
	database.Postgres.Preload("Advertiser").Find(&list)

	t, _ := template.ParseFiles(
		"control/templates/main.html",
		"control/templates/invoice/advertiser_invoice_list.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)
	t.ExecuteTemplate(w, "main", list)
}

func AdvertiserInvoiceCreateHandler(w http.ResponseWriter, r *http.Request) {
	item := &models.AdvertiserInvoice{}

	var advertiserList []*models.Advertiser
	database.Postgres.Find(&advertiserList)

	t, _ := template.ParseFiles(
		"control/templates/main.html",
		"control/templates/invoice/advertiser_invoice_edit.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)
	t.ExecuteTemplate(w, "main", responseAdvertiserInvoiceEdit{
		Item:        *item,
		Advertisers: advertiserList,
		IsEditing:   false,
		Errors:      strings.Split(r.URL.Query().Get("error"), "|"),
	})
}

func AdvertiserInvoiceEditHandler(w http.ResponseWriter, r *http.Request) {
	var success bool
	id, err := getUintIDFromRequest(r, "id")
	isNewRecord := err != nil || id == 0

	item := &models.AdvertiserInvoice{}

	var advertiserList []*models.Advertiser
	database.Postgres.Find(&advertiserList)

	if r.Method == "POST" {
		r.ParseForm()

		if isNewRecord {
			item.PopulateData(r)
		} else {
			item.GetByID(id)
			item.UpdateData(r)
		}

		if _, err := govalidator.ValidateStruct(item); err != nil {
			errorReplaced := strings.Replace(err.Error(), ";", "|", -1)
			if isNewRecord {
				http.Redirect(w, r, fmt.Sprintf("/invoice/advertiser/create/?error=%s", errorReplaced), 302)
			} else {
				http.Redirect(w, r, fmt.Sprintf("/invoice/advertiser/%d/edit/?error=%s", item.ID, errorReplaced), 302)
			}
			return
		} else {
			if isNewRecord {
				item.Create()
			} else {
				item.Save()
			}
		}

		http.Redirect(w, r, fmt.Sprintf("/invoice/advertiser/%d/edit/?success=true", item.ID), 302)
		return

	} else {
		if isNewRecord {
			// TODO: error handling
			panic(err)
		}

		success = len(r.URL.Query().Get("success")) > 0

		item.GetByID(id)
	}

	t, _ := template.ParseFiles(
		"control/templates/main.html",
		"control/templates/invoice/advertiser_invoice_edit.html",
		"control/templates/header.html",
		"control/templates/menu.html",
	)
	t.ExecuteTemplate(w, "main", responseAdvertiserInvoiceEdit{
		Item:        *item,
		Advertisers: advertiserList,
		IsEditing:   true,
		Success:     success,
		Errors:      strings.Split(r.URL.Query().Get("error"), "|"),
	})
}

func AdvertiserInvoiceStatusChangeHandler(w http.ResponseWriter, r *http.Request) {
	id, err := getUintIDFromRequest(r, "id")
	if err != nil {
		// TODO: add proper error handling if needed
		w.WriteHeader(400)
		w.Write([]byte("error"))
		return
	}
	item := &models.AdvertiserInvoice{}
	item.GetByID(id)

	item.Status, _ = getStringFromRequest(r, "status")

	if item.Status == "Paid" {
		item.DatePaid = time.Now()
	}

	item.Save()
	w.Write([]byte("success"))
}

func AdvertiserGenerateInvoiceHandler(w http.ResponseWriter, r *http.Request) {
	id, err := getUintIDFromRequest(r, "id")
	if err != nil {
		// TODO: add proper error handling if needed
		w.WriteHeader(400)
		w.Write([]byte("error"))
		return
	}
	item := &models.AdvertiserInvoice{}
	item.GetByID(id)

	t, _ := template.ParseFiles(
		"control/templates/invoice/advertiser_invoice_template.html",
	)

	htmlTemplatePath := fmt.Sprintf("/tmp/invoice_%d.html", item.ID)
	pdfPath := fmt.Sprintf("/tmp/invoice_%d.pdf", item.ID)

	f, err := os.OpenFile(htmlTemplatePath, os.O_WRONLY|os.O_CREATE, 0666)
	defer f.Close()

	t.ExecuteTemplate(f, "main", item)
	cmd := exec.Command("wkhtmltopdf", "--lowquality", htmlTemplatePath, pdfPath)
	cmd.Run()

	pdf, _ := os.OpenFile(pdfPath, os.O_RDWR, 0644)
	defer pdf.Close()

	var pdfContent = make([]byte, 1024*1024)
	for {
		_, err = pdf.Read(pdfContent)

		// break if finally arrived at end of file
		if err == io.EOF {
			break
		}

		// break if error occured
		if err != nil && err != io.EOF {
			if err != nil {
				fmt.Println(err)
			}
			break
		}
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment;filename=Invoice.pdf")
	io.Copy(w, bytes.NewReader(pdfContent))
}

func AdvertiserInvoiceViewHandler(w http.ResponseWriter, r *http.Request) {
	id, err := getUintIDFromRequest(r, "id")
	if err != nil {
		// TODO: add proper error handling if needed
		w.WriteHeader(400)
		w.Write([]byte("error"))
		return
	}
	item := &models.AdvertiserInvoice{}
	item.GetByID(id)

	t, _ := template.ParseFiles(
		"control/templates/invoice/advertiser_invoice_template.html",
	)

	htmlTemplatePath := fmt.Sprintf("/tmp/invoice_%d.html", item.ID)
	pdfPath := fmt.Sprintf("/tmp/invoice_%d.pdf", item.ID)

	f, err := os.OpenFile(htmlTemplatePath, os.O_WRONLY|os.O_CREATE, 0666)
	defer f.Close()

	t.ExecuteTemplate(f, "main", item)
	cmd := exec.Command("wkhtmltopdf", "--lowquality", htmlTemplatePath, pdfPath)
	cmd.Run()

	pdf, _ := ioutil.ReadFile(pdfPath)
	b := bytes.NewBuffer(pdf)

	w.Header().Set("Content-type", "application/pdf")
	if _, err := b.WriteTo(w); err != nil {
		fmt.Fprintf(w, "%s", err)
	}
}
