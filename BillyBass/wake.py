import openwakeword
from openwakeword.model import Model
import subprocess
import struct
import time
import os
import numpy as np


def main():
	oww_model = Model(
		wakeword_models=["/home/pi/billybass/BillyBass/hey_billy.tflite"],
		inference_framework="tflite"
	)
	#porcupine = pvporcupine.create(
	#access_key=str(os.getenv('porcupine')),
	#keyword_paths=["/home/pi/billybass/BillyBass/Hey-Billy_en_raspberry-pi_v3_0_0.ppn"]
	#)
	#print(porcupine)
	time.sleep(2)
	arecord_proc = subprocess.Popen(
		["arecord","-D","plughw:CARD=Microphone","-f","S16_LE","-r","16000","-c","1","-t","raw"],
		stdout=subprocess.PIPE
	)
	def get_next_audio_frame():
		raw = arecord_proc.stdout.read(512*2)
		if len(raw)<512*2:
			return None
		return struct.unpack_from("h"*512,raw)
	is_running=False
	try:
		while True:
			audio_frame = get_next_audio_frame()
			if audio_frame is None:
				continue
			pred = oww_model.predict(np.array(audio_frame,dtype=np.int16))
			for name, score in pred.items():
				if score > 0.5 and not is_running:
					is_running = True
					arecord_proc.terminate()
					arecord_proc.wait()
					try:
						result = subprocess.run(["/home/pi/billybass/BillyBass/billy"], capture_output=True, text=True, timeout=30)
						with open("/home/pi/billybass/BillyBass/go_stdout.log", "a") as f:
    							f.write("stdout:\n" + result.stdout + "\n")
    							f.write("stderr:\n" + result.stderr + "\n")
						print("stdout:", result.stdout)
						print("stderr:", result.stderr)
						print("exit code:", result.returncode)
					finally:
						time.sleep(1)
						arecord_proc = subprocess.Popen(
							["arecord","-D","plughw:CARD=Microphone","-f","S16_LE","-r","16000","-c","1","-t","raw"],
                					stdout=subprocess.PIPE
        					)
						is_running=False
					break
	finally: 
		arecord_proc.kill()




if __name__ == "__main__":
    main()
