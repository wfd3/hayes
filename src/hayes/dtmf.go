package main

// Generate DTMF tones and fake out modem carrier sounds.
//
// https://en.wikipedia.org/wiki/Precise_tone_plan
// https://en.wikipedia.org/wiki/Modem#Carterfone_and_direct_connection
//
// Requires libasound2-dev

import (
	"math"
	"time"
	"github.com/hajimehoshi/oto"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Code to handle sine waves
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
const sampleRate int = 44100 
const channelNum int = 1
const bitDepthInBytes int = 2

var oto_context *oto.Context
var volume float64 = 0.5

type sineWave struct {
	freq float64
	stop chan bool
	stopped chan bool
	playing bool
}

func setVolume(v int) {
	switch {
	case v < 0: v = 0
	case v > 100: v = 100
	}
	volume = float64(v) / 100
}

func getVolume() int {
	return int(volume * 100)
}

func NewSineWave(freq float64) *sineWave {
	if oto_context == nil {
		var err error
		oto_context, err = oto.NewContext(sampleRate, channelNum, bitDepthInBytes, 4096)
		if err != nil {
			panic(err)
		}
	}
	
	c := make(chan bool, 0)
	d := make(chan bool, 0)

	return &sineWave{
		freq: freq,
		stop: c,
		stopped: d,
		playing: false,
	}
}

func (s *sineWave) Play() {
	if s.playing {
		return
	}
	s.playing = true
	p := oto_context.NewPlayer()

	var buf []byte

	phase := float64(0.0)
	radians := s.freq * 2 * math.Pi / float64(sampleRate)
	for { 
		phase += radians
		t := math.Sin(phase) * volume
		select {
		case <- s.stop:
			s.playing = false
			s.stopped <- true
			p.Close()
			return	// Must return here.
		default:
			v := uint16(t * 32000)
			buf = []byte{uint8(v & 0xff), uint8(v >> 8)}
			_, err := p.Write(buf)
			if err != nil {
				panic(err)
			}
		}
	}
}

func (s *sineWave) Stop() {
	if s.playing {
		s.stop <- true
		<- s.stopped
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Code to handle playing one or more tones, either in foreground or background
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type tone struct {
	waves map[float64]*sineWave
}

func NewTone(freqs ...float64) *tone {
	var t tone
	t.waves = make(map[float64]*sineWave)
	for _, freq := range freqs {
		t.waves[freq] = NewSineWave(freq)
	}
	return &t
}

func (t *tone) StopFreq(freq float64) {
	if !flags.sound {
		return
	}

	wave, exists := t.waves[freq]
	if exists {
		wave.Stop()
	}
}

func (t *tone) Stop() {
	if !flags.sound {
		return
	}

	for _, wave := range t.waves {
		wave.Stop()
	}
}

func (t *tone) BackgroundPlay() {
	if !flags.sound {
		return
	}

	for _, wave := range t.waves {
		go wave.Play()
	}
}

func (t *tone) Play(duration time.Duration) {
	if !flags.sound {
		return
	}

	t.BackgroundPlay()
	time.Sleep(duration)
	t.Stop()
}

func (t *tone)AddFreq(freq float64) {
	if !flags.sound {
		return
	}

	if _, exists := t.waves[freq]; !exists {
		t.waves[freq] = NewSineWave(freq)
	}
	go t.waves[freq].Play()
}

func getKeyTones(key rune) *tone {
	switch key {
	case '1': return NewTone(1209.0, 697.0)
	case '2': return NewTone(1336.0, 697.0)
	case '3': return NewTone(1447.0, 697.0)
	case 'A': return NewTone(1633.0, 697.0)
	case '4': return NewTone(1209.0, 770.0)
	case '5': return NewTone(1336.0, 770.0)
	case '6': return NewTone(1447.0, 770.0)
	case 'B': return NewTone(1633.0, 770.0)
	case '7': return NewTone(1209.0, 852.0)
	case '8': return NewTone(1336.0, 852.0)
	case '9': return NewTone(1447.0, 852.0)
	case 'C': return NewTone(1633.0, 852.0)
	case '*': return NewTone(1209.0, 941.0)
	case '0': return NewTone(1336.0, 941.0)
	case '#': return NewTone(1447.0, 941.0)
	case 'D': return NewTone(1633.0, 941.0)
	default: logger.Printf("Unknown DTMF key: %c", key)
	}
	return nil
}

var DialTone = NewTone(350.0, 440.0)
var RingTone = NewTone(440.0, 480.0)
var BusyTone = NewTone(480.0, 620.0)

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Code to generate DTMF and carrier frequencies
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// Not exactly timed but close enough
func carrierTone(duration time.Duration) {
	if !flags.sound {
		return
	}

	t := NewTone(0, 2100.0)
	start := time.Now()
	t.BackgroundPlay()
	v := getVolume()
	for time.Now().Sub(start) < duration {
		setVolume(v)
		time.Sleep(time.Second)
		
		t.AddFreq(1600.0)
		t.AddFreq(1800.0)
		time.Sleep(400 * time.Millisecond)
		
		t.AddFreq(1646.0)
		t.AddFreq(1829.0)
		time.Sleep(400 * time.Millisecond)

		t.AddFreq(1680.0)
		t.AddFreq(1876.0)
		time.Sleep(200 * time.Millisecond)
		
		t.Stop()
		setVolume(v * 2)
		t.Play(500 * time.Millisecond)
	}
	t.Stop()

	setVolume(v)
}

func ringTone(count int) {
	if !flags.sound {
		return
	}

	for i := 0; i < count; i++ {
		RingTone.Play(2 * time.Second)
		time.Sleep(4 * time.Second)
	}
}

func busyTone(count int) {
	if !flags.sound {
		return
	}

	for i:= 0; i < count; i++ {
		BusyTone.Play(500 * time.Millisecond)
		time.Sleep(500 * time.Millisecond)
	}
}

func dialSounds(s string, keypressDelay, interkeyDelay time.Duration) {
	if !flags.sound {
		return 
	}

	for _, key := range s {
		if key == ',' {
			time.Sleep(2*time.Second) // TODO: I think there's a register for , delay time
			continue
		}

		t := getKeyTones(key)
		t.Play(keypressDelay)
		time.Sleep(interkeyDelay)
	}
}

func simulateDTMF(s string) {
	if !flags.sound {
		return
	}
	DialTone.Play(250 * time.Millisecond)
	dialSounds(s, 150 * time.Millisecond, 50 * time.Millisecond)
}
