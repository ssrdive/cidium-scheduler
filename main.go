package main

import (
	"crypto/tls"
	"database/sql"
	"flag"
	"fmt"
	"github.com/ssrdive/mysequel"
	"log"
	"math"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jasonlvhit/gocron"
	"github.com/ssrdive/sprinter"
)

func main() {
	dsn := flag.String("dsn", "user:password@tcp(host)/database_name?parseTime=true", "MySQL data source name")
	from := flag.String("from", "agrivest.mailer@randeepa.cloud", "Address to send emails from")
	password := flag.String("password", "password", "Password to authenticate")
	logPath := flag.String("logpath", "/var/www/agrivest.app/logs/", "Path to create or alter log files")
	flag.Parse()

	gocron.Every(1).Day().At("00:45").Do(runDayEnd, *dsn, *from, *password, *logPath)
	gocron.Every(1).Day().At("06:00").Do(sendCWAPendingList, *dsn, *from, *password)
	gocron.Every(1).Day().At("06:00").Do(sendFCPendingList, *dsn, *from, *password)
	gocron.Every(1).Day().At("01:15").Do(calculateDefault, *dsn, *logPath)

	<-gocron.Start()
}

func sendFCPendingList(dsn, from, password string) {
	db, err := openDB(dsn)
	if err != nil {
		fmt.Println(err)
		return
	}

	stmt := `
		SELECT C.id, DATEDIFF(NOW(), CST.transition_date) AS do_not_sent_for, U.name AS credit_officer, U2.name AS recovery_officer, C.customer_name, C.price
		FROM contract C
		LEFT JOIN contract_state CS ON CS.id = C.contract_state_id
		LEFT JOIN contract_state_transition CST ON CST.to_contract_state_id = C.contract_state_id
		LEFT JOIN user U ON U.id = C.credit_officer_id
		LEFT JOIN user U2 on U2.id = C.recovery_officer_id
		WHERE CS.state_id = 3
		ORDER BY do_not_sent_for DESC
	`
	rows, err := db.Query(stmt)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer rows.Close()

	emailHTML := "<html><head><style> body { font-family: arial, sans-serif; } table { border-collapse: collapse; width: 100%; } td, th { border: 1px solid #dddddd; text-align: left; padding: 8px; } </style></head><body>"
	emailHTML = emailHTML + "<h2 style='font-family: Arial, Helvetica, sans-serif;'>Pending DOs To Be Issued</h2>"
	emailHTML = emailHTML + `
	<table>
		<thead>
			<th>ID</th>
			<th>Days</th>
			<th>Credit Officer</th>
			<th>Recovery Officer</th>
			<th>Customer Name</th>
			<th>Price</th>
		</thead>
		<tbody>
	`

	for rows.Next() {
		var id, creditOfficer, recoveryOfficer, customerName string
		var doNotIssuedFor, price int

		err = rows.Scan(&id, &doNotIssuedFor, &creditOfficer, &recoveryOfficer, &customerName, &price)
		if err != nil {
			fmt.Println(err)
			return
		}

		emailHTML = emailHTML + fmt.Sprintf(`
			<tr>
				<td>%s</td>
				%s%d</td>
				<td>%s</td>
				<td>%s</td>
				<td>%s</td>
				<td>%d</td>
			</tr>
		`, id, getTrColorHTMLforDates(doNotIssuedFor), doNotIssuedFor, creditOfficer, recoveryOfficer, customerName, price)
	}

	emailHTML = emailHTML + "</tbody></table></body></html>"

	toList := []string{"shamal@randeepa.com", "dimuthu@randeepa.com", "samanthi.chandrika@randeepa.com", "lehan.randesh@randeepa.com", "chandika.prasad@randeepa.com", "kularathna@agrivest.lk", "minura.maduwantha@agrivest.lk", "tharushika.samarathunga@agrivest.lk", "dumeshika.aluvihare@agrivest.lk"}
	err = sendEmail(toList, from, password, "Pending DOs To Be Issued", emailHTML)

	if err != nil {
		fmt.Printf("Failed to send email %+v\n", err)
		return
	}
}

