package scheduler

import (
	"encoding/json"
	"goInsight/global"
	"goInsight/internal/orders/models"
	"time"

	"sync"

	"github.com/robfig/cron/v3"
)

var Cron *cron.Cron
var Executor func(orderID string, username string) error
var jobMap = make(map[string]cron.EntryID)
var mapMutex sync.Mutex

// SetExecutor sets the executor function
func SetExecutor(exec func(orderID string, username string) error) {
	Executor = exec
}

// Init initializes the cron scheduler
func Init() {
	Cron = cron.New(cron.WithSeconds())
	Cron.Start()
	ScanAndRegister()
}

// ScanAndRegister scans the database for unexecuted orders with a schedule time and registers them
func ScanAndRegister() {
	var records []models.InsightOrderRecords
	// Scan for orders that are 'Approved' and have a schedule time
	global.App.DB.Model(&models.InsightOrderRecords{}).
		Where("progress = ?", "已批准").
		Where("schedule_time IS NOT NULL").
		Scan(&records)

	for _, record := range records {
		AddJob(record)
	}
}

// AddJob adds a scheduled execution job for an order
func AddJob(record models.InsightOrderRecords) {
	if record.ScheduleTime == nil {
		return
	}

	targetTime := *record.ScheduleTime
	now := time.Now()

	// If the scheduled time is in the past, execute immediately (or consider it missed and ignore?)
	// Given the user wants to "re-register", implying persistence of intent.
	// If the server was down during the scheduled time, we should probably execute it.
	if targetTime.Before(now) {
		go executeOrder(record.OrderID.String())
		return
	}

	// Use custom schedule for one-time execution
	entryID := Cron.Schedule(&OneTimeSchedule{Time: targetTime}, cron.FuncJob(func() {
		// Remove from map before execution to avoid stale entries,
		// though strictly not necessary as Next() returns zero time after execution.
		// But good for cleanup.
		mapMutex.Lock()
		delete(jobMap, record.OrderID.String())
		mapMutex.Unlock()

		executeOrder(record.OrderID.String())
	}))

	mapMutex.Lock()
	jobMap[record.OrderID.String()] = entryID
	mapMutex.Unlock()
}

// RemoveJob removes a scheduled job for an order
func RemoveJob(orderID string) {
	mapMutex.Lock()
	defer mapMutex.Unlock()

	if entryID, ok := jobMap[orderID]; ok {
		Cron.Remove(entryID)
		delete(jobMap, orderID)
	}
}

// OneTimeSchedule implements cron.Schedule for a single execution
type OneTimeSchedule struct {
	Time time.Time
}

func (s *OneTimeSchedule) Next(t time.Time) time.Time {
	if s.Time.After(t) {
		return s.Time
	}
	return time.Time{}
}

func executeOrder(orderID string) {
	var record models.InsightOrderRecords
	global.App.DB.Where("order_id = ?", orderID).First(&record)

	// Double check status before execution
	if record.Progress != "已批准" {
		return
	}

	var executorList []string
	_ = json.Unmarshal([]byte(record.Executor), &executorList)

	username := ""
	if len(executorList) > 0 {
		username = executorList[0]
	} else {
		username = record.Applicant
	}

	if Executor != nil {
		err := Executor(orderID, username)
		if err != nil {
			global.App.Log.Errorf("Scheduled execution failed for order %s: %v", orderID, err)
		} else {
			global.App.Log.Infof("Scheduled execution completed for order %s", orderID)
		}
	}
}
