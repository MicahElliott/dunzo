package dun

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"github.com/go-co-op/gocron/v2"
)

func Schedule(a fyne.App) gocron.Scheduler {
	fmt.Println("Starting scheduler")

	a.SendNotification(fyne.NewNotification("CRON", "Starting scheduler" ))

	// create a scheduler
	s, err := gocron.NewScheduler()
	if err != nil { }

	// add a job to the scheduler
	j, err := s.NewJob(
		// gocron.DurationJob(3*time.Second),
		gocron.DurationJob(30*time.Minute),
		// gocron.DurationJob(10*time.Second),
		gocron.NewTask(
			func(x string, b int) {
				fmt.Println("doing something")
				a.SendNotification(fyne.NewNotification(
					"Notificaton from inside cron!!",
					"It's about that time" )) },
			"hello", 1 ))
	if err != nil { }
	// each job has a unique id
	fmt.Println(j.ID())

	sod, err := s.NewJob(
		gocron.DailyJob(1,
			gocron.NewAtTimes(gocron.NewAtTime(7, 25, 0)) ),
		gocron.NewTask(func() {
			notify(a,
				"Start your day",
				"What will you do today?") }) )
	fmt.Println(sod.Name())

	eod, err := s.NewJob(
		gocron.DailyJob( 1,
			gocron.NewAtTimes(gocron.NewAtTime(15, 45, 0)) ),
		gocron.NewTask(func() {
			notify(a,
				"End your day",
				"What did you accomplish today?") }))
	fmt.Println(eod.Name())
	if err != nil { }

	s.Start() // start the scheduler

	select { // block until you are ready to shut down
	case <-time.After(4 * time.Second):
		fmt.Println("a minute has passed")
	}

	return s

	// err = s.Shutdown() // when you're done, shut it down
	// if err != nil {
	// 	// handle error
	// }
}

func offHours() bool {
	now := time.Now()
	fmt.Println(now)
	return false
}

func notify(a fyne.App, headline, msg string) {
	if offHours() { fmt.Println("no-op since off hours")
	} else {
		fmt.Println(headline)
		a.SendNotification(fyne.NewNotification(headline, msg)) }
}