func sendCWAPendingList(dsn, from, password string) {
	db, err := openDB(dsn)
	if err != nil {
		fmt.Println(err)
		return
	}

	stmt := `
		SELECT C.id, DATEDIFF(NOW(), CST.transition_date) AS file_incomplete_for, U.name AS credit_officer, U2.name AS recovery_officer, C.customer_name, C.price
		FROM contract C
		LEFT JOIN contract_state CS ON CS.id = C.contract_state_id
		LEFT JOIN contract_state_transition CST ON CST.to_contract_state_id = C.contract_state_id
		LEFT JOIN user U ON U.id = C.credit_officer_id
		LEFT JOIN user U2 on U2.id = C.recovery_officer_id
		WHERE CS.state_id = 2
		ORDER BY file_incomplete_for DESC
	`
	rows, err := db.Query(stmt)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer rows.Close()

	emailHTML := "<html><head><style> body { font-family: arial, sans-serif; } table { border-collapse: collapse; width: 100%; } td, th { border: 1px solid #dddddd; text-align: left; padding: 8px; } </style></head><body>"
	emailHTML = emailHTML + "<h2 style='font-family: Arial, Helvetica, sans-serif;'>Pending Contracts to be Completed</h2>"
	emailHTML = emailHTML + `
	<table>
		<thead>
			<th>ID</th>
			<th>Days</th>
			<th>Credit Officer</th>
			<th>Recovery Officer</th>
			<th>Customer Name</th>
			<th>Price</th>
		</thead>
		<tbody>
	`

	for rows.Next() {
		var id, creditOfficer, recoveryOfficer, customerName string
		var fileIncompleteFor, price int

		err = rows.Scan(&id, &fileIncompleteFor, &creditOfficer, &recoveryOfficer, &customerName, &price)
		if err != nil {
			fmt.Println(err)
			return
		}

		emailHTML = emailHTML + fmt.Sprintf(`
			<tr>
				<td>%s</td>
				%s%d</td>
				<td>%s</td>
				<td>%s</td>
				<td>%s</td>
				<td>%d</td>
			</tr>
		`, id, getTrColorHTMLforDates(fileIncompleteFor), fileIncompleteFor, creditOfficer, recoveryOfficer, customerName, price)
	}

	emailHTML = emailHTML + "</tbody></table></body></html>"

	toList := []string{"shamal@randeepa.com", "dimuthu@randeepa.com", "samanthi.chandrika@randeepa.com", "lehan.randesh@randeepa.com", "chandika.prasad@randeepa.com", "kularathna@agrivest.lk", "minura.maduwantha@agrivest.lk", "tharushika.samarathunga@agrivest.lk", "dumeshika.aluvihare@agrivest.lk"}
	err = sendEmail(toList, from, password, "Pending Contracts to be Completed", emailHTML)

	if err != nil {
		fmt.Printf("Failed to send email %+v\n", err)
		return
	}
}

func getTrColorHTMLforDates(days int) string {
	if days <= 7 {
		return "<td style='background-color: #199c31; color: #FFFFFF;'>"
	} else if days <= 14 {
		return "<td style='background-color: #5e5717; color: #FFFFFF;'>"
	} else {
		return "<td style='background-color: #f51111; color: #FFFFFF;'>"
	}
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
		fmt.Println("Failed to open log file")
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

	toList := []string{"shamal@randeepa.com", "dimuthu@randeepa.com", "samanthi.chandrika@randeepa.com", "lehan.randesh@randeepa.com", "chandika.prasad@randeepa.com", "kularathna@agrivest.lk", "minura.maduwantha@agrivest.lk", "tharushika.samarathunga@agrivest.lk", "dumeshika.aluvihare@agrivest.lk"}
	err = sendEmail(toList, from, password, "Day End Run Summary "+today, emailHTML)

	if err != nil {
		dayEndLog.Printf("Failed to send email %+v\n", err)
		return
	}
	dayEndLog.Printf("Summary email sent.")
}

