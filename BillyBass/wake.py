import pvporcupine
import subprocess
from pvrecorder import PvRecorder
import struct
import time
import os


def main():
	porcupine = pvporcupine.create(
	access_key=str(os.getenv('porcupine')),
	keyword_paths=["/home/pi/billybass/Hey-Billy_en_raspberry-pi_v3_0_0.ppn"]
	)
	print(porcupine)
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
	try:
		while True:
			audio_frame = get_next_audio_frame()
			if audio_frame is None:
				continue
			keyword_index = porcupine.process(audio_frame)
			if keyword_index >= 0:
				arecord_proc.terminate()
				arecord_proc.wait()
				result = subprocess.run(["/home/pi/billybass/billy"], capture_output=True, text=True)
				with open("/home/pi/billybass/go_stdout.log", "a") as f:
    					f.write("stdout:\n" + result.stdout + "\n")
    					f.write("stderr:\n" + result.stderr + "\n")
				print("stdout:", result.stdout)
				print("stderr:", result.stderr)
				print("exit code:", result.returncode)
				arecord_proc = subprocess.Popen(
					["arecord","-f","S16_LE","-r","16000","-c","1","-t","raw"],
                			stdout=subprocess.PIPE
        			)
			
	finally: 
		arecord_proc.kill()
		porcupine.delete()




if __name__ == "__main__":
    main()
