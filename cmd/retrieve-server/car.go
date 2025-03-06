package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/urfave/cli/v2"
)

var carCmd = &cli.Command{
	Name:        "car",
	Usage:       "car <carInfo file path>",
	UsageText:   "retrieve-server car <carInfo file path>",
	Description: "fetch block from qiniu to retrieve server",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "server-addr",
			Value: "127.0.0.1:9876",
		},
		&cli.IntFlag{
			Name:  "parallel",
			Value: 10,
		},
	},
	Action: func(cctx *cli.Context) error {
		if cctx.Args().Len() < 1 {
			return fmt.Errorf("path is required")
		}

		path := cctx.Args().Get(0)
		if path == "" {
			return fmt.Errorf("path is required")
		}

		url := fmt.Sprintf("http://%s/car?path=%s&parallel=%d", cctx.String("server-addr"), url.QueryEscape(path), cctx.Int("parallel"))
		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			r, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			return fmt.Errorf("status: %s msg: %s", resp.Status, string(r))
		}

		return nil
	},
}
