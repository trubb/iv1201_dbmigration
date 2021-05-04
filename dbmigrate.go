package main

import (
	"database/sql"
	"fmt"
	"log"
	_ "mysql"
	"os"
	"strings"
)

// target files
const MIGRATIONFILE = "database_dump.sql"
const MAILFILE = "users_to_email.txt"
const ADMINFILE = "admins_to_email.txt"

// custom placeholders to replace NULL values
const PLACEHOLDERTEXT = "thisIsAStringOfWordsThatIsWrittenInPlaceOfProperDataInCaseAPasswordOrEmailOrSomethingElseIsMissingInTheOriginalDatabase"
const PLACEHOLDERINT = 133747116969666

func main() {
	// Read environment variables
	DBCREDENTIALS := os.Getenv("IV1201_DBCREDENTIALS")

	// create a db connection pool using the provided environment variables
	db, err := sql.Open("mysql", DBCREDENTIALS)
	// local test db, not reachable from internet
	//db, err := sql.Open("mysql", "iv1201:leif@tcp(127.0.0.1:3306)/iv1201mysql")

	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		fmt.Println("DB not accessible!")
		log.Fatal(err)
	}
	createFiles()
	readTableContent(db)
}

func createFiles() {
	err := os.Remove(MIGRATIONFILE)
	if err != nil {
		log.Println("No previous dump file found to overwrite")
	} else {
		log.Println("Deleted previous dump file")
	}

	_, err = os.Create(MIGRATIONFILE)
	if err != nil {
		log.Fatal(err)
	}

	err = os.Remove(MAILFILE)
	if err != nil {
		log.Println("No previous mail file found to overwrite")
	} else {
		log.Println("Deleted previous mail file")
	}

	_, err = os.Create(MAILFILE)
	if err != nil {
		log.Fatal(err)
	}

	err = os.Remove(ADMINFILE)
	if err != nil {
		log.Println("No previous admin mail file found to overwrite")
	} else {
		log.Println("Deleted previous admin mail file")
	}

	_, err = os.Create(ADMINFILE)
	if err != nil {
		log.Fatal(err)
	}
}

// Writes the provided input to a file
// Input:
// 	target:	a file to write to
//	input:	a string that shall be written to the file
func writeToFile(target string, input string) error {
	file, err := os.OpenFile(target, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	if _, err := file.WriteString(input + "\n"); err != nil {
		log.Fatal(err)
	}
	if err := file.Close(); err != nil {
		log.Fatal(err)
	}
	return nil
}

// Writes the provided input to a file and logs a reason for doing so.
// Input:
// 	email:		the user's email address
//	personID:	the user's ID
//	reason:		what mandatory data was missing from the user's profile
//	target:		the file that the input will be appended to
func writeToEmailFile(email string, personID int64, reason string, target string) {
	log.Printf("User %d lacks a valid ** " + strings.ToUpper(reason) + " **\n    User was NOT migrated! \n    User was appended to list of users to email!", personID)
	err := writeToFile(target, email + ", reason: " + reason)
	if err != nil {
		log.Fatal(err)
	}
}

// Call the functions that read each table one by one
// Input: a database handle
func readTableContent(db *sql.DB) {
	err := migrateRole(db)
	if err != nil {
		log.Fatal(err)
	}
	err = migrateCompetence(db)
	if err != nil {
		log.Fatal(err)
	}
	err = migratePerson(db)
	if err != nil {
		log.Fatal(err)
	}
}

// Read data from the table "role" and export it to the dumpfile
// Input: a database handle
func migrateRole(db *sql.DB) error {
	rows, err := db.Query("SELECT role_id, `name` from role")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			roleId sql.NullInt64
			name   sql.NullString
		)
		err := rows.Scan(&roleId, &name)
		if err != nil {
			log.Fatal(err)
		}

		if roleId.Valid == false {
			roleId.Int64 = PLACEHOLDERINT
		}
		if name.Valid == false {
			name.String = PLACEHOLDERTEXT
		}

		log.Println(roleId.Int64, name.String)

		row := fmt.Sprintf("INSERT INTO role (role_id, name) VALUES (%d, '%s');", roleId.Int64, name.String)
		writeToFile(MIGRATIONFILE, row)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	return nil
}

