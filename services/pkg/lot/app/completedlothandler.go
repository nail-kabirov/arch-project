package app

import (
	"github.com/sirupsen/logrus"

	"context"
	"time"
)

const handleCompletedLotDelayDelay = time.Second * 5

func StartCompletedLotsHandler(ctx context.Context, lotService LotService, logger *logrus.Logger) {
	handler := completedLotsHandler{
		lotService: lotService,
		logger:     logger,
	}
	handler.start(ctx)
}

type completedLotsHandler struct {
	lotService LotService
	logger     *logrus.Logger
}

func (handler *completedLotsHandler) start(ctx context.Context) {
	ticker := time.NewTicker(handleCompletedLotDelayDelay)

	go func() {
		for {
			select {
			case <-ticker.C:
				handler.handleCompletedLots()
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (handler *completedLotsHandler) handleCompletedLots() {
	err := handler.lotService.ProcessCompletedLots()
	if err != nil {
		handler.logger.Error(err)
	}
}
