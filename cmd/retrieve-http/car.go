package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/filecoin-project/boost-graphsync/storeutil"
	bdclient "github.com/filecoin-project/boost/extern/boostd-data/client"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/ipfs/go-cid"
	"github.com/ipld/frisbii"
	carv2 "github.com/ipld/go-car/v2"
	"github.com/ipld/go-car/v2/blockstore"
	carindex "github.com/ipld/go-car/v2/index"
)

type CarInfo struct {
	DataCid   string `json:"dataCid"`
	PieceCid  string `json:"pieceCid"`
	PieceSize int64  `json:"pieceSize"`
	CarSize   int64  `json:"carSize"`
	FileName  string `json:"fileName"`
}

var (
	carInfoMap sync.Map
	indexCache *lru.Cache[string, carindex.IterableIndex]
)

func init() {
	// 初始化LRU缓存，设置容量为1000
	var err error
	indexCache, err = lru.New[string, carindex.IterableIndex](1000)
	if err != nil {
		panic(fmt.Sprintf("failed to create index cache: %v", err))
	}
}

func loadCarInfo(dirPath string) error {
	carInfoMap.Range(func(key, value interface{}) bool {
		carInfoMap.Delete(key)
		return true
	})

	files, err := os.ReadDir(dirPath)
	if err != nil {
		return err
	}

	for _, file := range files {
		if !file.IsDir() {
			filePath := filepath.Join(dirPath, file.Name())
			log.Infof("load car info from %s", filePath)

			f, err := os.Open(filePath)
			if err != nil {
				return err
			}
			defer f.Close()

			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				var carInfo CarInfo
				if err := json.Unmarshal(scanner.Bytes(), &carInfo); err != nil {
					return err
				}
				carInfoMap.Store(carInfo.PieceCid, carInfo)
			}

			if err := scanner.Err(); err != nil {
				return err
			}
		}
	}
	count := 0
	carInfoMap.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	log.Infof("load car info from %s, total %d", dirPath, count)
	return nil
}

func getCarInfo(ctx context.Context, blockCid cid.Cid, store *bdclient.Store) (CarInfo, error) {
	pieceCids, err := store.PiecesContainingMultihash(ctx, blockCid.Hash())
	if err != nil {
		return CarInfo{}, err
	}

	for _, pieceCid := range pieceCids {
		if value, ok := carInfoMap.Load(pieceCid.String()); ok {
			return value.(CarInfo), nil
		}
	}
	return CarInfo{}, fmt.Errorf("blockCid: %s pieceCids: %v cannot get car info", blockCid, pieceCids)
}

func car(w http.ResponseWriter, r *http.Request, store *bdclient.Store) {
	block := strings.TrimPrefix(r.URL.Path, "/ipfs/")
	blockCid, err := cid.Parse(block)
	if err != nil {
		log.Errorw("invalid cid", "cid", blockCid, "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	carInfo, err := getCarInfo(r.Context(), blockCid, store)
	if err != nil {
		log.Errorw("cannot get car info", "cid", blockCid, "error", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	log.Debugw("car handler", "blockCid", blockCid, "carInfo.DataCid", carInfo.DataCid, "fileName", carInfo.FileName)
	idx, err := getIterableIndex(r.Context(), store, cid.MustParse(carInfo.PieceCid))
	if err != nil {
		log.Errorw("cannot get index", "cid", carInfo.PieceCid, "error", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	bs, err := blockstore.NewReadOnly(NewDownloadReaderAt(carInfo.FileName), idx, carv2.ZeroLengthSectionAsEOF(true))
	if err != nil {
		log.Errorw("cannot create blockstore", "cid", carInfo.DataCid, "fileName", carInfo.FileName, "error", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	lsys := storeutil.LinkSystemForBlockstore(bs)
	frisbii.NewHttpIpfs(r.Context(), lsys, frisbii.WithCompressionLevel(gzip.NoCompression)).ServeHTTP(w, r)
}

func getIterableIndex(ctx context.Context, store *bdclient.Store, pieceCid cid.Cid) (carindex.IterableIndex, error) {
	// 先从LRU缓存中查找
	key := pieceCid.String()
	if idx, ok := indexCache.Get(key); ok {
		log.Debugw("get index from cache", "pieceCid", pieceCid)
		return idx, nil
	}

	// 如果缓存中没有，则从store中获取
	idx, err := store.GetIndex(ctx, pieceCid)
	if err != nil {
		return nil, err
	}

	switch concrete := idx.(type) {
	case carindex.IterableIndex:
		// 存入LRU缓存
		indexCache.Add(key, concrete)
		return concrete, nil
	default:
		return nil, fmt.Errorf("expected index to be MultihashIndexSorted but got %T", idx)
	}
}
