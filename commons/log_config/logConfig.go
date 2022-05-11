package log_config

import (
	"context"

	logconst "github.eagleview.com/engineering/assess-platform-library/constants"
	"github.eagleview.com/engineering/assess-platform-library/log"
	plog "github.eagleview.com/engineering/platform-gosdk/log"
)

func InitLogging(logLevel string) {
	plog.SetFormat("json")
	l := plog.ParseLevel(logLevel)
	plog.SetLevel(l)
	traceId := log.TrackID(logconst.TrackIDCorrelationID)
	log.SetContextKeys(traceId)
}

func SetTraceIdInContext(ctx context.Context, reportId, workFlowId string) context.Context {
	newCtx := ctx
	if ctx != nil {
		coId := reportId + ":" + workFlowId
		traceId := log.TrackID(logconst.TrackIDCorrelationID)
		newCtx = context.WithValue(ctx, traceId, coId)
	}
	return newCtx
}
