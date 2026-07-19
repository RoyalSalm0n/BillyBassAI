from flask import Flask, request, jsonify, send_file
import whisper
import tempfile
import requests
from faster_whisper import WhisperModel
from piper import PiperVoice
import wave
from google import genai
from google.genai import types
import configparser
from dotenv import load_dotenv
from elevenlabs.client import ElevenLabs
from elevenlabs import play
import os
from pydub import AudioSegment
import re
from functools import wraps


app = Flask(__name__)
wModel = WhisperModel("small")
pModel = "/app/models/norman.onnx"
pVoice = PiperVoice.load(pModel)
load_dotenv()
elevenlabs = ElevenLabs(
  api_key=os.getenv("ELEVENLABS_API_KEY"),
)
API_KEY=os.getenv("API_KEY")
external = False

def require_auth(f):
    @wraps(f)
    def decorated(*args, **kwargs):
        auth_header = request.headers.get("Authorization", "")
        if not auth_header.startswith("Bearer ") or auth_header[7:] != API_KEY:
            return jsonify({"error": "Unauthorized"}), 401
        return f(*args, **kwargs)
    return decorated


@app.route('/transcribe', methods=['POST'])
@require_auth
#transcribe audio file to text
def transcribe():
    ALLOWED_EXTENSIONS = {'.wav', '.mp3', '.m4a'}
    if request.method != 'POST':
        return jsonify({"error": "Invalid request method"}), 405
    if 'audio' not in request.files:
        return jsonify({"error": "No audio file provided"}), 400
    audio_file = request.files['audio']
    if audio_file.filename == '':
        return jsonify({"error": "No selected file"}), 400
    if not audio_file:
        return jsonify({"error": "No file provided"}), 400
    if not audio_file.filename.endswith(tuple(ALLOWED_EXTENSIONS)):
        return jsonify({"error": "Invalid file type"}), 400
    ext = os.path.splitext(audio_file.filename)[1].lower()
    suffix = ext if ext in ALLOWED_EXTENSIONS else ".wav"
    with tempfile.NamedTemporaryFile(delete=True,suffix=suffix) as temp_file:
        audio_file.save(temp_file.name)
        if not external:
            segments, info = wModel.transcribe(temp_file.name)
            segments = list(segments)
            result = " ".join(x.text for x in segments)
        else:
            client = genai.Client()
            mime = "audio/wav" if suffix == ".wav" else "audio/mpeg" if suffix == ".mp3" else "audio/mp4"
            myfile = client.files.upload(file=temp_file.name,config={"mime_type": mime})
            response = client.models.generate_content(
                model="gemini-3-flash-preview", contents=["Transcribe this audio clip and return the transcription as plain text", myfile]
            )
            result = response.text
        #audio = whisper.load_audio(temp_file.name)
        #audio = whisper.pad_or_trim(audio)
        #mel = whisper.log_mel_spectrogram(audio, n_mels=Wmodel.dims.n_mels).to(Wmodel.device)
        #options = whisper.DecodingOptions()
        #result = whisper.decode(Wmodel, mel, options)

    return result

@app.route('/ai', methods=['POST'])
@require_auth
def ai():
    info = "\n"
    with open("config.txt") as f:
        info = info + f.read()
    request_data = request.json
    prompt = request_data.get('prompt')
    Localurl = "http://ollama:11434/api/generate"
    Localheader = {"Content-Type": "application/json"}
    client = genai.Client(http_options=types.HttpOptions(api_version='v1alpha'))
    grounding_tool = types.Tool(
        google_search=types.GoogleSearch()
    )
    config = types.GenerateContentConfig(
        system_instruction=info,
        tools=[grounding_tool],
	thinking_config=types.ThinkingConfig(thinking_budget=0),
	temperature=0.9,
	max_output_tokens=150
    )
    if not prompt:
        return jsonify({"error": "No prompt provided"}), 400
    try: 
    	response = client.models.generate_content(
    		model='gemini-2.5-flash',
    		contents=prompt,
    		config=config,
        )
    	text = re.sub(r"[^?!.,;—'A-Za-z0-9 ]", "", response.text)
    	return text
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
@require_auth
def tts():
    request_data = request.json
    inputText = request_data.get("text")
    if not inputText:
        return jsonify({"error": "No data provided"}), 400
    output_path = "/app/output/output.wav"
    try:
        audio = elevenlabs.text_to_speech.convert(
            text=inputText,
            voice_id="3M7aIC5oK9dgAfqFxCYk",
            model_id="eleven_multilingual_v2"
        )
        audio_bytes = b"".join(audio)
        with open(output_path, "wb") as wav_file:
            wav_file.write(audio_bytes)
        sound = AudioSegment.from_mp3(output_path)
        sound.export(output_path,format="wav")
        with wave.open(output_path,"rb") as f:
              f.getparams()
        return send_file(output_path, mimetype='audio/wav',as_attachment=False)
    except Exception as e:
        print(f"ElevenLabs failed {e}")
    with wave.open(output_path,"wb") as wav_file:
        pVoice.synthesize_wav(inputText,wav_file)
    return send_file(output_path, mimetype='audio/wav',as_attachment=False)

if __name__ == '__main__':
    app.debug = False
    app.run(host='0.0.0.0', port=5000, threaded=True)


