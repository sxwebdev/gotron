package client

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sxwebdev/gotron/schema/pb/api"
	"github.com/sxwebdev/gotron/schema/pb/core"
)

// mockMetricsCollector records calls to MetricsCollector methods.
type mockMetricsCollector struct {
	requests []recordedRequest
	retries  []recordedRetry
	pools    []recordedPool
}

type recordedRequest struct {
	blockchain, method, status string
	duration                   time.Duration
}

type recordedRetry struct {
	blockchain, method string
}

type recordedPool struct {
	blockchain              string
	total, healthy, disabled int
}

func (m *mockMetricsCollector) RecordRequest(blockchain, method, status string, duration time.Duration) {
	m.requests = append(m.requests, recordedRequest{blockchain, method, status, duration})
}

func (m *mockMetricsCollector) RecordRetry(blockchain, method string) {
	m.retries = append(m.retries, recordedRetry{blockchain, method})
}

func (m *mockMetricsCollector) SetPoolHealth(blockchain string, total, healthy, disabled int) {
	m.pools = append(m.pools, recordedPool{blockchain, total, healthy, disabled})
}

var _ MetricsCollector = (*mockMetricsCollector)(nil)

// mockTransport is a minimal Transport for testing MetricsTransport.
type mockTransport struct {
	err error
}

func (m *mockTransport) GetNowBlock(ctx context.Context) (*api.BlockExtention, error) {
	return nil, m.err
}

// Stub all other Transport methods to satisfy the interface.
func (m *mockTransport) GetAccount(context.Context, *core.Account) (*core.Account, error)             { return nil, m.err }
func (m *mockTransport) GetAccountResource(context.Context, *core.Account) (*api.AccountResourceMessage, error) { return nil, m.err }
func (m *mockTransport) CreateAccount(context.Context, *core.AccountCreateContract) (*api.TransactionExtention, error) { return nil, m.err }
func (m *mockTransport) GetBlockByNum(context.Context, int64) (*api.BlockExtention, error)             { return nil, m.err }
func (m *mockTransport) GetBlockById(context.Context, []byte) (*core.Block, error)                     { return nil, m.err }
func (m *mockTransport) GetBlockByLimitNext(context.Context, int64, int64) (*api.BlockListExtention, error) { return nil, m.err }
func (m *mockTransport) GetBlockByLatestNum(context.Context, int64) (*api.BlockListExtention, error)   { return nil, m.err }
func (m *mockTransport) GetTransactionInfoByBlockNum(context.Context, int64) (*api.TransactionInfoList, error) { return nil, m.err }
func (m *mockTransport) GetTransactionById(context.Context, []byte) (*core.Transaction, error)         { return nil, m.err }
func (m *mockTransport) GetTransactionInfoById(context.Context, []byte) (*core.TransactionInfo, error) { return nil, m.err }
func (m *mockTransport) BroadcastTransaction(context.Context, *core.Transaction) (*api.Return, error)  { return nil, m.err }
func (m *mockTransport) CreateTransaction(context.Context, *core.TransferContract) (*api.TransactionExtention, error) { return nil, m.err }
func (m *mockTransport) TriggerContract(context.Context, *core.TriggerSmartContract) (*api.TransactionExtention, error) { return nil, m.err }
func (m *mockTransport) TriggerConstantContract(context.Context, *core.TriggerSmartContract) (*api.TransactionExtention, error) { return nil, m.err }
func (m *mockTransport) EstimateEnergy(context.Context, *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error) { return nil, m.err }
func (m *mockTransport) DeployContract(context.Context, *core.CreateSmartContract) (*api.TransactionExtention, error) { return nil, m.err }
func (m *mockTransport) GetContract(context.Context, []byte) (*core.SmartContract, error)               { return nil, m.err }
func (m *mockTransport) UpdateSetting(context.Context, *core.UpdateSettingContract) (*api.TransactionExtention, error) { return nil, m.err }
func (m *mockTransport) UpdateEnergyLimit(context.Context, *core.UpdateEnergyLimitContract) (*api.TransactionExtention, error) { return nil, m.err }
func (m *mockTransport) GetAccountResourceMessage(context.Context, *core.Account) (*api.AccountResourceMessage, error) { return nil, m.err }
func (m *mockTransport) GetDelegatedResource(context.Context, *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) { return nil, m.err }
func (m *mockTransport) GetDelegatedResourceV2(context.Context, *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) { return nil, m.err }
func (m *mockTransport) GetDelegatedResourceAccountIndex(context.Context, []byte) (*core.DelegatedResourceAccountIndex, error) { return nil, m.err }
func (m *mockTransport) GetDelegatedResourceAccountIndexV2(context.Context, []byte) (*core.DelegatedResourceAccountIndex, error) { return nil, m.err }
func (m *mockTransport) GetCanDelegatedMaxSize(context.Context, *api.CanDelegatedMaxSizeRequestMessage) (*api.CanDelegatedMaxSizeResponseMessage, error) { return nil, m.err }
func (m *mockTransport) DelegateResource(context.Context, *core.DelegateResourceContract) (*api.TransactionExtention, error) { return nil, m.err }
func (m *mockTransport) UnDelegateResource(context.Context, *core.UnDelegateResourceContract) (*api.TransactionExtention, error) { return nil, m.err }
func (m *mockTransport) GetAssetIssueById(context.Context, []byte) (*core.AssetIssueContract, error)  { return nil, m.err }
func (m *mockTransport) GetAssetIssueListByName(context.Context, []byte) (*api.AssetIssueList, error) { return nil, m.err }
func (m *mockTransport) ListNodes(context.Context) (*api.NodeList, error)                               { return nil, m.err }
func (m *mockTransport) GetChainParameters(context.Context) (*core.ChainParameters, error)              { return nil, m.err }
func (m *mockTransport) GetNextMaintenanceTime(context.Context) (*api.NumberMessage, error)             { return nil, m.err }
func (m *mockTransport) TotalTransaction(context.Context) (*api.NumberMessage, error)                   { return nil, m.err }
func (m *mockTransport) Close() error                                                                   { return nil }

