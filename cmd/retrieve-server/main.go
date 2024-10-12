package main

import (
	"net/http"
	"os"
	"time"

	_ "net/http/pprof"

	"contrib.go.opencensus.io/exporter/prometheus"
	cliutil "github.com/filecoin-project/lotus/cli/util"
	"github.com/mitchellh/go-homedir"
	"github.com/yhio/retrieve-server/build"
	"github.com/yhio/retrieve-server/db"
	"github.com/yhio/retrieve-server/metrics"
	"github.com/yhio/retrieve-server/server"

	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var log = logging.Logger("main")

func main() {
	local := []*cli.Command{
		runCmd,
		postCmd,
		pprofCmd,
	}

	app := &cli.App{
		Name:     "retrieve-server",
		Usage:    "retrieve server ",
		Version:  build.UserVersion(),
		Commands: local,
	}

	if err := app.Run(os.Args); err != nil {
		log.Errorf("%+v", err)
	}
}

var runCmd = &cli.Command{
	Name: "run",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "listen",
			Value: "0.0.0.0:9876",
		},
		&cli.BoolFlag{
			Name:  "debug",
			Value: false,
		},
		&cli.StringFlag{
			Name:  "db",
			Value: "./rserver.db",
		},
	},
	Action: func(cctx *cli.Context) error {
		setLog(cctx.Bool("debug"))

		log.Info("starting retrieve server ...")

		ctx := cliutil.ReqContext(cctx)

		exporter, err := prometheus.NewExporter(prometheus.Options{
			Namespace: "rserver",
		})
		if err != nil {
			return err
		}

		ctx, _ = tag.New(ctx,
			tag.Insert(metrics.Version, build.BuildVersion),
			tag.Insert(metrics.Commit, build.CurrentCommit),
		)
		if err := view.Register(
			metrics.Views...,
		); err != nil {
			return err
		}
		stats.Record(ctx, metrics.Info.M(1))

		listen := cctx.String("listen")
		log.Infow("retrieve server", "listen", listen)

		http.Handle("/metrics", exporter)

		path, err := homedir.Expand(cctx.String("db"))
		if err != nil {
			return err
		}
		log.Infof("db path: %s", path)

		db, err := db.OpenDB(path)
		if err != nil {
			return err
		}
		server.New(db).Handle()

		server := &http.Server{
			Addr: listen,
		}

		go func() {
			<-ctx.Done()
			time.Sleep(time.Millisecond * 100)
			log.Info("closed retrieve server")
			server.Shutdown(ctx)
		}()

		return server.ListenAndServe()
	},
}

func setLog(debug bool) {
	level := "INFO"
	if debug {
		level = "DEBUG"
	}

	logging.SetLogLevel("main", level)
	logging.SetLogLevel("db", level)
	logging.SetLogLevel("metrics", level)
	logging.SetLogLevel("server", level)
	logging.SetLogLevel("middleware", level)
	logging.SetLogLevel("client", level)
}
