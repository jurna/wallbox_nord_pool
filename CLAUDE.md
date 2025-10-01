# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is an AWS Lambda-based Go application that automatically manages EV charging based on Nord Pool electricity prices. The system fetches real-time electricity pricing, compares it against configured thresholds, and controls a Wallbox EV charger to charge when prices are optimal.

## Architecture

### Core Components

**Main Entry Point** (`main.go`)
- AWS Lambda handler that orchestrates the entire flow
- Reads configuration from S3 (`config.yaml`)
- Coordinates between Nord Pool pricing and Wallbox control
- Calculates desired price based on minimum price until charge deadline or configured max price

**Flow Controller** (`internal/flow/main.go`)
- State machine that determines actions based on combined state of charger status and price status
- Maps charger status (Locked/Waiting, Paused, Scheduled, Charging) + price status (Good/TooBig) to actions (unlock, resume, pause, empty)
- Price threshold logic: Price is considered "TooBig" when `price - desiredPrice >= 0.01` (1 cent threshold)

**Nord Pool Integration** (`internal/nordpool/main.go`)
- Fetches electricity prices from Elering API (`https://dashboard.elering.ee/api/nps/price`)
- Caches pricing data in S3 with format: `nord_pool_YYYY-MM-DD_H.json`
- Calculates final price including VAT and transmission costs (different rates for day/night and weekday/weekend)
- Finds minimum price until charge deadline (configurable day/night deadlines: `charge-till-hour-day` and `charge-till-hour-night`)

**Wallbox Integration** (`internal/wallbox/main.go`)
- Authenticates with Wallbox API using basic auth, caches JWT token in S3 (`user_token.json`)
- Maps numeric status codes to charger states (e.g., 210=LockedWaiting, 193-195=Charging, 178,182=Paused)
- Supports actions: Unlock, PauseCharging, ResumeCharging, SetEnergyCost

### Data Flow

1. Lambda triggered (scheduled invocation)
2. Fetch current Nord Pool price + find minimum price until charge deadline
3. Calculate desired price (min of minimum price or configured max price)
4. Get Wallbox charger status
5. Flow state machine determines action based on (charger_status, price_status)
6. Execute action: unlock/resume charging if price good, pause if price too big

### Configuration

Configuration is stored in S3 as `config.yaml` with structure:
```yaml
nord-pool:
  max-price: float          # Maximum acceptable price
  charge-till-hour-day: int # Hour by which to complete charging during day period (e.g., 16 = 4 PM)
  charge-till-hour-night: int # Hour by which to complete charging during night period (e.g., 7 = 7 AM)
  vat: float                # VAT rate (e.g., 0.21 = 21%)
  timezone: string          # Timezone for Nord Pool prices
  transmission-cost:
    day: float              # Day transmission cost
    night: float            # Night transmission cost
    day-starts-at: int      # Hour when day rate starts (e.g., 7)
    night-starts-at: int    # Hour when night rate starts (e.g., 23)
    timezone: string        # Timezone for transmission cost calculation
wallbox:
  username: string
  password: string
  device-id: string
```

## Common Commands

**Build:**
```bash
go build
```

**Run tests:**
```bash
go test ./...
```

**Run single test:**
```bash
go test ./internal/flow -run TestDoFlow
go test ./internal/flow -run TestNewFlowsState
```

**Build for Lambda (Linux/AMD64):**
```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build
```

**Create deployment package:**
```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build
zip wallbox_nord_pool.zip wallbox_nord_pool
```

## Environment Variables

- `AWS_REGION`: AWS region for S3 access
- `AWS_S3_BUCKET`: S3 bucket name for config and cache storage

## Testing Notes

- Flow logic tests verify state machine transitions and price threshold calculations
- Price threshold is 0.01 (1 cent difference between current and desired price)
- Charge deadline logic differs between day period (uses `charge-till-hour-night`) and night period (uses `charge-till-hour-day`)