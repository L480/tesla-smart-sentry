package util

import (
	"os"
	"strconv"
	"strings"
	"time"
)

func InTimeframe(s string) bool {
	schedule := strings.Split(s, "-")
	startHour, _ := strconv.Atoi(strings.Split(schedule[0], ":")[0])
	startMinute, _ := strconv.Atoi(strings.Split(schedule[0], ":")[1])
	endHour, _ := strconv.Atoi(strings.Split(schedule[1], ":")[0])
	endMinute, _ := strconv.Atoi(strings.Split(schedule[1], ":")[1])

	now := time.Now().Local()
	startDate := time.Date(now.Year(), now.Month(), now.Day(), startHour, startMinute, 0, 0, time.Local)
	endDate := time.Date(now.Year(), now.Month(), now.Day(), endHour, endMinute, 0, 0, time.Local)

	if now.After(startDate) && now.Before(endDate) {
		return true
	} else {
		return false
	}
}

func CheckEnvs(e []string) bool {
	complete := true
	for k := range e {
		if _, ok := os.LookupEnv(e[k]); !ok {
			complete = false
			break
		}
	}
	if complete {
		return true
	} else {
		return false
	}
}
