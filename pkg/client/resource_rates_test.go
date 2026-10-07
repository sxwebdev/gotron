package client

import (
	"math"
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/sxwebdev/gotron/pkg/units"
)

// The least stake reaches the units the chain counts and a SUN less does not,
// on every network shape and in both modes; a contract call's stake is whole TRX
// whose TRX below falls short.
func TestResourceRatesStakeFor(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewPCG(1, 2))
	for range 20_000 {
		r := ResourceRates{
			TotalEnergyCurrentLimit: 1e9 + rng.Int64N(2e11),
			TotalEnergyWeight:       1e6 + rng.Int64N(5e10),
			TotalNetLimit:           1e9 + rng.Int64N(5e10),
			TotalNetWeight:          1e6 + rng.Int64N(8e10),
			Harden:                  rng.IntN(2) == 0,
		}
		for _, resource := range []ResourceType{ResourceTypeEnergy, ResourceTypeBandwidth} {
			n := 1 + rng.Int64N(200_000)
			staked := r.StakeFor(resource, n)
			if r.Limit(resource, staked) < n || r.Limit(resource, staked-1) >= n {
				t.Fatalf("%+v %v: StakeFor(%d) = %d gives %d, a SUN less %d", r, resource, n, staked, r.Limit(resource, staked), r.Limit(resource, staked-1))
			}
		}
		n := 1 + rng.Int64N(200_000)
		staked := r.CallStakeFor(n)
		if staked%units.SunPerTRX != 0 || r.Limit(ResourceTypeEnergy, staked) < n || r.Limit(ResourceTypeEnergy, staked-units.SunPerTRX) >= n {
			t.Fatalf("%+v: CallStakeFor(%d) = %d", r, n, staked)
		}
	}
}

// The exact share can sit below the chain's figure: at a limit of 25 and a
// weight of 1, 1.16 TRX is exactly 29 units, but 1.16 in doubles is a hair low
// and java-tron counts 28.
func TestResourceRatesStakeForPastTheExactShare(t *testing.T) {
	t.Parallel()
	r := ResourceRates{TotalEnergyCurrentLimit: 25, TotalEnergyWeight: 1}
	if got := r.Limit(ResourceTypeEnergy, 1_160_000); got != 28 {
		t.Fatalf("Limit of the exact share = %d, want 28", got)
	}
	if got := r.StakeFor(ResourceTypeEnergy, 29); got != 1_160_001 {
		t.Fatalf("StakeFor(29) = %d, want 1160001", got)
	}
	if got := r.CallStakeFor(29); got != 2_000_000 {
		t.Fatalf("CallStakeFor(29) = %d, want 2000000", got)
	}
}

// Limit against the chain: a Nile account with 58 050.2 TRX staked and
// 25 341.549771 TRX acquired for bandwidth, at the network's totals then,
// showed NetLimit 52651.
func TestResourceRatesLimitNile(t *testing.T) {
	t.Parallel()
	for _, harden := range []bool{false, true} {
		r := ResourceRates{TotalNetLimit: 43_200_000_000, TotalNetWeight: 68_422_218_480, Harden: harden}
		if got := r.Limit(ResourceTypeBandwidth, 58_050_200_000+25_341_549_771); got != 52_651 {
			t.Errorf("harden %t: Limit = %d, want the node's 52651", harden, got)
		}
	}
}

// 49 TRX at 1/49 is exactly 1 unit; in doubles 49 * (1.0/49) is
// 0.9999999999999999, which java-tron's (long) truncates to 0. Hardened, the
// chain computes in integers and counts 1.
func TestResourceRatesLimitModes(t *testing.T) {
	t.Parallel()
	r := ResourceRates{TotalEnergyCurrentLimit: 1, TotalEnergyWeight: 49}
	if got := r.Limit(ResourceTypeEnergy, 49*units.SunPerTRX); got != 0 {
		t.Errorf("double Limit = %d, want 0", got)
	}
	r.Harden = true
	if got := r.Limit(ResourceTypeEnergy, 49*units.SunPerTRX); got != 1 {
		t.Errorf("hardened Limit = %d, want 1", got)
	}
}

