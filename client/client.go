package client

import (
	"context"
	"errors"

	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("client")

var ErrNotFound = errors.New("block not found")

type Client struct {
	addr string
}

func New(addr string) *Client {
	return &Client{
		addr: addr,
	}
}

func (c *Client) BlockstoreGet(ctx context.Context, cid cid.Cid) ([]byte, error) {
	rb, err := GetBlock(c.addr, cid.String())
	if err != nil {
		log.Error(err)
		return nil, ErrNotFound
	}

	return rb.Block, nil
}

func (c *Client) BlockstoreGetSize(ctx context.Context, cid cid.Cid) (int, error) {
	rz, err := GetSize(c.addr, cid.String())
	if err != nil {
		log.Error(err)
		return 0, ErrNotFound
	}

	return rz.Size, nil
}

func (c *Client) BlockstoreHas(ctx context.Context, cid cid.Cid) (bool, error) {
	return GetHas(c.addr, cid.String()), nil
}

func (c *Client) Get(ctx context.Context, cid cid.Cid) (b blocks.Block, err error) {
	data, err := c.BlockstoreGet(ctx, cid)
	if err != nil {
		return nil, err
	}

	return blocks.NewBlockWithCid(data, cid)
}

func (c *Client) Has(ctx context.Context, cid cid.Cid) (bool, error) {
	return c.BlockstoreHas(ctx, cid)
}

func (c *Client) GetSize(ctx context.Context, cid cid.Cid) (int, error) {
	return c.BlockstoreGetSize(ctx, cid)
}

// --- UNSUPPORTED BLOCKSTORE METHODS -------
func (c *Client) DeleteBlock(context.Context, cid.Cid) error {
	return errors.New("unsupported operation DeleteBlock")
}
func (c *Client) HashOnRead(_ bool) {}
func (c *Client) Put(context.Context, blocks.Block) error {
	return errors.New("unsupported operation Put")
}
func (c *Client) PutMany(context.Context, []blocks.Block) error {
	return errors.New("unsupported operation PutMany")
}
func (c *Client) AllKeysChan(ctx context.Context) (<-chan cid.Cid, error) {
	return nil, errors.New("unsupported operation AllKeysChan")
}
