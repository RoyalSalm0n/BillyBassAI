package main

import (
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/host/v3"
	"log"
	"time"
	"os/exec"
	"fmt"
	"bytes"
	"os"
	"encoding/json"
	"io"
	"syscall"
	"os/signal"
	"sync"
	"net/http"
	"mime/multipart"
	//github.com/gordonklaus/portaudio
)

var (
	head1  gpio.PinIO
	head2  gpio.PinIO
	mouth1 gpio.PinIO
	mouth2 gpio.PinIO
	tail1  gpio.PinIO
	tail2  gpio.PinIO
)


func recordAudio() {
	cmd := exec.Command("pkill", "-f", "arecord")
	cmd.Run()
	cmd = exec.Command("pulseaudio","-k")
	cmd.Run()
	cmd = exec.Command("pkill", "-f", "porcupine")
	cmd.Run()
	time.Sleep(300 * time.Millisecond)
	fmt.Println("Recording Audio")
	cmd = exec.Command("sox", "-t","alsa", "plughw:2,0", "input.wav", "gain", "+20", "silence", "1", "0.1", "1%", "1", "2.0", "4%")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		fmt.Println(stderr.String())
		stopMotors()
		log.Fatal(err)
	}
}


func mustGetPin(name string) gpio.PinIO {
	pin := gpioreg.ByName(name)
	if pin == nil {
		log.Fatalf("GPIO pin %s not found", name)
	}
	return pin
}

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

func playaudio(done chan struct{},file string,wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Println("playing audio")
	cmd := exec.Command("aplay","-D","plughw:0,0",file)
	var stdout, stderr bytes.Buffer
        cmd.Stdout = &stdout
        cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		fmt.Println(stderr.String())
		stopMotors()
		log.Fatal(err)
	}
	defer close(done)
}

func generatePrompt (audio_path string) string {
	transcribeUrl := "http://***REMOVED***:5000/transcribe"
	file,err :=os.Open(audio_path)
	if err != nil {
		stopMotors()
		log.Fatal(err)
	}
	defer file.Close()
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)
	part, err := writer.CreateFormFile("audio",audio_path)
	if err != nil {
		stopMotors()
		log.Fatal(err)
	}
	_,err = io.Copy(part,file)
	if err != nil {
		stopMotors()
		log.Fatal(err)
	}
	writer.Close()
	req, err  := http.NewRequest("POST", transcribeUrl, &requestBody)
	if err != nil {
		stopMotors()
		log.Fatal(err)
	}
	req.Header.Set("Content-Type",writer.FormDataContentType())
	client :=&http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		stopMotors()
		log.Fatal(err)
	}
	defer resp.Body.Close()
	prompt, err := io.ReadAll(resp.Body)
	return string(prompt)
}

func generateResponse(prompt string) string {
	url := "http://***REMOVED***:5000/ai"
	payload,err := json.Marshal(map[string]string{"prompt":prompt,})
	if err != nil {
		stopMotors()
		log.Fatal(err)
	}
	req,err :=http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil { 
		stopMotors()
		log.Fatal(err)
	}
	req.Header.Set("Content-Type","application/json")

	client := &http.Client{}
	resp,err := client.Do(req)
	if err != nil{
		stopMotors()
		log.Fatal(err)
	}
	defer resp.Body.Close()
	response, err := io.ReadAll(resp.Body)
	return string(response)

}

func tts(response string) {
	url := "http://***REMOVED***:5000/tts"
        payload,err := json.Marshal(map[string]string{"text":response,})
        if err != nil {
                stopMotors()
                log.Fatal(err)
        }
        req,err :=http.NewRequest("POST", url, bytes.NewBuffer(payload))
        if err != nil { 
                stopMotors()
                log.Fatal(err)
        }
        req.Header.Set("Content-Type","application/json")

        client := &http.Client{}
        resp,err := client.Do(req)
        if err != nil{
                stopMotors()
                log.Fatal(err)
        }
        defer resp.Body.Close()
        //audio, err := io.ReadAll(resp.Body)
	output,err := os.Create("output.wav")
	if err != nil {
		stopMotors()
		log.Fatal(err)
	}
	defer output.Close()
	if _,err := io.Copy(output,resp.Body); err != nil {
		stopMotors()
		log.Fatal(err)
	}
}

