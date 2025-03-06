package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"

	carv1 "github.com/ipld/go-car"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/service-sdk/go-sdk-qn/v2/operation"
)

type CarInfo struct {
	DataCid   string `json:"dataCid"`
	PieceCid  string `json:"pieceCid"`
	PieceSize int64  `json:"pieceSize"`
	CarSize   int64  `json:"carSize"`
	FileName  string `json:"fileName"`
}

type MinioConfig struct {
	Endpoint  string `json:"endpoint"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	Bucket    string `json:"bucket"`
	Region    string `json:"region"`
	UseSSL    bool   `json:"use_ssl"`
}

func getMinioConfig() (*MinioConfig, error) {
	minio := os.Getenv("MINIO")
	if minio == "" {
		return nil, nil
	}

	var mc MinioConfig
	conf, err := os.Open(minio)
	if err != nil {
		return nil, err
	}
	defer conf.Close()

	err = json.NewDecoder(conf).Decode(&mc)
	if err != nil {
		return nil, err
	}

	return &mc, nil
}

func (s *Server) Fetch(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "path is required", http.StatusBadRequest)
		return
	}

	parallel, err := strconv.Atoi(r.URL.Query().Get("parallel"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 立即返回成功响应
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("fetch task started"))

	// 在后台执行任务
	go func() {
		log.Infow("fetch start background", "path", path, "parallel", parallel)
		err := s.fetch(path, parallel)
		if err != nil {
			log.Errorw("fetch failed", "path", path, "error", err)
		} else {
			log.Infow("fetch completed", "path", path)
		}
	}()
}

func (s *Server) fetch(path string, parallel int) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, parallel)

	for scanner.Scan() {
		var carInfo CarInfo
		err := json.Unmarshal(scanner.Bytes(), &carInfo)
		if err != nil {
			log.Errorf("json unmarshal error: %v", err)
			continue
		}

		wg.Add(1)
		semaphore <- struct{}{}
		go func(info CarInfo) {
			defer wg.Done()
			defer func() { <-semaphore }()

			err := s.fetchBlock(info)
			if err != nil {
				log.Errorf("%s fetch error: %v", info.FileName, err)
			} else {
				log.Infof("%s fetch success", info.FileName)
			}
		}(carInfo)
	}

	return nil
}

func (s *Server) fetchBlock(info CarInfo) error {
	var carReader *carv1.CarReader
	if os.Getenv("MINIO") != "" && s.mc != nil {
		minioClient, err := minio.New(s.mc.Endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(s.mc.AccessKey, s.mc.SecretKey, ""),
			Secure: s.mc.UseSSL,
			Region: s.mc.Region,
		})
		if err != nil {
			return err
		}
		object, err := minioClient.GetObject(context.Background(), s.mc.Bucket, info.FileName, minio.GetObjectOptions{})
		if err != nil {
			return err
		}
		defer object.Close()

		carReader, err = carv1.NewCarReader(object)
		if err != nil {
			return err
		}

	} else if os.Getenv("QINIU") != "" {
		downloader := operation.NewDownloaderV2()
		resp, err := downloader.DownloadRaw(info.FileName, nil)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		carReader, err = carv1.NewCarReader(resp.Body)
		if err != nil {
			return err
		}
	} else if os.Getenv("FILE_PATH") != "" {
		file, err := os.Open(os.Getenv("FILE_PATH") + "/" + info.FileName)
		if err != nil {
			return err
		}
		defer file.Close()
		carReader, err = carv1.NewCarReader(file)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("no config")
	}

	for {
		block, err := carReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		err = s.upsert(&RootBlock{
			Root:  block.Cid().String(),
			Block: block.RawData(),
		})
		if err != nil {
			return err
		}
	}

	return nil
}