func calculateDefault(dsn, logPath string) {
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

	type ContractArrears struct {
		ContractID int
		DueAmount  float64
	}

	today := time.Now().Format("2006-01-02")

	defaultAdderLogFile, err := openLogFile(logPath + today + "_default_adder.log")
	if err != nil {
		fmt.Println("Failed to open log file")
		os.Exit(1)
	}
	defaultAdderLog := log.New(defaultAdderLogFile, "", log.Ldate|log.Ltime)

	defaultAdderLog.Println("Starting default adder program")

	// load contracts from default_view
	var arrearsContracts []ContractArrears
	err = mysequel.QueryToStructs(&arrearsContracts, tx, "SELECT contract_id, due_amount FROM view_contract_arrears_for_default")
	if err != nil {
		defaultAdderLog.Printf("Failed to load arrears list %+v\n", err)
		return
	}

	var defaultForDay map[int]float64
	defaultForDay = make(map[int]float64)

	const DEFAULT_RATE_PER_DAY = 0.0986

	for _, contract := range arrearsContracts {
		defaultValue := (contract.DueAmount * DEFAULT_RATE_PER_DAY) / 100
		defaultForDay[contract.ContractID] = defaultValue
		defaultAdderLog.Printf("Contract: %d\tDefault Amount: %f\tDefault Per Day: %f", contract.ContractID,
			contract.DueAmount, defaultValue)

		var defaultEntryPresent int32
		err = tx.QueryRow("SELECT COUNT(*) AS entry_present FROM contract_default WHERE contract_id = ?", contract.ContractID).Scan(&defaultEntryPresent)

		if defaultEntryPresent == 1 {
			_, err = tx.Exec("UPDATE contract_default SET amount = amount + ? WHERE contract_id = ?", math.Round((defaultValue)*100)/100, contract.ContractID)
			if err != nil {
				defaultAdderLog.Printf("Failed to add default entry for %d\n", contract.ContractID)
			} else {
				defaultAdderLog.Printf("Default value successfully incremented for %d\n", contract.ContractID)
			}
		} else {
			_, err = mysequel.Insert(mysequel.Table{
				TableName: "contract_default",
				Columns:   []string{"contract_id", "amount"},
				Vals:      []interface{}{contract.ContractID, math.Round((defaultValue)*100) / 100},
				Tx:        tx,
			})

			if err != nil {
				defaultAdderLog.Printf("Failed to add first time default entry for %d\n", contract.ContractID)
			} else {
				defaultAdderLog.Printf("Default value successfully inserted for %d\n", contract.ContractID)
			}
		}
	}

	defaultAdderLog.Printf("Default adder program complete")
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
	fromAddress := mail.Address{Name: "Agrivest Mailer", Address: from}

	headers := make(map[string]string)
	headers["From"] = fromAddress.String()
	headers["To"] = strings.Join(to, ",")
	headers["MIME-version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=\"UTF-8\""
	headers["Subject"] = subject

	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	servername := "smtp.zoho.com:465"
	host, _, _ := net.SplitHostPort(servername)
	auth := smtp.PlainAuth("", from, password, host)

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         host,
	}

	conn, err := tls.Dial("tcp", servername, tlsConfig)
	if err != nil {
		return err
	}

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}

	if err = c.Auth(auth); err != nil {
		return err
	}

	if err = c.Mail(from); err != nil {
		return err
	}

	for _, toAddress := range to {
		if err = c.Rcpt(toAddress); err != nil {
			return err
		}
	}

	w, err := c.Data()
	if err != nil {
		return err
	}

	_, err = w.Write([]byte(message))
	if err != nil {
		return err
	}

	err = w.Close()
	if err != nil {
		return err
	}

	c.Quit()

	return nil
}

func openLogFile(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
}
