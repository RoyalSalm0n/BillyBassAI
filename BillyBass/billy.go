package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-audio/wav"
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/host/v3"
)

// Get server IP AND API KEY from local ENV variable
var server string = os.Getenv("server")
var auth string = os.Getenv("API_KEY")

// predefine motor gpios
var (
	head1  gpio.PinIO
	head2  gpio.PinIO
	mouth1 gpio.PinIO
	mouth2 gpio.PinIO
	tail1  gpio.PinIO
	tail2  gpio.PinIO
)

const outputPath = "/home/pi/billybass/BillyBass/output.wav"

var client = &http.Client{
	Timeout: 90 * time.Second,
}

type mouthFrame struct {
	open bool
}

// record the audio being asked by billy
func recordAudio() error {

	//kill any possible procesess that could interfere with sox
	cmd := exec.Command("pkill", "-f", "arecord")
	cmd.Run()
	cmd = exec.Command("pulseaudio", "-k")
	cmd.Run()

	time.Sleep(300 * time.Millisecond)

	fmt.Println("Recording Audio")

	//record, when audio above 1% is detected for 0.1s and stop recording when audio is below 10% for 2s
	cmd = exec.Command("sox", "-t", "alsa", "plughw:CARD=Microphone", "input.wav", "gain", "+1", "silence", "1", "0.1", "1%", "1", "2.0", "10%")
	var stdout, stderr bytes.Buffer
	//capture standard out and standard err
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("recordAudio: %s: %w", stderr.String(), err)
	}
	return nil
}

// helper fucnction to get pin on raspberry pi
func mustGetPin(name string) (gpio.PinIO, error) {
	pin := gpioreg.ByName(name)
	if pin == nil {
		return nil, fmt.Errorf("GPIO pin %s not found", name)
	}
	return pin, nil
}

// stop all motors
func stopMotors() {
	if err := mouth1.Out(gpio.Low); err != nil {
		fmt.Println("Cant find mouth1")
	}
	if err := mouth2.Out(gpio.Low); err != nil {
		fmt.Println("Cant find mouth2")
	}
	if err := head1.Out(gpio.Low); err != nil {
		fmt.Println("Cant find head1")
	}
	if err := head2.Out(gpio.Low); err != nil {
		fmt.Println("Cant find head2")
	}
	if err := tail1.Out(gpio.Low); err != nil {
		fmt.Println("Cant find tail1")
	}
	if err := tail2.Out(gpio.Low); err != nil {
		fmt.Println("Cant find tail2")
	}
}

// play audio using aplay
func playaudio(done chan struct{}, file string, wg *sync.WaitGroup) error {
	// Use a WaitGroup to signal when the goroutine is done and close the done channel to stop the mouth and tail movement
	defer wg.Done()
	defer close(done)

	fmt.Println("aplay starting at", time.Now())
	fmt.Println("playing audio")
	cmd := exec.Command("aplay", "-D", "plughw:0,0", file)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("playaudio: %s: %w", stderr.String(), err)
	}
	return nil
}

// generate prompt from audio file using server API endpoint
func generatePrompt(audioPath string) (string, error) {
	transcribeUrl := server + "/transcribe"
	file, err := os.Open(audioPath)
	if err != nil {
		return "", fmt.Errorf("generatePrompt: %w", err)
	}
	defer file.Close()

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)
	part, err := writer.CreateFormFile("audio", audioPath)
	if err != nil {
		return "", fmt.Errorf("generatePrompt: %w", err)
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return "", fmt.Errorf("generatePrompt: %w", err)
	}
	writer.Close()
	req, err := http.NewRequest("POST", transcribeUrl, &requestBody)
	if err != nil {
		return "", fmt.Errorf("generatePrompt: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+auth)
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("generatePrompt: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("generatePrompt: server returned status %d", resp.StatusCode)
	}
	prompt, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("generatePrompt: %w", err)
	}
	return string(prompt), nil
}

