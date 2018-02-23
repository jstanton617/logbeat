package main

import (
	"fmt"
	"github.com/gordonklaus/portaudio"
	"time"
	"math"
	"sync/atomic"
	"bufio"
	"os"
	"regexp"
	"github.com/bmizerany/perks/quantile"
	"strconv"
	"io"
	"github.com/cznic/mathutil"
)

const sampleRate = 44100

var bpm int64
var freq int64
var lastPlayed time.Duration

func main() {
	portaudio.Initialize()
	defer portaudio.Terminate();
	bpm = 30
	freq = 450

	quit := make(chan bool)
	go play(quit);
	re, _ := regexp.Compile("php_time=(.*?) ")
	stdin := read(os.Stdin)

	q := quantile.NewTargeted(0.90)
	tick := time.NewTicker(15 * time.Second)

	for {
		select {
		case <-tick.C:
			ninety := q.Query(0.90)
			throughput := q.Count() / 15
			fmt.Printf("90: %v Throughput: %v\n", ninety, throughput)
			atomic.StoreInt64(&freq, mathutil.ClampInt64(int64(ninety*1000),200, 10000))
			atomic.StoreInt64(&bpm, mathutil.ClampInt64(int64(throughput), 5, 480))

			q.Reset()
		case logline := <-stdin:
			match := re.FindStringSubmatch(logline)
			phpTime, _ := strconv.ParseFloat(match[1], 64)
			q.Insert(phpTime)
		}
	}

	quit <- true

	//Wait for shutdown
	time.Sleep(1 * time.Second)

	fmt.Println("Stopped");
}

func read(r io.Reader) <-chan string {
	lines := make(chan string)
	go func() {
		defer close(lines)
		scan := bufio.NewScanner(r)
		for scan.Scan() {
			lines <- scan.Text()
		}
	}()
	return lines
}

func play(quit chan bool) {
	s := newSine()
	defer s.Close()

	chk(s.Start())
	<-quit
	chk(s.Stop())
}

type sine struct {
	*portaudio.Stream
	phase float64
}

func newSine() *sine {
	s := &sine{nil, 0}
	var err error
	s.Stream, err = portaudio.OpenDefaultStream(0, 1, sampleRate, 0, s.processAudio)
	chk(err)
	return s
}

func (g *sine) processAudio(out []float32, timeInfo portaudio.StreamCallbackTimeInfo) {

	if lastPlayed == time.Second*0 {
		lastPlayed = timeInfo.OutputBufferDacTime
	}

	localBpm := atomic.LoadInt64(&bpm)
	localFreq := atomic.LoadInt64(&freq)

	interval := 1.0 / (float64(localBpm) / 60.0) * 1000
	intervalDuration := time.Duration(interval) * time.Millisecond

	if lastPlayed+intervalDuration < timeInfo.OutputBufferDacTime {
		for i := range out {
			out[i] = float32(math.Sin(2 * math.Pi * g.phase))
			_, g.phase = math.Modf(g.phase + (float64(localFreq) / sampleRate))
		}

		lastPlayed = timeInfo.OutputBufferDacTime
	} else {
		for i := range out {
			out[i] = 0.0
		}
	}
}

func chk(err error) {
	if err != nil {
		panic(err)
	}
}