func TestResourceRatesEdges(t *testing.T) {
	t.Parallel()
	empty := ResourceRates{}
	if got := empty.Limit(ResourceTypeEnergy, units.SunPerTRX); got != 0 {
		t.Errorf("Limit on a network with no weight = %d, want 0", got)
	}
	if got := empty.StakeFor(ResourceTypeBandwidth, 1); got != math.MaxInt64 {
		t.Errorf("StakeFor on a network with no limit = %d, want MaxInt64", got)
	}
	if got := empty.CallStakeFor(1); got != math.MaxInt64 {
		t.Errorf("CallStakeFor on a network with no limit = %d, want MaxInt64", got)
	}
	r := ResourceRates{TotalNetLimit: 43_200_000_000, TotalNetWeight: 68_422_218_480}
	for _, n := range []int64{0, -5} {
		if got := r.StakeFor(ResourceTypeBandwidth, n); got != 0 {
			t.Errorf("StakeFor(%d) = %d, want 0", n, got)
		}
		if got := r.CallStakeFor(n); got != 0 {
			t.Errorf("CallStakeFor(%d) = %d, want 0", n, got)
		}
	}
	if got := r.Limit(ResourceTypeBandwidth, -1); got != 0 {
		t.Errorf("Limit of a negative stake = %d, want 0", got)
	}
	// Past int64: one unit costs more SUN than int64 holds.
	huge := ResourceRates{TotalEnergyCurrentLimit: 1, TotalEnergyWeight: math.MaxInt64}
	if got := huge.StakeFor(ResourceTypeEnergy, 2); got != math.MaxInt64 {
		t.Errorf("StakeFor past int64 = %d, want MaxInt64", got)
	}
	if got := huge.CallStakeFor(2); got != math.MaxInt64 {
		t.Errorf("CallStakeFor past int64 = %d, want MaxInt64", got)
	}
	hardened := ResourceRates{TotalNetLimit: math.MaxInt64, TotalNetWeight: 1, Harden: true}
	if got := hardened.Limit(ResourceTypeBandwidth, math.MaxInt64); got != math.MaxInt64 {
		t.Errorf("hardened Limit past int64 = %d, want MaxInt64", got)
	}
	for _, f := range []float64{math.NaN(), math.Inf(1), math.Inf(-1), 1.9, -1.9} {
		want := map[bool]int64{true: 0}[math.IsNaN(f)]
		switch {
		case math.IsInf(f, 1):
			want = math.MaxInt64
		case math.IsInf(f, -1):
			want = math.MinInt64
		case !math.IsNaN(f):
			want = int64(f)
		}
		if got := javaLong(f); got != want {
			t.Errorf("javaLong(%v) = %d, want %d", f, got, want)
		}
	}
}

// The exact share rounded up, against an independent ceiling. Dividing first in
// decimals rounded the quotient to 16 digits and came back a SUN short of it on
// about a quarter of networks.
func TestConvertToStakedIsTheExactShare(t *testing.T) {
	t.Parallel()
	c := &Client{}
	rng := rand.New(rand.NewPCG(3, 4))
	for range 20_000 {
		limit, weight, n := 1e9+rng.Int64N(2e11), 1e6+rng.Int64N(5e10), 1+rng.Int64N(200_000)
		num := new(big.Int).Mul(big.NewInt(n), big.NewInt(weight))
		num.Mul(num, big.NewInt(units.SunPerTRX))
		want := new(big.Int).Div(new(big.Int).Add(num, big.NewInt(limit-1)), big.NewInt(limit)).Int64()
		if got := c.ConvertEnergyToStaked(limit, weight, decimal.NewFromInt(n)); int64(got) != want {
			t.Fatalf("ConvertEnergyToStaked(%d, %d, %d) = %d, want %d", limit, weight, n, got, want)
		}
		if got := c.ConvertBandwidthToStaked(weight, limit, decimal.NewFromInt(n)); int64(got) != want {
			t.Fatalf("ConvertBandwidthToStaked(%d, %d, %d) = %d, want %d", weight, limit, n, got, want)
		}
	}
	// A fractional amount rounds up too.
	if got := c.ConvertEnergyToStaked(3, 1, decimal.RequireFromString("0.5")); got != 166_667 {
		t.Errorf("ConvertEnergyToStaked of half a unit = %d, want 166667", got)
	}
}
