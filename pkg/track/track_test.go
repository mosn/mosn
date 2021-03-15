package track

import (
	"regexp"
	"testing"
	"time"
)

type phaseCase struct {
	last    TrackPhase
	current TrackPhase
}

// expected 8 [], datas can exists in []
const outreg = `^(\[[^(\[\])]*?\]){8}$`

var outexp = regexp.MustCompile(outreg)

func TestTrack(t *testing.T) {
	track := Tracks{}
	testcases := []phaseCase{
		{NoTrack, ProtocolDecode},                            // request receievd, record only
		{ProtocolDecode, StreamFilterBeforeRoute},            // protocol decode costs
		{StreamFilterBeforeRoute, NoTrack},                   // filter stage 1 costs
		{NoTrack, MatchRoute},                                // macth route record only
		{MatchRoute, StreamFilterAfterRoute},                 // match route costs
		{StreamFilterAfterRoute, NoTrack},                    // filter stage 2 costs
		{NoTrack, LoadBalanceChooseHost},                     // lb record
		{LoadBalanceChooseHost, StreamFilterAfterChooseHost}, // lb costs
		{StreamFilterAfterChooseHost, NetworkDataWrite},      // filter stage 3 costs
		{NetworkDataWrite, NoTrack},                          // write data costs
		{NoTrack, NoTrack},                                   // invalid pass
		{MaxTrackPhase + 1, NoTrack},                         // invalid pass
		{NoTrack, MaxTrackPhase + 1},                         // invalid pass
		{NoTrack, ProtocolDecode},                            // response received, record only
		{ProtocolDecode, StreamSendFilter},                   // protocol decode costs
		{StreamSendFilter, NetworkDataWrite},                 // filter stage 4 costs
		{NetworkDataWrite, NoTrack},                          // write data costs
	}
	interval := 100 * time.Millisecond // 100ms
	for _, tc := range testcases {
		track.EndTrack(tc.last)
		track.StartTrack(tc.current)
		time.Sleep(interval)
	}
	s := track.GetTrackCosts()
	if !outexp.MatchString(s) {
		t.Fatalf("unexpected output: %s", s)
	}
	t.Logf("output is %s", s)
}

func TestTrack2(t *testing.T) {
	track := Tracks{}
	// no protocol decode setted
	track.EndTrack(ProtocolDecode)
	track.StartTrack(StreamSendFilter)
	track.EndTrack(StreamSendFilter)
	track.StartTrack(MaxServedField + 1)
	track.EndTrack(MaxServedField + 1)
	//
	track.Range(func(phase TrackPhase, tk TrackTime) bool {
		switch phase {
		case ProtocolDecode:
			if len(tk.Costs) > 0 {
				t.Fatalf("unexpected costs: %v", tk.Costs)
			}
		case StreamSendFilter, MaxServedField + 1:
			if len(tk.Costs) != 1 {
				t.Fatalf("unexpected costs: %v", tk.Costs)
			}
		}
		return true
	})
	// extends will not output in default get
	s := track.GetTrackCosts()
	if !outexp.MatchString(s) {
		t.Fatalf("unexpected output: %s", s)
	}
	t.Logf("output is %s", s)
}

func BenchmarkTrack(b *testing.B) {
	for i := 0; i < b.N; i++ {
		track := &Tracks{}
		for _, p := range []TrackPhase{
			ProtocolDecode, StreamFilterBeforeRoute, MatchRoute, StreamFilterAfterRoute, LoadBalanceChooseHost,
			StreamFilterAfterChooseHost, NetworkDataWrite, ProtocolDecode, StreamSendFilter, NetworkDataWrite,
		} {

			track.StartTrack(p)
			track.EndTrack(p)
		}
	}
}
