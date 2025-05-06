package validators

import (
	"context"
	"github.com/mcasperson/OctoterraWizard/internal/sensitivevariables"
	"github.com/mcasperson/OctoterraWizard/internal/state"
	"log"
	"strconv"
)

// ValidateDatabase pings the database to confirm the connection details are correct
func ValidateDatabase(state state.State) error {
	ctx, stop := context.WithCancel(context.Background())
	defer stop()

	// Port must be a number
	if _, err := strconv.Atoi(state.DatabasePort); err != nil {
		return err
	}

	db, err := sensitivevariables.GetDatabaseConnection(state.DatabaseServer, state.DatabasePort, state.DatabaseName, state.DatabaseUser, state.DatabasePass, ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Println(err.Error())
		}
	}()

	return sensitivevariables.PingDatabase(ctx, db)
}
