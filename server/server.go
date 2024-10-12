package server

import (
	"database/sql"

	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("server")

type Server struct {
	db *sql.DB
}

func New(db *sql.DB) *Server {
	return &Server{
		db: db,
	}
}

func (s *Server) upsert(rb *RootBlock) error {
	_, err := s.db.Exec(`INSERT OR IGNORE INTO RootBlocks(root, size, block) VALUES ($1, $2, $3)`, rb.Root, len(rb.Block), rb.Block)
	if err != nil {
		return err
	}

	log.Debugw("upsertr", "root", rb.Root, "size", len(rb.Block))
	return nil
}

func (s *Server) delete(root string) error {
	_, err := s.db.Exec(`DELETE FROM RootBlocks WHERE root=$1`, root)
	if err != nil {
		return err
	}

	log.Debugw("delete", "root", root)
	return nil
}

func (s *Server) block(root string) ([]byte, error) {
	var block []byte
	err := s.db.QueryRow(`SELECT block FROM RootBlocks WHERE root=$1`, root).Scan(&block)
	if err != nil {
		return nil, err
	}

	log.Debugw("getblock", "root", root, "size", len(block))
	return block, nil
}

func (s *Server) size(root string) (int, error) {
	var size int
	err := s.db.QueryRow(`SELECT size FROM RootBlocks WHERE root=$1`, root).Scan(&size)
	if err != nil {
		return 0, err
	}

	log.Debugw("getsize", "root", root, "size", size)
	return size, nil
}
