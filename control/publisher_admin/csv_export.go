package publisherAdmin

import (
	"net/http"

	"time"

	"fmt"

	"bytes"
	"encoding/csv"

	"strconv"

	"bitbucket.org/tapgerine/pmp/control/models"
)

func PublisherStatsCSVExportHandler(w http.ResponseWriter, r *http.Request) {
	sessionCookie, _ := r.Cookie("Session")
	session := &models.Session{}
	session.GetByID(sessionCookie.Value)

	var startDate, endDate, search string
	if len(r.URL.Query()) > 0 {
		startDate = r.URL.Query().Get("start_date")
		endDate = r.URL.Query().Get("end_date")
		search = r.URL.Query().Get("search")
	}

	if startDate == "" {
		startDate = time.Now().Add(7 * -24 * time.Hour).UTC().Format("2006-01-02")
	}
	if endDate == "" {
		endDate = time.Now().UTC().Format("2006-01-02")
	}

	statsSorted, _ := collectStatistics(session.User.PublisherID, search, session.User.DefaultTimezone, startDate, endDate)

	statsSortedTargeting, _ := collectStatisticsForPublisherLinks(session.User.PublisherID, search, session.User.DefaultTimezone, startDate, endDate)

	for k, v := range statsSortedTargeting {
		statsSorted[k] = v
	}

	b := &bytes.Buffer{}
	csvWriter := csv.NewWriter(b)
	csvWriter.Write([]string{"Name", "Date", "Requests", "Impressions", "FillRate", "Revenue"})
	for _, statsByName := range statsSorted {
		for _, value := range statsByName {
			csvWriter.Write([]string{
				value.PublisherTagName, value.Date, strconv.Itoa(int(value.Requests)),
				strconv.Itoa(int(value.Impressions)), fmt.Sprintf("%.2f", value.FillRate),
				fmt.Sprintf("$%.2f", value.Amount),
			})
		}
	}
	csvWriter.Flush()

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment;filename=Tapgerine_Data_%s_%s.csv", startDate, endDate))
	w.Write(b.Bytes())
}
