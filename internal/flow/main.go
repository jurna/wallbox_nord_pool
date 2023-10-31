package flow

import (
	"log"
	"wallbox_nord_pool/internal/nordpool"
	"wallbox_nord_pool/internal/wallbox"
)

type FlowState struct {
	ChargerStatus wallbox.ChargerStatus
	PriceStatus   nordpool.PriceStatus
}

type ActionFunc func(wb wallbox.Wallbox, energyCost float64) (err error)

func DoFlow(state FlowState) (action ActionFunc) {
	switch state {
	case FlowState{wallbox.LockedWaiting, nordpool.PriceGood}:
		return actionUnlock
	case FlowState{wallbox.Waiting, nordpool.PriceGood}:
		return actionResume
	case FlowState{wallbox.Charging, nordpool.PriceTooBig}:
		return actionPause
	default:
		return actionEmpty
	}
}

func NewFlowsState(price float64, cutOffPrice float64, chargerStatus wallbox.ChargerStatus) (flowState FlowState) {
	if price > cutOffPrice {
		return FlowState{chargerStatus, nordpool.PriceTooBig}
	} else {
		return FlowState{chargerStatus, nordpool.PriceGood}
	}
}
func actionUnlock(wb wallbox.Wallbox, energyCost float64) (err error) {
	log.Printf("Setting energy cost to %f and performing action unlock", energyCost)
	err = wb.SetEnergyCost(energyCost)
	if err != nil {
		return
	}
	return wb.Unlock()
}

func actionResume(wb wallbox.Wallbox, energyCost float64) (err error) {
	log.Printf("Setting energy cost to %f and performing action resume", energyCost)
	err = wb.SetEnergyCost(energyCost)
	if err != nil {
		return
	}
	return wb.ResumeCharging()
}

func actionPause(wb wallbox.Wallbox, _ float64) (err error) {
	log.Println("Performing action pause")
	return wb.PauseCharging()
}

func actionEmpty(_ wallbox.Wallbox, _ float64) (err error) {
	log.Println("Performing empty action")
	return err
}
