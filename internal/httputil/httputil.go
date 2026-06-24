package httputil

import (
	"net/http"

	"go.uber.org/zap"
)

func ServerError(w http.ResponseWriter, log *zap.Logger, userMsg string, logMsg string, err error) {
	log.Error(logMsg, zap.Error(err))
	http.Error(w, userMsg, http.StatusInternalServerError)
}