// Read data from the table "person" and export it to the dumpfile
// Users with missing data is handled differently depending on if they are
// administrators or users.
// Input: a database handle
func migratePerson(db *sql.DB) error {
	rows, err := db.Query("SELECT `person_id`, `name`, `surname`, ssn, `email`, `password`, role_id, `username` from person")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			personId, roleId                              sql.NullInt64
			name, surname, ssn, email, password, username sql.NullString
		)
		err := rows.Scan(&personId, &name, &surname, &ssn, &email, &password, &roleId, &username)
		if err != nil {
			log.Fatal(err)
		}

		// User with complete data should just go through without any issue.
		// And then we can fire off the other migrations too, but only checking for this specific user
		if personId.Valid && name.Valid && surname.Valid && ssn.Valid && email.Valid && password.Valid && roleId.Valid && username.Valid {
			log.Println(personId.Int64, name.String, surname.String, ssn.String, email.String, password.String, roleId.Int64, username.String)

			row := fmt.Sprintf("INSERT INTO person (person_id, name, surname, ssn, email, password, role, username) VALUES (%d, '%s', '%s', '%s', '%s', '%s', %d, '%s');", personId.Int64, name.String, surname.String, ssn.String, email.String, password.String, roleId.Int64, username.String)
			err = writeToFile(MIGRATIONFILE, row)

			// Check for the user in the other tables
			err = migrateAvailability(db, personId.Int64)
			if err != nil {
				log.Fatal(err)
			}
			err = migrateCompetenceProfile(db, personId.Int64)
			if err != nil {
				log.Fatal(err)
			}
			err = checkApplications(db, personId.Int64)
			if err != nil {
				log.Fatal(err)
			}
		}

		// Below, check faulty users for solutions to their issues

		// User type: any
		// Primary key missing || RoleID missing
		// Migrate: no
		if personId.Valid == false || roleId.Valid == false {
			continue
		}
		// User type: admin
		// Mail: none
		// Migrate: No
		// Send mail: yes
		if email.Valid == false && roleId.Int64 == 1 {
			writeToEmailFile(username.String, personId.Int64, "email", ADMINFILE)
			continue
		}
		// User type: applicant
		// Mail: none
		// Migrate: No
		if email.Valid == false && roleId.Int64 == 2 {
			continue
		}
		// User type: admin
		// Mail: yes
		// Obligatory data: missing
		// Migrate: no
		// Send mail: yes
		if email.Valid == true && roleId.Int64 == 1 {
			if password.Valid == false {
				writeToEmailFile(email.String, personId.Int64, "password", ADMINFILE)
				continue
			}
			if username.Valid == false {
				writeToEmailFile(email.String, personId.Int64, "username", ADMINFILE)
				continue
			}
			if name.Valid == false {
				writeToEmailFile(email.String, personId.Int64, "name", ADMINFILE)
				continue
			}
			if surname.Valid == false {
				writeToEmailFile(email.String, personId.Int64, "surname", ADMINFILE)
				continue
			}
			if ssn.Valid == false {
				writeToEmailFile(email.String, personId.Int64, "ssn", ADMINFILE)
				continue
			}
		}
		// User type: applicant
		// Mail: yes
		// Obligatory data: missing
		// Migrate: no
		// Send mail: yes
		if email.Valid == true && roleId.Int64 == 2 {
			if password.Valid == false {
				writeToEmailFile(email.String, personId.Int64, "password", MAILFILE)
				continue
			}
			if username.Valid == false {
				writeToEmailFile(email.String, personId.Int64, "username", MAILFILE)
				continue
			}
			if name.Valid == false {
				writeToEmailFile(email.String, personId.Int64, "name", MAILFILE)
				continue
			}
			if surname.Valid == false {
				writeToEmailFile(email.String, personId.Int64, "surname", MAILFILE)
				continue
			}
			if ssn.Valid == false {
				writeToEmailFile(email.String, personId.Int64, "ssn", MAILFILE)
				continue
			}
		}
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	return nil
}

