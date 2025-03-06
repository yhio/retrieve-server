package server

import (
	"encoding/binary"

	"github.com/dgraph-io/badger/v3"
	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("server")

type Server struct {
	db *badger.DB
	mc *MinioConfig
}

func New(db *badger.DB) *Server {
	mc, err := getMinioConfig()
	if err != nil {
		log.Fatal("get minio config error: %v", err)
	}

	return &Server{
		db: db,
		mc: mc,
	}
}

func (s *Server) upsert(rb *RootBlock) error {
	// 将 size 转换为字节数组
	sizeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(sizeBytes, uint64(len(rb.Block)))

	// 创建事务
	txn := s.db.NewTransaction(true)
	defer txn.Discard()

	// 存储 block 数据
	blockKey := []byte("block:" + rb.Root)
	if err := txn.Set(blockKey, rb.Block); err != nil {
		return err
	}

	// 存储 size 数据
	sizeKey := []byte("size:" + rb.Root)
	if err := txn.Set(sizeKey, sizeBytes); err != nil {
		return err
	}

	// 提交事务
	if err := txn.Commit(); err != nil {
		return err
	}

	log.Debugw("upsert", "root", rb.Root, "size", len(rb.Block))
	return nil
}

func (s *Server) delete(root string) error {
	// 创建事务
	txn := s.db.NewTransaction(true)
	defer txn.Discard()

	// 删除 block 数据
	blockKey := []byte("block:" + root)
	if err := txn.Delete(blockKey); err != nil {
		return err
	}

	// 删除 size 数据
	sizeKey := []byte("size:" + root)
	if err := txn.Delete(sizeKey); err != nil {
		return err
	}

	// 提交事务
	if err := txn.Commit(); err != nil {
		return err
	}

	log.Debugw("delete", "root", root)
	return nil
}

func (s *Server) block(root string) ([]byte, error) {
	// 创建只读事务
	txn := s.db.NewTransaction(false)
	defer txn.Discard()

	// 获取 block 数据
	blockKey := []byte("block:" + root)
	item, err := txn.Get(blockKey)
	if err != nil {
		return nil, err
	}

	block, err := item.ValueCopy(nil)
	if err != nil {
		return nil, err
	}

	log.Debugw("getblock", "root", root, "size", len(block))
	return block, nil
}

func (s *Server) size(root string) (int, error) {
	// 创建只读事务
	txn := s.db.NewTransaction(false)
	defer txn.Discard()

	// 获取 size 数据
	sizeKey := []byte("size:" + root)
	item, err := txn.Get(sizeKey)
	if err != nil {
		return 0, err
	}

	sizeBytes, err := item.ValueCopy(nil)
	if err != nil {
		return 0, err
	}

	size := int(binary.BigEndian.Uint64(sizeBytes))
	log.Debugw("getsize", "root", root, "size", size)
	return size, nil
}
