package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/codegangsta/cli"
	"github.com/sebest/hooky/models"
	"github.com/sebest/hooky/restapi"
	"github.com/sebest/hooky/scheduler"
	"github.com/sebest/hooky/store"
	"github.com/stretchr/graceful"
)

func main() {
	app := cli.NewApp()
	app.Name = "hooky"
	app.Usage = "the webhooks scheduler"
	app.Version = "0.1"
	app.Author = "Sébastien Estienne"
	app.Email = "sebastien.estienne@gmail.com"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "bind-address",
			Value:  "",
			Usage:  "host address to bind on",
			EnvVar: "HOOKY_BIND_ADDRESS",
		},
		cli.StringFlag{
			Name:   "bind-port",
			Value:  "8000",
			Usage:  "port number to bind on",
			EnvVar: "HOOKY_BIND_PORT,PORT",
		},
		cli.StringFlag{
			Name:   "mongo-uri",
			Value:  "mongodb://127.0.0.1/hooky",
			Usage:  "MongoDB URI to connect to",
			EnvVar: "HOOKY_MONGO_URI",
		},
		cli.StringFlag{
			Name:   "admin-password",
			Value:  "admin",
			Usage:  "admin password",
			EnvVar: "HOOKY_ADMIN_PASSWORD",
		},
		cli.StringFlag{
			Name:   "accesslog-format",
			Value:  "none",
			Usage:  "format of the access log: json, apache-fancy, apache-combined, apache-common or none",
			EnvVar: "HOOKY_ACCESSLOG_FORMAT",
		},
		cli.IntFlag{
			Name:   "max-mongo-query",
			Value:  1,
			Usage:  "maximum number of parallel queries on MongoDB",
			EnvVar: "HOOKY_MAX_MONGO_QUERY",
		},
		cli.IntFlag{
			Name:   "max-http-request",
			Value:  20,
			Usage:  "maximum number of parallel HTTP requests",
			EnvVar: "HOOKY_MAX_HTTP_REQUEST",
		},
		cli.IntFlag{
			Name:   "touch-interval",
			Value:  5,
			Usage:  "frequency to update the tasks reservation duration in seconds",
			EnvVar: "HOOKY_TOUCH_INTERVAL",
		},
		cli.IntFlag{
			Name:   "clean-finished-attempts",
			Value:  7 * 24,
			Usage:  "delete finished attempts that are older than this age in hours",
			EnvVar: "HOOKY_CLEAN_FINISHED_ATTEMPTS",
		},
	}
	app.Action = func(c *cli.Context) {
		s, err := store.New(c.String("mongo-uri"))
		if err != nil {
			log.Fatal(err)
		}

		db := s.DB()
		if err := models.NewBase(db).Bootstrap(); err != nil {
			log.Fatal(err)
		}
		db.Session.Close()

		sched := scheduler.New(s, c.Int("max-mongo-query"), c.Int("max-http-request"), c.Int("touch-interval"), c.Int("clean-finished-attempts")*3600)
		sched.Start()
		ra, err := restapi.New(s, c.String("admin-password"), c.String("accesslog-format"))
		if err != nil {
			log.Fatal(err)
		}
		server := &graceful.Server{
			Timeout: 10 * time.Second,
			Server: &http.Server{
				Addr:    c.String("bind-host") + ":" + c.String("bind-port"),
				Handler: ra.MakeHandler(),
			},
		}
		err = server.ListenAndServe()
		if err != nil {
			log.Println(err)
		}
		log.Println("exiting...")
		sched.Stop()
		log.Println("exited")
	}
	app.Run(os.Args)
}
