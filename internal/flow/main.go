package flow

import (
	"log"
	"wallbox_nord_pool/internal/nordpool"
	"wallbox_nord_pool/internal/wallbox"
)

type State struct {
	ChargerStatus wallbox.ChargerStatus
	PriceStatus   nordpool.PriceStatus
}

var (
	LockedWaitingPriceGood = State{wallbox.LockedWaiting, nordpool.PriceGood}
	PausedPriceGood        = State{wallbox.Paused, nordpool.PriceGood}
	ChargingPriceTooBig    = State{wallbox.Charging, nordpool.PriceTooBig}
)

type ActionFunc func(wb wallbox.Wallbox, energyCost float64) (err error)

func DoFlow(state State) (action ActionFunc) {
	switch state {
	case LockedWaitingPriceGood:
		return actionUnlock
	case PausedPriceGood:
		return actionResume
	case ChargingPriceTooBig:
		return actionPause
	default:
		return actionEmpty
	}
}

func NewFlowsState(price float64, desiredPrice float64, chargerStatus wallbox.ChargerStatus) (flowState State) {
	if price > desiredPrice {
		return State{chargerStatus, nordpool.PriceTooBig}
	} else {
		return State{chargerStatus, nordpool.PriceGood}
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
