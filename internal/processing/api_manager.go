package processing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gophermart/internal/db"
	"gophermart/internal/order/model"
	"gophermart/internal/utils"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

type apiManager struct {
	client http.Client
	host   string
	db     db.Storage
	logger *zap.SugaredLogger
}

type response struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual"`
}

const url = "%v/api/orders/%v"

type ProcessResult int

const (
	InProgress = iota
	Completed
	Failed
	Undefined
)

func getResult(status string) (ProcessResult, error) {
	switch status {
	case "REGISTERED", "PROCESSING":
		return InProgress, nil
	case "INVALID":
		return Failed, nil
	case "PROCESSED":
		return Completed, nil
	default:
		return Undefined, errors.New("undefined status")
	}
}

func (m *apiManager) getCalc(number int64) (float64, ProcessResult, error) {
	if r, err := m.client.Get(fmt.Sprintf(url, m.host, number)); err != nil {
		return 0, Undefined, err
	} else {
		defer r.Body.Close()
		switch r.StatusCode {
		case 200:
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				return 0, Undefined, err
			}
			res := &response{}
			json.Unmarshal(body, &res)
			result, err := getResult(res.Status)
			return res.Accrual, result, err
			//429, 500
		default:
			return 0, InProgress, nil
		}
	}
}

func mapResultOnStatus(r ProcessResult) model.OrderStatus {
	switch r {
	case Completed:
		return model.Processed
	case Failed:
		return model.Invalid
	default:
		return model.Processing
	}
}

func (m *apiManager) updF(nums []int64) map[int64]db.CalcAmountsUpdateResult {
	result := make(map[int64]db.CalcAmountsUpdateResult)
	for i := 0; i < len(nums); i++ {
		if accrual, respResult, err := m.getCalc(nums[i]); err != nil {
			m.logger.Errorf("update order by number %v failed: %w", nums[i], err)
		} else {
			result[nums[i]] = db.CalcAmountsUpdateResult{Accrual: utils.GetPersistentAccrual(accrual), Status: mapResultOnStatus(respResult)}
		}
	}
	return result
}

func (m *apiManager) runCollectСalcs() {
	offset := 0
	limit := 10
	for {
		selectedCount, err := m.db.CalcAmounts(offset, limit, m.updF)
		if err != nil {
			m.logger.Errorf("error on runCollectСalcs: %w", err)
			return
		}
		if selectedCount < limit {
			return
		}

		offset += limit
	}
}

var once sync.Once

func RunDaemon(client http.Client, host string, db db.Storage, logger *zap.SugaredLogger, ctx context.Context, wg *sync.WaitGroup) {
	go once.Do(func() {
		wg.Add(1)
		defer wg.Done()

		ticker := time.NewTicker(time.Second * 1)
		m := &apiManager{client, host, db, logger}
		select {
		case <-ticker.C:
			logger.Infof("run update orders...")
			m.runCollectСalcs()

		case <-ctx.Done():
			ticker.Stop()
			return
		}
	})
}
