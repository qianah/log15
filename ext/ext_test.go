package ext

import (
	log "github.com/inconshreveable/log15"
	"math"
	"testing"
)

type testHandler struct {
    r log.Record
}
func (h *testHandler) Log(r *log.Record) error {
    h.r = *r
    return nil
}

func TestHotSwapHandler(t *testing.T) {
	t.Parallel()

	h1 := &testHandler{}

	l := log.New()
	h := HotSwapHandler(h1)
	l.SetHandler(h)

	l.Info("to h1")
	if h1.r.Msg != "to h1" {
		t.Fatalf("didn't get expected message to h1")
	}

	h2 := &testHandler{}
	h.Swap(h2)
	l.Info("to h2")
	if h2.r.Msg != "to h2" {
		t.Fatalf("didn't get expected message to h2")
	}
}

func TestSpeculativeHandler(t *testing.T) {
	t.Parallel()

	// test with an even multiple of the buffer size, less than full buffer size
	// and not a multiple of the buffer size
	for _, count := range []int{10000, 50, 432} {
		recs := make(chan *log.Record)
		done := make(chan int)
		spec := SpeculativeHandler(100, log.ChannelHandler(recs))

		go func() {
			defer close(done)
			expectedCount := int(math.Min(float64(count), float64(100)))
			expectedIdx := count - expectedCount
			for r := range recs {
				if r.Ctx[1] != expectedIdx {
					t.Errorf("Bad ctx 'i', got %d expected %d", r.Ctx[1], expectedIdx)
					return
				}
				expectedIdx++
				expectedCount--

				if expectedCount == 0 {
					// got everything we expected
					break
				}
			}

			select {
			case <-recs:
				t.Errorf("got an extra record we shouldn't have!")
			default:
			}
		}()

		lg := log.New()
		lg.SetHandler(spec)
		for i := 0; i < count; i++ {
			lg.Debug("test speculative", "i", i)
		}

		go spec.Flush()

		// wait for the go routine to finish
		<-done
	}
}
