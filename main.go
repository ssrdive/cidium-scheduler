package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/smtp"
	"os"
	"time"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jasonlvhit/gocron"
	"github.com/ssrdive/sprinter"
)

func main() {
	dsn := flag.String("dsn", "user:password@tcp(host)/database_name?parseTime=true", "MySQL data source name")
	from := flag.String("from", "agrivestlimited@gmail.com", "Address to send emails from")
	password := flag.String("password", "password", "Password to authenticate")
	logPath := flag.String("logpath", "/var/www/agrivest.app/logs/", "Path to create or alter log files")
	flag.Parse()

	gocron.Every(1).Day().At("00:45").Do(runDayEnd, *dsn, *from, *password, *logPath)

	<-gocron.Start()
}

func runDayEnd(dsn, from, password, logPath string) {
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

	dayEndLogFile, err := openLogFile(logPath + today + "_day_end.log")
	if err != nil {
		fmt.Println("Failed to open receipt log file")
		os.Exit(1)
	}
	dayEndLog := log.New(dayEndLogFile, "", log.Ldate|log.Ltime)

	dayEndLog.Println("Starting day end program")

	uConts, elapsed, err := sprinter.Run(today, "", false, tx)
	if err != nil {
		dayEndLog.Printf("Day end failed with error: %v\n", err)
		tx.Rollback()
		return
	}
	dayEndLog.Printf("Day end succes with processed contracts %+v\n", uConts)

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

	dayEndLog.Println("Sending program run summary")

	toList := []string{"shamal@randeepa.com", "agrivest@randeepa.com", "psmfdo@gmail.com"}
	err = sendEmail(toList, from, password, "Day End Run Summary "+today, emailHTML)

	if err != nil {
		dayEndLog.Printf("Failed to send email %+v\n", err)
		return
	}
	dayEndLog.Printf("Summary email sent.")
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

func sendEmail(to []string, from, password, subject, body string) error {
	toHeader := strings.Join(to, ",")
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	msg := "From: " + from + "\n" +
		"To: " + toHeader + "\n" +
		"Subject: " + subject + "\n" + mime +
		body

	err := smtp.SendMail("smtp.gmail.com:587",
		smtp.PlainAuth("", from, password, "smtp.gmail.com"),
		from, to, []byte(msg))

	if err != nil {
		return err
	}

	return nil
}

func openLogFile(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
}
