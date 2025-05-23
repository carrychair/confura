package catchup

import (
	"context"
	"sync"
	"time"

	"github.com/Conflux-Chain/confura/store"
	"github.com/sirupsen/logrus"
)

type worker struct {
	// worker name
	name string
	// result channel to collect queried epoch data
	resultChan chan *store.EpochData
	// RPC client delegated to fetch blockchain data
	client IRpcClient
}

func mustNewWorker(name string, client IRpcClient, chanSize int) *worker {
	return &worker{
		name:       name,
		resultChan: make(chan *store.EpochData, chanSize),
		client:     client,
	}
}

func (w *worker) Sync(ctx context.Context, wg *sync.WaitGroup, epochFrom, epochTo, stepN uint64) {
	defer wg.Done()

	for eno := epochFrom; eno <= epochTo; {
		select {
		case <-ctx.Done():
			return
		default:
			epochData, err := w.fetchEpoch(eno)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"epochNo":    eno,
					"workerName": w.name,
				}).WithError(err).Info("Catch-up worker failed to fetch epoch")
				time.Sleep(time.Second)
				break
			}

			select {
			case <-ctx.Done():
				return
			case w.resultChan <- epochData:
				eno += stepN
			}
		}
	}
}

func (w *worker) Close() {
	w.client.Close()
	close(w.resultChan)
}

func (w *worker) Data() <-chan *store.EpochData {
	return w.resultChan
}

func (w *worker) fetchEpoch(epochNo uint64) (*store.EpochData, error) {
	epochData, err := w.client.QueryEpochData(context.Background(), epochNo, epochNo)
	if err == nil {
		return epochData[0], nil
	}

	return nil, err
}
