package hue

import (
	"encoding/json"
	"slices"
	"time"
)

func CheckMotion(r []byte, s []string) bool {
	type hueJson []struct {
		Creationtime time.Time `json:"creationtime"`
		Data         []struct {
			ID     string `json:"id"`
			IDV1   string `json:"id_v1"`
			Motion struct {
				Motion       bool `json:"motion"`
				MotionReport struct {
					Changed time.Time `json:"changed"`
					Motion  bool      `json:"motion"`
				} `json:"motion_report"`
				MotionValid bool `json:"motion_valid"`
			} `json:"motion"`
			Owner struct {
				Rid   string `json:"rid"`
				Rtype string `json:"rtype"`
			} `json:"owner"`
			Type string `json:"type"`
		} `json:"data"`
		ID   string `json:"id"`
		Type string `json:"type"`
	}
	var result hueJson
	json.Unmarshal(r, &result)
	for k := range result {
		if result[k].Data[0].Motion.Motion && slices.Contains(s, result[k].Data[0].IDV1) {
			return true
		}
	}
	return false
}
