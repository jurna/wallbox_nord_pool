package flow

import (
	"reflect"
	"runtime"
	"testing"
	"wallbox_nord_pool/internal/nordpool"
	"wallbox_nord_pool/internal/wallbox"
)

func TestDoFlow(t *testing.T) {
	tests := []struct {
		name       string
		state      State
		wantAction string
	}{
		{name: "LockedWaitingPriceGood", state: LockedWaitingPriceGood, wantAction: "wallbox_nord_pool/internal/flow.actionUnlock"},
		{name: "PausedPriceGood", state: PausedPriceGood, wantAction: "wallbox_nord_pool/internal/flow.actionResume"},
		{name: "ScheduledPriceGood", state: ScheduledPriceGood, wantAction: "wallbox_nord_pool/internal/flow.actionResume"},
		{name: "ChargingPriceTooBig", state: ChargingPriceTooBig, wantAction: "wallbox_nord_pool/internal/flow.actionPause"},
		{name: "WaitingForCarPriceGood", state: State{wallbox.WaitingForCar, nordpool.PriceGood}, wantAction: "wallbox_nord_pool/internal/flow.actionEmpty"},
		{name: "WaitingPriceGood", state: State{wallbox.Waiting, nordpool.PriceGood}, wantAction: "wallbox_nord_pool/internal/flow.actionEmpty"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAction := DoFlow(tt.state)
			gotActionName := runtime.FuncForPC(reflect.ValueOf(gotAction).Pointer()).Name()
			if gotActionName != tt.wantAction {
				t.Errorf("DoFlow() = %s, want %s", gotActionName, tt.wantAction)
			}
		})
	}
}

func TestNewFlowsState(t *testing.T) {
	tests := []struct {
		name            string
		price           float64
		wantPriceStatus nordpool.PriceStatus
	}{
		{name: "PriceEqualsDesiredPrice", price: 0.10, wantPriceStatus: nordpool.PriceGood},
		{name: "PriceHigherWithinThreshold", price: 0.105, wantPriceStatus: nordpool.PriceGood},
		{name: "PriceHigher", price: 0.12, wantPriceStatus: nordpool.PriceTooBig},
		{name: "PriceLower", price: 0.09, wantPriceStatus: nordpool.PriceGood},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			priceStatus := NewFlowsState(tt.price, 0.10, wallbox.LockedWaiting).PriceStatus
			if priceStatus != tt.wantPriceStatus {
				t.Errorf("TestNewFlowsState() = %s, want %s", priceStatus, tt.wantPriceStatus)
			}
		})
	}
}
