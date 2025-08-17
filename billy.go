package main

import (
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/host/v3"
	"log"
	"time"
	"os/exec"
	"fmt"
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
	fmt.Println("Recording Audio")
	cmd := exec.Command("rec", "input.wav", "gain +10", "silence", "1", "0.5", "2%", "1", "2.0", "5%")
	if err := cmd.Run(); err != nil {
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
        if err := mouth1.Out(gpio.Low) err != nil {
                fmt.Println("Cant find mouth1")
        }
        if err := mouth2.Out(gpio.Low) err != nil {
                fmt.Println("Cant find mouth2")
        }
        if err := head1.Out(gpio.Low) err != nil {
                fmt.Println("Cant find head1")
        }
        if err := head2.Out(gpio.Low) err != nil {
                fmt.Println("Cant find head2")
        }
        if err := tail1.Out(gpio.Low) err != nil {
                fmt.Println("Cant find tail1")
        }
        if err := tail2.Out(gpio.Low) err != nil {
                fmt.Println("Cant find tail2")
        }



}

func playaudio(done chan struct{},file string,wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Println("playing audio")
	cmd := exec.Command("aplay",file)
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
	defer close(done)
}

func generatePrompt (audio_path string) {
	transcribeUrl := "***REMOVED***:5000/transcribe"
	file,err :=os.Open(audio_path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)
	part, err := writer.CreateFormFile("audio",audio_path)
	if err != nil {
		log.Fatal(err)
	}
	_,err = io.copy(part,file)
	if err != nil {
		log.Fatal(err)
	}
	writer.Close()
	req, err  := http.NewRequest("POST", transcribeUrl, &requestBody)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Conent-Type",writer.FormDataContentType())
	client :=&http.Client{}
	resp, err != client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer prompt.Body.Close()
	prompt, err := io.ReadAll(resp.Body)
	return string(body)
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
	moveHeadOut()
	recordAudio()
	moveHeadIn()
	fmt.Println(generatePrompt("input.wav"))
	done := make(chan struct{})
	wg.Add(3)
	go playaudio(done,"input.wav",&wg)
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