// generate a response from the prompt using the AI API
func generateResponse(prompt string) (string, error) {
	url := server + "/ai"
	payload, err := json.Marshal(map[string]string{"prompt": prompt})
	if err != nil {
		return "", fmt.Errorf("generateResponse: %w", err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return "", fmt.Errorf("generateResponse: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+auth)
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("generateResponse: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("generateResponse: server returned status %d", resp.StatusCode)
	}
	response, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("generateResponse: %w", err)
	}
	return string(response), nil

}

// convert text to speech using the TTS API endpoint
func tts(response string) error {
	url := server + "/tts"
	payload, err := json.Marshal(map[string]string{"text": response})
	if err != nil {
		return fmt.Errorf("tts: %w", err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("tts: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+auth)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("tts: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("tts: server returned status %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	output, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("tts: %w", err)
	}
	defer output.Close()
	if _, err := io.Copy(output, resp.Body); err != nil {
		return fmt.Errorf("tts: %w", err)
	}
	return nil
}

func moveHeadOut() error {
	if head1 == nil {
		return fmt.Errorf("failed to find GPIO17")
	}
	if head2 == nil {
		return fmt.Errorf("failed to find GPIO22")
	}
	if err := head1.Out(gpio.Low); err != nil {
		return fmt.Errorf("moveHeadOut: %w", err)
	}
	if err := head2.Out(gpio.Low); err != nil {
		return fmt.Errorf("moveHeadOut: %w", err)
	}
	time.Sleep(500 * time.Millisecond)
	fmt.Println("moving head")
	if err := head1.Out(gpio.High); err != nil {
		return fmt.Errorf("moveHeadOut: %w", err)
	}
	if err := head2.Out(gpio.Low); err != nil {
		return fmt.Errorf("moveHeadOut: %w", err)
	}
	return nil
}

func moveHeadIn() error {
	if head1 == nil {
		return fmt.Errorf("failed to find GPIO17")
	}
	if head2 == nil {
		return fmt.Errorf("failed to find GPIO22")
	}
	if err := head1.Out(gpio.Low); err != nil {
		return fmt.Errorf("moveHeadIn: %w", err)
	}
	if err := head2.Out(gpio.Low); err != nil {
		return fmt.Errorf("moveHeadIn: %w", err)
	}
	time.Sleep(50 * time.Millisecond)
	return nil
}

func computeMouthTineline(file string) ([]mouthFrame, int, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, 0, fmt.Errorf("computeMouthTineline: %w", err)
	}
	defer f.Close()
	d := wav.NewDecoder(f)
	buf, err := d.FullPCMBuffer()
	if err != nil {
		return nil, 0, fmt.Errorf("computeMouthTineline: %w", err)
	}
	sampleRate := buf.Format.SampleRate
	numChannels := buf.Format.NumChannels
	windowMS := 50
	windowSamples := (sampleRate * windowMS / 1000) * numChannels
	var timeline []mouthFrame
	if windowSamples <= 0 {
		return nil, 0, fmt.Errorf("computeMouthTineline: invalid window size")
	}
	smoothedRMS := 0.0
	attack := 0.4
	release := 0.1
	for i := 0; i < len(buf.Data); i += windowSamples {
		end := i + windowSamples
		if end > len(buf.Data) {
			end = len(buf.Data)
		}
		var sumSQ float64
		for _, s := range buf.Data[i:end] {
			v := float64(s) / float64(1<<15)
			sumSQ += v * v
		}
		rms := math.Sqrt(sumSQ / float64(end-i))
		if rms > smoothedRMS {
			smoothedRMS = attack*rms + (1-attack)*smoothedRMS
		} else {
			smoothedRMS = release*rms + (1-release)*smoothedRMS
		}
		threshold := 0.1
		timeline = append(timeline, mouthFrame{open: smoothedRMS > threshold})
	}
	return timeline, windowMS, nil
}

func syncMouth(done chan struct{}, windowMS int, wg *sync.WaitGroup, timeline []mouthFrame) error {
	defer wg.Done()
	if mouth1 == nil {
		return fmt.Errorf("failed to find GPIO23")
	}
	if mouth2 == nil {
		return fmt.Errorf("failed to find GPIO24")
	}
	fmt.Println("mouth timeline starting playback at", time.Now())
	start := time.Now()
	for {
		select {
		case _, ok := <-done:
			if !ok {
				mouth1.Out(gpio.Low)
				mouth2.Out(gpio.Low)
				return nil
			}
		default:
			elapsed := time.Since(start)
			idx := int(elapsed.Milliseconds()) / windowMS
			if idx >= len(timeline) {
				mouth1.Out(gpio.Low)
				mouth2.Out(gpio.Low)
				return nil
			}
			if timeline[idx].open {
				if err := mouth1.Out(gpio.High); err != nil {
					return fmt.Errorf("moveMouth: %w", err)
				}
				if err := mouth2.Out(gpio.Low); err != nil {
					return fmt.Errorf("moveMouth: %w", err)
				}
			} else {
				if err := mouth1.Out(gpio.Low); err != nil {
					return fmt.Errorf("moveMouth: %w", err)
				}
				if err := mouth2.Out(gpio.High); err != nil {
					return fmt.Errorf("moveMouth: %w", err)
				}
			}
			fmt.Printf("Window %d: Mouth open: %v\n", idx, timeline[idx].open)
			time.Sleep(time.Duration(windowMS) * time.Millisecond)
		}
	}

}

func moveMouth(done chan struct{}, wg *sync.WaitGroup) error {
	defer wg.Done()
	for {
		select {
		case _, ok := <-done:
			if !ok {
				if err := mouth1.Out(gpio.Low); err != nil {
					return fmt.Errorf("moveMouth: %w", err)
				}
				return nil
			}
		default:
			if mouth1 == nil {
				return fmt.Errorf("failed to find GPIO23")
			}
			if mouth2 == nil {
				return fmt.Errorf("failed to find GPIO24")
			}
			if err := mouth1.Out(gpio.High); err != nil {
				return fmt.Errorf("moveMouth: %w", err)
			}
			if err := mouth2.Out(gpio.Low); err != nil {
				return fmt.Errorf("moveMouth: %w", err)
			}
			time.Sleep(200 * time.Millisecond)
			if err := mouth1.Out(gpio.Low); err != nil {
				return fmt.Errorf("moveMouth: %w", err)
			}
			if err := mouth2.Out(gpio.High); err != nil {
				return fmt.Errorf("moveMouth: %w", err)
			}
			time.Sleep(200 * time.Millisecond)

		}
	}
}

func moveTail(done chan struct{}, wg *sync.WaitGroup) error {
	defer wg.Done()
	for {
		select {
		case _, ok := <-done:
			if !ok {
				if err := tail1.Out(gpio.Low); err != nil {
					return fmt.Errorf("moveTail: %w", err)
				}
				return nil
			}
		default:
			if tail1 == nil {
				return fmt.Errorf("failed to find GPIO5")
			}
			if tail2 == nil {
				return fmt.Errorf("failed to find GPIO6")
			}
			if err := tail1.Out(gpio.High); err != nil {
				return fmt.Errorf("moveTail: %w", err)
			}
			if err := tail2.Out(gpio.Low); err != nil {
				return fmt.Errorf("moveTail: %w", err)
			}
			time.Sleep(200 * time.Millisecond)
			if err := tail1.Out(gpio.Low); err != nil {
				return fmt.Errorf("moveTail: %w", err)
			}
			if err := tail2.Out(gpio.High); err != nil {
				return fmt.Errorf("moveTail: %w", err)
			}
			time.Sleep(800 * time.Millisecond)

		}
	}

}

func main() {
	var wg sync.WaitGroup
	_, err := host.Init()
	if err != nil {
		log.Fatal(err)
	}
	head1, err = mustGetPin("17")
	if err != nil {
		fmt.Println("failed to find GPIO17")
		os.Exit(1)
	}
	head2, err = mustGetPin("22")
	if err != nil {
		fmt.Println("failed to find GPIO22")
		os.Exit(1)
	}
	mouth1, err = mustGetPin("23")
	if err != nil {
		fmt.Println("failed to find GPIO23")
		os.Exit(1)
	}
	mouth2, err = mustGetPin("24")
	if err != nil {
		fmt.Println("failed to find GPIO24")
		os.Exit(1)
	}
	tail1, err = mustGetPin("5")
	if err != nil {
		fmt.Println("failed to find GPIO5")
		os.Exit(1)
	}
	tail2, err = mustGetPin("6")
	if err != nil {
		fmt.Println("failed to find GPIO6")
		os.Exit(1)
	}
	if auth == "" {
		fmt.Println("API_KEY environment variable not set")
		os.Exit(1)
	}
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigs
		fmt.Println("\nCaught interrupt! Stopping motors and exiting...")
		stopMotors()
		os.Exit(0)
	}()

	defer stopMotors()
	if err := moveHeadOut(); err != nil {
		fmt.Println("failed to move head out:", err)
		stopMotors()
		os.Exit(1)
	}
	fmt.Print("moving head\n")
	if err := recordAudio(); err != nil {
		fmt.Println("failed to record audio:", err)
		stopMotors()
		os.Exit(1)
	}
	if err := moveHeadIn(); err != nil {
		fmt.Println("failed to move head in:", err)
		stopMotors()
		os.Exit(1)
	}
	prompt, err := generatePrompt("input.wav")
	if err != nil {
		fmt.Println("Failed to generate prompt:", err)
		os.Exit(1)
	}
	fmt.Println(prompt)
	response, err := generateResponse(prompt)
	if err != nil {
		fmt.Println("Failed to generate response:", err)
		os.Exit(1)
	}
	fmt.Println(response)
	if err := tts(response); err != nil {
		fmt.Println("Failed to generate speech:", err)
		os.Exit(1)
	}
	timeline, windowMS, err := computeMouthTineline(outputPath)
	if err != nil {
		fmt.Println("Failed to compute mouth timeline:", err)
		os.Exit(1)
	}
	done := make(chan struct{})
	errCh := make(chan error, 3)
	wg.Add(3)
	go func() {
		errCh <- playaudio(done, outputPath, &wg)
	}()
	go func() {
		errCh <- syncMouth(done, windowMS, &wg, timeline)
	}()
	go func() {
		errCh <- moveTail(done, &wg)
	}()
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			fmt.Println("Error:", err)
		}
	}
	if err := tail1.Out(gpio.Low); err != nil {
		fmt.Println("Failed to set tail1 low:", err)
		stopMotors()
		os.Exit(1)
	}
	if err := tail2.Out(gpio.Low); err != nil {
		fmt.Println("Failed to set tail2 low:", err)
		stopMotors()
		os.Exit(1)
	}

}
