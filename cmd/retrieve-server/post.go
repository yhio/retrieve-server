package main

import (
	"fmt"

	"github.com/gh-efforts/retrieve-server/client"
	"github.com/ipfs/go-cid"
	car "github.com/ipld/go-car/v2"
	"github.com/ipld/go-car/v2/blockstore"
	"github.com/urfave/cli/v2"
)

// post root block from car file to retrieve server
var postCmd = &cli.Command{
	Name:  "post",
	Usage: "<file.car> <block cid>",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "server-addr",
			Value: "127.0.0.1:9876",
		},
	},
	Action: func(cctx *cli.Context) error {
		if cctx.Args().Len() < 2 {
			return fmt.Errorf("args < 2")
		}

		bs, err := blockstore.OpenReadOnly(cctx.Args().Get(0), car.ZeroLengthSectionAsEOF(true))
		if err != nil {
			return err
		}

		cid, err := cid.Parse(cctx.Args().Get(1))
		if err != nil {
			return err
		}

		block, err := bs.Get(cctx.Context, cid)
		if err != nil {
			return err
		}

		return client.PostRootBlock(cctx.String("server-addr"), cid.String(), block.RawData())
	},
}
