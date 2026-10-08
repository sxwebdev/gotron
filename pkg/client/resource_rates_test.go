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

// The fewest units whose least stake is a delegation's minimum, on every
// network shape and in both modes: their StakeFor reaches MinDelegateBalance
// and a unit fewer stakes under it.
func TestResourceRatesMinDelegateUnits(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewPCG(5, 6))
	for range 20_000 {
		// Up to a few hundred units per TRX, and down to none, so the
		// minimum lands both past one unit and at it.
		r := ResourceRates{
			TotalEnergyCurrentLimit: 1 + rng.Int64N(2e11),
			TotalEnergyWeight:       1e6 + rng.Int64N(5e10),
			TotalNetLimit:           1 + rng.Int64N(5e10),
			TotalNetWeight:          1e6 + rng.Int64N(8e10),
			Harden:                  rng.IntN(2) == 0,
		}
		for _, resource := range []ResourceType{ResourceTypeEnergy, ResourceTypeBandwidth} {
			n := r.MinDelegateUnits(resource)
			if n < 1 || r.StakeFor(resource, n) < MinDelegateBalance || (n > 1 && r.StakeFor(resource, n-1) >= MinDelegateBalance) {
				t.Fatalf("%+v %v: MinDelegateUnits = %d, StakeFor of it %d, of a unit fewer %d", r, resource, n, r.StakeFor(resource, n), r.StakeFor(resource, n-1))
			}
			// The minimum lent instead covers every count under it.
			if r.Limit(resource, MinDelegateBalance) < n-1 {
				t.Fatalf("%+v %v: the minimum gives %d units, under the %d below MinDelegateUnits", r, resource, r.Limit(resource, MinDelegateBalance), n-1)
			}
		}
	}
}

// At a limit of 3 per 999 999 TRX of weight, a SUN under a TRX is exactly 3
// units. Hardened, the chain counts 3, so 3 units stake under the minimum and 4
// is the fewest that reach it; in doubles it counts 2, and 3 units already
// take a whole TRX.
func TestResourceRatesMinDelegateUnitsModes(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		harden bool
		want   int64
	}{{false, 3}, {true, 4}} {
		r := ResourceRates{
			TotalEnergyCurrentLimit: 3_000_000, TotalEnergyWeight: 999_999,
			TotalNetLimit: 3_000_000, TotalNetWeight: 999_999,
			Harden: tc.harden,
		}
		for _, resource := range []ResourceType{ResourceTypeEnergy, ResourceTypeBandwidth} {
			if got := r.MinDelegateUnits(resource); got != tc.want {
				t.Errorf("harden %t %v: MinDelegateUnits = %d, want %d", tc.harden, resource, got, tc.want)
			}
			if s := r.StakeFor(resource, tc.want-1); s >= MinDelegateBalance {
				t.Errorf("harden %t %v: %d units stake %d, not under the minimum", tc.harden, resource, tc.want-1, s)
			}
		}
	}
}

// Mainnet-shaped totals, worked by hand. Energy: 180e9 over 19e9 TRX is
// 9.47 a TRX, so 9 units stake 0.95 TRX and 10 stake 1.055556. Bandwidth:
// 43.2e9 over 3e6 TRX is exactly 14 400 a TRX, which a SUN less falls short of.
func TestResourceRatesMinDelegateUnitsMainnet(t *testing.T) {
	t.Parallel()
	for _, harden := range []bool{false, true} {
		r := ResourceRates{
			TotalEnergyCurrentLimit: 180_000_000_000, TotalEnergyWeight: 19_000_000_000,
			TotalNetLimit: 43_200_000_000, TotalNetWeight: 3_000_000,
			Harden: harden,
		}
		if got := r.MinDelegateUnits(ResourceTypeEnergy); got != 10 {
			t.Errorf("harden %t: energy MinDelegateUnits = %d, want 10", harden, got)
		}
		if got := r.MinDelegateUnits(ResourceTypeBandwidth); got != 14_400 {
			t.Errorf("harden %t: bandwidth MinDelegateUnits = %d, want 14400", harden, got)
		}
	}
}

func TestResourceRatesMinDelegateUnitsEdges(t *testing.T) {
	t.Parallel()
	if MinDelegateBalance != 1_000_000 {
		t.Errorf("MinDelegateBalance = %d, want java-tron's TRX_PRECISION, 1000000", MinDelegateBalance)
	}
	// No totals: no stake gives a unit, so one unit already needs more than
	// the minimum.
	for _, harden := range []bool{false, true} {
		if got := (ResourceRates{Harden: harden}).MinDelegateUnits(ResourceTypeEnergy); got != 1 {
			t.Errorf("harden %t: MinDelegateUnits with no totals = %d, want 1", harden, got)
		}
	}
	// Totals no node reports still give a count a delegation can be sized by.
	for _, harden := range []bool{false, true} {
		r := ResourceRates{TotalEnergyCurrentLimit: -180_000_000_000, TotalEnergyWeight: 19_000_000_000, Harden: harden}
		if got := r.MinDelegateUnits(ResourceTypeEnergy); got != 1 {
			t.Errorf("harden %t: MinDelegateUnits with a negative limit = %d, want 1", harden, got)
		}
	}
	// The largest limit int64 holds: a stake under the minimum gives under
	// it, so one past that figure stays in range and is still the fewest.
	for _, harden := range []bool{false, true} {
		r := ResourceRates{TotalNetLimit: math.MaxInt64, TotalNetWeight: 1, Harden: harden}
		n := r.MinDelegateUnits(ResourceTypeBandwidth)
		if n < 1 || r.StakeFor(ResourceTypeBandwidth, n) < MinDelegateBalance || r.StakeFor(ResourceTypeBandwidth, n-1) >= MinDelegateBalance {
			t.Errorf("harden %t: MinDelegateUnits at the largest limit = %d, StakeFor of it %d, of a unit fewer %d", harden, n, r.StakeFor(ResourceTypeBandwidth, n), r.StakeFor(ResourceTypeBandwidth, n-1))
		}
	}
}
