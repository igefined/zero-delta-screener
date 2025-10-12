package domain

import (
	"context"
	"time"
)

// SwapExecutor interface for executing swaps on different exchanges
type SwapExecutor interface {
	ExecuteSwap(ctx context.Context, request *SwapRequest) (*SwapResult, error)
	GetName() string
}

// SwapRequest represents a swap transaction request
type SwapRequest struct {
	Symbol      string  `json:"symbol"`
	Side        string  `json:"side"` // "buy" or "sell"
	Amount      float64 `json:"amount"`
	Price       float64 `json:"price,omitempty"` // For limit orders
	OrderType   string  `json:"order_type"`      // "market" or "limit"
	RequestID   string  `json:"request_id"`
	RequestTime time.Time `json:"request_time"`
}

// SwapResult contains the result of a swap execution
type SwapResult struct {
	TransactionID    string    `json:"transaction_id"`
	Symbol          string    `json:"symbol"`
	Side            string    `json:"side"`
	Amount          float64   `json:"amount"`
	Price           float64   `json:"price"`
	Fee             float64   `json:"fee"`
	Status          string    `json:"status"` // "pending", "confirmed", "failed"
	RequestTime     time.Time `json:"request_time"`
	SubmissionTime  time.Time `json:"submission_time"`
	ConfirmationTime *time.Time `json:"confirmation_time,omitempty"`
	BlockNumber     *int64    `json:"block_number,omitempty"`
	BlockHash       *string   `json:"block_hash,omitempty"`
	GasUsed         *int64    `json:"gas_used,omitempty"`
	RequestID       string    `json:"request_id"`
	
	// Timing metrics
	SubmissionLatency    time.Duration `json:"submission_latency"`    // Time from request to submission
	ConfirmationLatency  *time.Duration `json:"confirmation_latency,omitempty"` // Time from submission to confirmation
	TotalLatency         *time.Duration `json:"total_latency,omitempty"`        // Time from request to confirmation
}

// SwapTiming tracks detailed timing metrics for swap execution
type SwapTiming struct {
	RequestID        string        `json:"request_id"`
	RequestTime      time.Time     `json:"request_time"`
	PreProcessTime   *time.Time    `json:"pre_process_time,omitempty"`
	SubmissionTime   *time.Time    `json:"submission_time,omitempty"`
	MempoolTime      *time.Time    `json:"mempool_time,omitempty"`
	ConfirmationTime *time.Time    `json:"confirmation_time,omitempty"`
	FinalizedTime    *time.Time    `json:"finalized_time,omitempty"`
	
	// Derived timing metrics
	PreProcessLatency   *time.Duration `json:"pre_process_latency,omitempty"`
	SubmissionLatency   *time.Duration `json:"submission_latency,omitempty"`
	MempoolLatency      *time.Duration `json:"mempool_latency,omitempty"`
	ConfirmationLatency *time.Duration `json:"confirmation_latency,omitempty"`
	FinalizationLatency *time.Duration `json:"finalization_latency,omitempty"`
	TotalLatency        *time.Duration `json:"total_latency,omitempty"`
}

// UpdateTiming updates the timing metrics based on current stage
func (st *SwapTiming) UpdateTiming(stage string) {
	now := time.Now()
	
	switch stage {
	case "pre_process":
		st.PreProcessTime = &now
		if st.PreProcessLatency == nil {
			latency := now.Sub(st.RequestTime)
			st.PreProcessLatency = &latency
		}
	case "submission":
		st.SubmissionTime = &now
		if st.SubmissionLatency == nil {
			latency := now.Sub(st.RequestTime)
			st.SubmissionLatency = &latency
		}
	case "mempool":
		st.MempoolTime = &now
		if st.MempoolLatency == nil && st.SubmissionTime != nil {
			latency := now.Sub(*st.SubmissionTime)
			st.MempoolLatency = &latency
		}
	case "confirmation":
		st.ConfirmationTime = &now
		if st.ConfirmationLatency == nil && st.SubmissionTime != nil {
			latency := now.Sub(*st.SubmissionTime)
			st.ConfirmationLatency = &latency
		}
		if st.TotalLatency == nil {
			latency := now.Sub(st.RequestTime)
			st.TotalLatency = &latency
		}
	case "finalized":
		st.FinalizedTime = &now
		if st.FinalizationLatency == nil && st.ConfirmationTime != nil {
			latency := now.Sub(*st.ConfirmationTime)
			st.FinalizationLatency = &latency
		}
		if st.TotalLatency == nil {
			latency := now.Sub(st.RequestTime)
			st.TotalLatency = &latency
		}
	}
}