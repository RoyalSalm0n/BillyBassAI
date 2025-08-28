from flask import Flask, request, jsonify, send_file
import whisper
import tempfile
import requests
from faster_whisper import WhisperModel
from piper import PiperVoice
import wave
from google import genai
from google.genai import types



app = Flask(__name__)
wModel = WhisperModel("small")
@app.route('/transcribe', methods=['POST'])

#transcribe audio file to text
def transcribe():
    if request.method != 'POST':
        return jsonify({"error": "Invalid request method"}), 405
    if 'audio' not in request.files:
        return jsonify({"error": "No audio file provided"}), 400
    audio_file = request.files['audio']
    if audio_file.filename == '':
        return jsonify({"error": "No selected file"}), 400
    if not audio_file:
        return jsonify({"error": "No file provided"}), 400
    if not audio_file.filename.endswith(('.wav', '.mp3', '.m4a')):
        return jsonify({"error": "Invalid file type"}), 400
    with tempfile.NamedTemporaryFile(delete=True) as temp_file:
        audio_file.save(temp_file.name)
        segments, info = wModel.transcribe(temp_file.name)
        segments = list(segments)
        result = " ".join(x.text for x in segments)
        #audio = whisper.load_audio(temp_file.name)
        #audio = whisper.pad_or_trim(audio)
        #mel = whisper.log_mel_spectrogram(audio, n_mels=Wmodel.dims.n_mels).to(Wmodel.device)
        #options = whisper.DecodingOptions()
        #result = whisper.decode(Wmodel, mel, options)
    
    return result

@app.route('/ai', methods=['POST'])
def ai():
    info = "You are Billy, a taxidermied fish mounted on a wall. You have a constant, unblinking view of the room and an encyclopedic knowledge of the current internet,news, genz memes, and brainrot. Your persona is joking,friendly and hip You make use of slang. Your responses should be brief and humrous when appropriate. if you need inforation on soemthing look it up"
    request_data = request.json
    prompt = request_data.get('prompt')
    Localurl = "http://ollama:11434/api/generate"
    Localheader = {"Content-Type": "application/json"}
    client = genai.Client(http_options=types.HttpOptions(api_version='v1alpha'))
    if not prompt:
        return jsonify({"error": "No prompt provided"}), 400
    try: 
    	response = client.models.generate_content(
    		model='gemini-2.5-flash-lite',
    		contents=prompt,
    		config=types.GenerateContentConfig(
        	system_instruction=info,
        	),
        )
    	return response.text
    except Exception as e:
    	print(f"Gemini failed: {e}")
    try:
    	Localresponse = requests.post(Localurl, headers=Localheader, json={
        	"prompt": prompt,
        	"model": "phi3",
        	"stream": False,
        	"system": info,
        	"temperature": 0.2,
        	"num_predict": 60
        	})
    	print(Localresponse)
    	if Localresponse.status_code != 200:
        	return jsonify({"error": "Failed to get response from AI service"}), 500
    	return Localresponse.json()["response"]
    except Exception as e2:
    	print(f"Local model failed {e2}")
    	return jsonify({"error": "Both Gemini and local model failed"}), 500

@app.route('/tts', methods=['POST'])
def tts():
    request_data = request.json
    text = request_data.get("text")
    if not text:
        return jsonify({"error": "No data provided"}), 400
    model = "/app/models/en_US-bryce-medium.onnx"
    voice = PiperVoice.load(model)
    output_path = "/app/output/output.wav"
    
    with wave.open(output_path, "wb") as wav_file:
    	voice.synthesize_wav(text,wav_file)
    return send_file(output_path, mimetype='audio/wav',as_attachment=False)

if __name__ == '__main__':
    app.debug = True
    app.run(host='0.0.0.0', port=5000)