func moveHeadOut() {
	if head1 == nil {
		log.Fatal("Failed to find GIPO17")
	}
	if head2 == nil {
		log.Fatal("Failed to find GIPO22")
	}
	if err := head1.Out(gpio.High); err != nil {
		log.Fatal(err)
	}
	if err := head2.Out(gpio.Low); err != nil {
		log.Fatal(err)
	}

}

func moveHeadIn() {
	if head1 == nil {
                log.Fatal("Failed to find GIPO17")
        }
        if head2 == nil {
                log.Fatal("Failed to find GIPO22")
        }
        if err := head1.Out(gpio.Low); err != nil {
                log.Fatal(err)
        }
        if err := head2.Out(gpio.High); err != nil {
                log.Fatal(err)
        }
	if err:= head2.Out(gpio.Low); err!=nil {
		log.Fatal(err)
	}





}


func moveMouth(done chan struct{},wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case _, ok := <-done:
			if !ok {
				if err := mouth1.Out(gpio.Low); err!=nil {
					log.Fatal(err)
				}
				return
			}
		default:
			if mouth1 == nil {
				log.Fatal("Failed to find GPIO23")
			}
			if mouth2 == nil {
				log.Fatal("Failed to find GPIO24")
			}
			if err := mouth1.Out(gpio.High); err != nil {
				log.Fatal(err)
			}
			if err := mouth2.Out(gpio.Low); err != nil {
				log.Fatal(err)
			}
			time.Sleep(200*time.Millisecond)
			if err := mouth1.Out(gpio.Low); err != nil {
                                log.Fatal(err)
                        }
                        if err := mouth2.Out(gpio.High); err != nil {
                                log.Fatal(err)
                        }
			time.Sleep(200*time.Millisecond)

		}
	}
}

func moveTail(done chan struct{},wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case _,ok := <-done:
			if !ok{
				if err := tail1.Out(gpio.Low); err !=nil {
					log.Fatal(err)
				}
				return
			}
		default:
			if tail1 == nil {
				log.Fatal("Failed to find GIPO5")
			}
			if tail2 ==nil {
				log.Fatal("Failed to find GIPO6")
			}
			if err := tail1.Out(gpio.High); err != nil {
                        	log.Fatal(err)
                	}
                	if err := tail2.Out(gpio.Low); err != nil {
                        	log.Fatal(err)
                	}
			time.Sleep(200* time.Millisecond)
			if err := tail1.Out(gpio.Low); err != nil {
                                log.Fatal(err)
                        }
                        if err := tail2.Out(gpio.High); err != nil {
                                log.Fatal(err)
                        }
			time.Sleep(800*time.Millisecond)

		}
	}


}


func main() {
	var wg sync.WaitGroup
	_, err := host.Init()
	if err != nil {
		log.Fatal(err)
	}
	head1 = mustGetPin("17")
	head2 = mustGetPin("22")
	mouth1 = mustGetPin("23")
	mouth2 = mustGetPin("24")
	tail1 = mustGetPin("5")
	tail2 = mustGetPin("6")
	
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigs
		fmt.Println("\nCaught interrupt! Stopping motors and exiting...")
		stopMotors()
		os.Exit(0) // Important: skip defers, so call stopMotors explicitly
	}()

	defer stopMotors()
	stopMotors()
	moveHeadOut()
	fmt.Print("moving head")
	recordAudio()
	moveHeadIn()
	prompt := generatePrompt("input.wav")
	fmt.Println(prompt)
	response := generateResponse(prompt)
	fmt.Println(response)
	tts(response)
	done := make(chan struct{})
	wg.Add(3)
	go playaudio(done,"/home/pi/billybass/output.wav",&wg)
	go moveMouth(done,&wg)
	go moveTail(done,&wg)
	wg.Wait()
	if err := tail1.Out(gpio.Low); err != nil {
        	 log.Fatal(err)
        }
        if err := tail2.Out(gpio.Low); err != nil {
        	log.Fatal(err)
        }

}
