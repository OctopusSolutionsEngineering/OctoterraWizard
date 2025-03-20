package sensitivevariables

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/mcasperson/OctoterraWizard/internal/naming"
	"log"
	"strconv"
	"strings"
	"time"
)
import _ "github.com/microsoft/go-mssqldb"

func GetDatabaseConnection(server string, port string, database string, username string, password string, ctx context.Context) (*sql.DB, error) {
	portNum, err := strconv.Atoi(port)
	if err != nil {
		return nil, err
	}

	dsn := "sqlserver://" + username + ":" + password + "@" + server + ":" + fmt.Sprint(portNum) + "?database=" + database
	db, err := sql.Open("sqlserver", dsn)
	if err != nil {
		return nil, err
	}

	db.SetConnMaxLifetime(0)
	db.SetMaxIdleConns(3)
	db.SetMaxOpenConns(3)

	return db, nil
}

// ExtractVariables extracts sensitive variables from the database and returns them as terraform variable values
func ExtractVariables(server string, port string, database string, username string, password string, masterKey string) (string, error) {
	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	db, err := GetDatabaseConnection(server, port, database, username, password, ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Println(err.Error())
		}
	}()

	// Test the connection
	if err := PingDatabase(ctx, db); err != nil {
		return "", err
	}

	// Get the sensitive variables
	sensitiveVars, err := getVariableSetSecrets(ctx, db, masterKey)

	return sensitiveVars, err
}

func PingDatabase(ctx context.Context, db *sql.DB) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return db.PingContext(ctx)
}

func getVariableSetSecrets(ctx context.Context, db *sql.DB, masterKey string) (string, error) {
	var id string
	var jsonValue string
	var isFrozen bool

	timeout, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	rows, err := db.QueryContext(timeout, "SELECT Id, JSON, IsFrozen FROM VariableSet")
	if err != nil {
		return "", err
	}

	defer func() {
		if err := rows.Close(); err != nil {
			log.Println(err.Error())
		}
	}()

	var builder strings.Builder

	for rows.Next() {
		if err = rows.Scan(&id, &jsonValue, &isFrozen); err != nil {
			return "", err
		}

		if isFrozen {
			continue
		}

		var result map[string]interface{}

		if err := json.Unmarshal([]byte(jsonValue), &result); err != nil {
			return "", err
		}

		if variables, ok := result["Variables"].([]interface{}); ok {
			for _, variable := range variables {

				if variableMap, ok := variable.(map[string]interface{}); ok {

					// Don't include the library variable set where we save the secrets
					if fmt.Sprint(variableMap["Name"]) == SecretsVariableName {
						continue
					}

					// only include sensitive variables
					if fmt.Sprint(variableMap["Type"]) != "Sensitive" {
						continue
					}

					variableName := naming.VariableSecretName(fmt.Sprint(variableMap["Id"]))
					variableValue, err := DecryptSensitiveVariable(masterKey, fmt.Sprint(variableMap["Value"]))

					if err != nil {
						return "", err
					}

					escapedValue, err := json.Marshal(variableValue)

					if err != nil {
						return "", err
					}

					builder.WriteString(variableName + " = " + string(escapedValue) + "\n")

				}
			}
		}

	}

	return builder.String(), nil
}
