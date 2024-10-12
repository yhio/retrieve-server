package middleware

import (
	"context"
	"net/http"

	logging "github.com/ipfs/go-log/v2"
	"github.com/yhio/retrieve-server/metrics"
	"go.opencensus.io/tag"
)

var log = logging.Logger("middleware")

func Timer(handler http.HandlerFunc, name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Debugw("request", "method", r.Method, "path", r.URL.Path, "name", name, "remoteAddr", r.RemoteAddr)

		ctx, _ := tag.New(context.Background(), tag.Upsert(metrics.Endpoint, name))
		stop := metrics.Timer(ctx, metrics.APIRequestDuration)
		defer stop()

		handler(w, r)
	}
}