// Read data from the table "availability" and export it to the dumpfile
// Input:
//	db:				a database handle
//	inputPersonId:	personID of the sought after records
func migrateAvailability(db *sql.DB, inputPersonId int64) error {
	rows, err := db.Query("SELECT availability_id, person_id, `from_date`, `to_date` from availability where person_id=?", inputPersonId)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			availabilityId, personId sql.NullInt64
			fromDate, toDate         sql.NullString
		)
		err := rows.Scan(&availabilityId, &personId, &fromDate, &toDate)
		if err != nil {
			log.Fatal(err)
		}

		if availabilityId.Valid == false || personId.Valid == false || fromDate.Valid == false || toDate.Valid == false {
			break
		}

		log.Println(availabilityId.Int64, personId.Int64, fromDate.String, toDate.String)

		row := fmt.Sprintf("INSERT INTO availability (availability_id, person, from_date, to_date) VALUES (%d, %d, '%s', '%s');", availabilityId.Int64, personId.Int64, fromDate.String, toDate.String)
		err = writeToFile(MIGRATIONFILE, row)
		if err != nil {
			log.Fatal(err)
		}
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	return nil
}

// Read data from the table "competence" and export it to the dumpfile
// Input: a database handle
func migrateCompetence(db *sql.DB) error {
	rows, err := db.Query("SELECT competence_id, `name` from competence")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			competenceId sql.NullInt64
			name         sql.NullString
		)
		err := rows.Scan(&competenceId, &name)
		if err != nil {
			log.Fatal(err)
		}

		if competenceId.Valid == false {
			competenceId.Int64 = PLACEHOLDERINT
		}
		if name.Valid == false {
			name.String = PLACEHOLDERTEXT
		}

		log.Println(competenceId.Int64, name.String)

		row := fmt.Sprintf("INSERT INTO competence (competence_id, name) VALUES (%d, '%s');", competenceId.Int64, name.String)

		err = writeToFile(MIGRATIONFILE, row)
		if err != nil {
			log.Fatal(err)
		}
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	return nil
}

// Read data from the table "competence_profile" and export it to the dumpfile
// Input:
//	db:				a database handle
//	inputPersonId:	personID of the sought after records
func migrateCompetenceProfile(db *sql.DB, inputPersonId int64) error {
	rows, err := db.Query("SELECT competence_profile_id, person_id, competence_id, years_of_experience from competence_profile where person_id=?", inputPersonId)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			competenceProfileId, personId, competenceId sql.NullInt64
			yearsOfExperience                           sql.NullFloat64
		)
		err := rows.Scan(&competenceProfileId, &personId, &competenceId, &yearsOfExperience)
		if err != nil {
			log.Fatal(err)
		}

		if competenceProfileId.Valid == false || personId.Valid == false || competenceId.Valid == false || yearsOfExperience.Valid == false {
			break
		}

		log.Println(competenceProfileId.Int64, personId.Int64, competenceId.Int64, yearsOfExperience.Float64)

		row := fmt.Sprintf("INSERT INTO competence_profile (competence_profile_id, person, competence, years_of_experience) VALUES (%d, %d, %d, %.1f);", competenceId.Int64, personId.Int64, competenceId.Int64, yearsOfExperience.Float64)

		err = writeToFile(MIGRATIONFILE, row)
		if err != nil {
			log.Fatal(err)
		}
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	return nil
}

// Read data from the table "person", and for each person in the table check if they have entries in
// the table "availability" in order to determine if they have an active application, if yes export it to the dumpfile
// Input:
//	db:				a database handle
//	inputPersonId:	personID of the sought after records
func checkApplications(db *sql.DB, inputPersonId int64) error {
	applicant, err := db.Query("SELECT person_id from person where person_id=?", inputPersonId)
	if err != nil {
		log.Fatal(err)
	}
	defer applicant.Close()

	for applicant.Next() {
		var (
			applicantId sql.NullInt64
		)
		err := applicant.Scan(&applicantId)
		if err != nil {
			log.Fatal(err)
		}

		availability, err := db.Query("SELECT person_id from availability where person_id=?", inputPersonId)
		if err != nil {
			log.Fatal(err)
		}
		defer availability.Close()

		for availability.Next() {
			var (
				availpersonId sql.NullInt64
			)
			err := availability.Scan(&availpersonId)
			if err != nil {
				log.Fatal(err)
			}

			log.Println(applicantId.Int64, availpersonId.Int64)

			if applicantId.Int64 == availpersonId.Int64 {
				output := fmt.Sprintf("INSERT INTO application (version, person, status) VALUES (1, %d, 'unhandled');", availpersonId.Int64)

				err = writeToFile(MIGRATIONFILE, output)
				if err != nil {
					log.Fatal(err)
				}
			}
			break
		}
		if err := availability.Err(); err != nil {
			log.Fatal(err)
		}
	}
	if err := applicant.Err(); err != nil {
		log.Fatal(err)
	}
	return nil
}
