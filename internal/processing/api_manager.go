package processing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gophermart/internal/config"
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
	cfg    *config.Config
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

const (
	Registered = "REGISTERED"
	Processing = "PROCESSING"
	Invalid    = "INVALID"
	Processed  = "PROCESSED"
)

func getResult(status string) (ProcessResult, error) {
	switch status {
	case Registered, Processing:
		return InProgress, nil
	case Invalid:
		return Failed, nil
	case Processed:
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

func (m *apiManager) runCollect??alcs() {
	offset := 0

	for {
		selectedCount, err := m.db.CalcAmounts(offset, m.cfg.OrdersUpdateCountInPar, m.updF)
		if err != nil {
			m.logger.Errorf("error on runCollect??alcs: %w", err)
			return
		}
		if selectedCount < m.cfg.OrdersUpdateCountInPar {
			return
		}

		offset += m.cfg.OrdersUpdateCountInPar
	}
}

var once sync.Once

func RunDaemon(
	client http.Client,
	host string, db db.Storage,
	logger *zap.SugaredLogger,
	ctx context.Context,
	wg *sync.WaitGroup,
	cfg *config.Config) {
	go once.Do(func() {
		wg.Add(1)
		defer wg.Done()

		ticker := time.NewTicker(time.Second * 1)
		m := &apiManager{client, host, db, logger, cfg}
		for {
			select {
			case <-ticker.C:
				logger.Infof("run update orders...")
				m.runCollect??alcs()

			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	})
}
