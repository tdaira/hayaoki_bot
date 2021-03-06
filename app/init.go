package app

import (
	"time"

	"github.com/tdaira/hayaoki_bot/handler"
)

const location = "Asia/Tokyo"
const channelID = "C0FN5ULTD"

func init() {
	// Init timezone
	loc, err := time.LoadLocation(location)
	if err != nil {
		loc = time.FixedZone(location, 9*60*60)
	}
	time.Local = loc

	slashHandler := handler.NewSlashHandler(channelID)
	slashHandler.Run()

	cronHandler := handler.NewCronHandler(channelID)
	cronHandler.Run()
}
