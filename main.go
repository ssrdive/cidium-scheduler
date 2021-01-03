package main

import (
	"database/sql"
	"flag"
	"fmt"
	"net/smtp"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jasonlvhit/gocron"
	"github.com/ssrdive/sprinter"
)

func main() {
	dsn := flag.String("dsn", "user:password@tcp(host)/database_name?parseTime=true", "MySQL data source name")
	from := flag.String("from", "agrivestlimited@gmail.com", "Address to send emails from")
	password := flag.String("password", "password", "Password to authenticate")
	flag.Parse()

	gocron.Every(1).Day().At("01:00").Do(runDayEnd, *dsn, *from, *password)

	<-gocron.Start()
}

func runDayEnd(dsn, from, password string) {
	db, err := openDB(dsn)
	if err != nil {
		fmt.Println(err)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		_ = tx.Commit()
		defer db.Close()
	}()

	today := time.Now().Format("2006-01-02")

	uConts, elapsed, err := sprinter.Run(today, "", false, tx)
	if err != nil {
		tx.Rollback()
		return
	}
	emailHTML := "<html><head><style> body { font-family: arial, sans-serif; } table { border-collapse: collapse; width: 100%; } td, th { border: 1px solid #dddddd; text-align: left; padding: 8px; } </style></head><body>"
	emailHTML = emailHTML + "<h2 style='font-family: Arial, Helvetica, sans-serif;'>Day end runs " + today + "</h2>"
	emailHTML = emailHTML + "<p style='font-family: Arial, Helvetica, sans-serif;'>Program completed in " + fmt.Sprintf("%f", elapsed.Seconds()) + " seconds</p>"
	emailHTML = emailHTML + `
	<table>
		<thead>
			<th>ID</th>
			<th>Previous Recovery Status</th>
			<th>New Recovery Status</th>
		</thead>
		<tbody>
	`

	for _, contract := range uConts {
		emailHTML = emailHTML + "<tr><td>" + fmt.Sprintf("%d", contract.ContractID) + "</td>" + convertStatusCodeToHTML(contract.RecoveryStatus) + convertStatusCodeToHTML(contract.UpdatedRecoveryStatus) + "</tr>"
	}
	emailHTML = emailHTML + "</tbody></table></body></html>"
	sendEmail(from, "shamal@randeepa.com, manuka.hapugoda@agrivest.lk, kularathna@agrivest.lk", password, "Day End Run Summary "+today, emailHTML)
}

func convertStatusCodeToHTML(code int) string {
	switch code {
	case 1:
		return "<td style='background-color: #199c31; color: #FFFFFF;'>Active</td>"
	case 2:
		return "<td style='background-color: #5e5717; color: #FFFFFF;'>Arrears</td>"
	case 3:
		return "<td style='background-color: #522525; color: #FFFFFF;'>NPL</td>"
	case 4:
		return "<td style='background-color: #f51111; color: #FFFFFF;'>BDP</td>"
	}
	return ""
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	return db, err
}

func sendEmail(from, to, password, subject, body string) error {

	msg := "From: " + from + "\n" +
		"To: " + to + "\n" +
		"Subject: " + subject + "\n\n" +
		body

	err := smtp.SendMail("smtp.gmail.com:587",
		smtp.PlainAuth("", from, password, "smtp.gmail.com"),
		from, []string{to}, []byte(msg))

	if err != nil {
		return err
	}

	return nil
}