// --- Built-in Metrics tests ---

func TestNewMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)
	if m == nil {
		t.Fatal("NewMetrics returned nil")
	}
}

func TestMetricsRecordRequest(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	m.RecordRequest("tron", "GetNowBlock", "success", 100*time.Millisecond)

	val := testutil.ToFloat64(m.requestsTotal.WithLabelValues("tron", "GetNowBlock", "success"))
	if val != 1 {
		t.Errorf("requestsTotal: got %v, want 1", val)
	}

	count := testutil.CollectAndCount(m.requestDuration)
	if count == 0 {
		t.Error("requestDuration: expected observations, got none")
	}
}

func TestMetricsRecordRetry(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	m.RecordRetry("tron", "GetNowBlock")
	m.RecordRetry("tron", "GetNowBlock")

	val := testutil.ToFloat64(m.retriesTotal.WithLabelValues("tron", "GetNowBlock"))
	if val != 2 {
		t.Errorf("retriesTotal: got %v, want 2", val)
	}
}

func TestMetricsSetPoolHealth(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	m.SetPoolHealth("tron", 5, 4, 1)

	if v := testutil.ToFloat64(m.poolTotal.WithLabelValues("tron")); v != 5 {
		t.Errorf("poolTotal: got %v, want 5", v)
	}
	if v := testutil.ToFloat64(m.poolHealthy.WithLabelValues("tron")); v != 4 {
		t.Errorf("poolHealthy: got %v, want 4", v)
	}
	if v := testutil.ToFloat64(m.poolDisabled.WithLabelValues("tron")); v != 1 {
		t.Errorf("poolDisabled: got %v, want 1", v)
	}
}

func TestMetricsInFlight(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	m.IncInFlight()
	m.IncInFlight()
	if v := testutil.ToFloat64(m.requestsInFlight); v != 2 {
		t.Errorf("in-flight after 2 inc: got %v, want 2", v)
	}

	m.DecInFlight()
	if v := testutil.ToFloat64(m.requestsInFlight); v != 1 {
		t.Errorf("in-flight after dec: got %v, want 1", v)
	}
}

// --- MetricsTransport tests ---

func TestMetricsTransportSuccess(t *testing.T) {
	mock := &mockMetricsCollector{}
	mt := NewMetricsTransport(&mockTransport{}, mock, "tron")

	_, _ = mt.GetNowBlock(context.Background())

	if len(mock.requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(mock.requests))
	}
	r := mock.requests[0]
	if r.blockchain != "tron" {
		t.Errorf("blockchain: got %q, want %q", r.blockchain, "tron")
	}
	if r.method != "GetNowBlock" {
		t.Errorf("method: got %q, want %q", r.method, "GetNowBlock")
	}
	if r.status != "success" {
		t.Errorf("status: got %q, want %q", r.status, "success")
	}
	if r.duration <= 0 {
		t.Errorf("duration: got %v, want > 0", r.duration)
	}
}

func TestMetricsTransportError(t *testing.T) {
	mock := &mockMetricsCollector{}
	mt := NewMetricsTransport(&mockTransport{err: errors.New("connection refused")}, mock, "tron")

	_, err := mt.GetNowBlock(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}

	if len(mock.requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(mock.requests))
	}
	if mock.requests[0].status != "error" {
		t.Errorf("status: got %q, want %q", mock.requests[0].status, "error")
	}
}

func TestMetricsTransportCustomBlockchain(t *testing.T) {
	mock := &mockMetricsCollector{}
	mt := NewMetricsTransport(&mockTransport{}, mock, "ethereum")

	_, _ = mt.GetNowBlock(context.Background())

	if len(mock.requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(mock.requests))
	}
	if mock.requests[0].blockchain != "ethereum" {
		t.Errorf("blockchain: got %q, want %q", mock.requests[0].blockchain, "ethereum")
	}
}

func TestMetricsTransportDefaultBlockchain(t *testing.T) {
	mock := &mockMetricsCollector{}
	mt := NewMetricsTransport(&mockTransport{}, mock, "")

	_, _ = mt.GetNowBlock(context.Background())

	if len(mock.requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(mock.requests))
	}
	if mock.requests[0].blockchain != "tron" {
		t.Errorf("blockchain: got %q, want %q", mock.requests[0].blockchain, "tron")
	}
}
