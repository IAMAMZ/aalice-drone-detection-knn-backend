import React, { useEffect, useState, useRef } from "react";
import io from "socket.io-client";
import Listen from "../components/Listen";
import ResultsPanel from "../components/ResultsPanel";
import DroneMap from "../components/DroneMap";
import FileUploadDetection from "../components/FileUploadDetection";
import { FaMicrophoneLines } from "react-icons/fa6";
import { LiaLaptopSolid } from "react-icons/lia";
import { toast } from "react-toastify";
import { MediaRecorder, register } from "extendable-media-recorder";
import { connect } from "extendable-media-recorder-wav-encoder";
import { FFmpeg } from "@ffmpeg/ffmpeg";
import { fetchFile } from "@ffmpeg/util";

const getBackendUrl = () => {
  if (process.env.REACT_APP_BACKEND_URL) {
    return process.env.REACT_APP_BACKEND_URL;
  }
  if (window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1') {
    return 'http://localhost:5001';
  }
  return window.location.origin;
};

const server = getBackendUrl();

function DetectionPage({ modelInfo, setModelInfo }) {
  let ffmpegLoaded = false;
  const ffmpeg = new FFmpeg();
  const isPhone = window.innerWidth <= 550;
  const [stream, setStream] = useState();
  const [classification, setClassification] = useState(null);
  const [lastUpdated, setLastUpdated] = useState(null);
  const [isListening, setisListening] = useState(false);
  const [audioInput, setAudioInput] = useState("device");
  const [registeredMediaEncoder, setRegisteredMediaEncoder] = useState(false);
  const [socketConnected, setSocketConnected] = useState(false);
  const [audioWaveform, setAudioWaveform] = useState(null);
  const [audioFrequencyData, setAudioFrequencyData] = useState(null);

  const streamRef = useRef(stream);
  const socketRef = useRef(null);
  let sendRecordingRef = useRef(true);

  useEffect(() => {
    streamRef.current = stream;
  }, [stream]);

  useEffect(() => {
    if (isPhone) {
      setAudioInput("mic");
    }
  }, [isPhone]);

  useEffect(() => {
    if (socketRef.current) {
      console.log("Cleaning up existing socket");
      socketRef.current.removeAllListeners();
      socketRef.current.disconnect();
      socketRef.current = null;
    }

    console.log("Creating new socket connection to:", server);
    const socket = io(server, {
      transports: ['websocket', 'polling'],
      reconnection: true,
      reconnectionAttempts: 5,
      reconnectionDelay: 2000,
      timeout: 20000,
      forceNew: true,
      autoConnect: true,
    });
    
    socketRef.current = socket;

    socket.io.on("error", (error) => {
      console.error("Socket.IO engine error:", error);
    });

    socket.io.on("open", () => {
      console.log("Socket.IO engine opened");
    });

    socket.on("connect", () => {
      console.log("✓ Socket connected:", socket.id);
      setSocketConnected(true);
      console.log("Emitting requestModelInfo event");
      socket.emit("requestModelInfo", "test");
    });

    socket.on("disconnect", (reason) => {
      console.log("✗ Socket disconnected:", reason);
      setSocketConnected(false);
    });

    socket.on("connect_error", (error) => {
      console.error("✗ Socket connection error:", error.message, error);
    });

    socket.on("reconnect", (attemptNumber) => {
      console.log("✓ Socket reconnected after", attemptNumber, "attempts");
      setSocketConnected(true);
    });

    socket.on("reconnect_attempt", (attemptNumber) => {
      console.log("→ Reconnection attempt", attemptNumber);
    });

    socket.on("reconnect_failed", () => {
      console.error("✗ Failed to reconnect after all attempts");
      toast.error("Lost connection to server. Please refresh the page.");
    });

    socket.on("classification", (result) => {
      console.log("Received classification result");
      setClassification(result);
      setLastUpdated(new Date());
      cleanUp();
    });

    socket.on("analysisError", (msg) => {
      const message = msg?.message || "Analysis error";
      console.error("Analysis error from server:", message);
      toast.error(() => <div>{message}</div>);
    });

    socket.on("modelInfo", (info) => {
      console.log("Received model info:", info);
      if (setModelInfo) {
        setModelInfo(info);
      }
    });

    return () => {
      console.log("Cleaning up socket connection");
      if (socketRef.current) {
        socketRef.current.removeAllListeners();
        socketRef.current.disconnect();
      }
    };
  }, []);

  // Function to extract waveform data from audio buffer
  const extractAudioData = async (audioBlob) => {
    try {
      const audioContext = new (window.AudioContext || window.webkitAudioContext)();
      const arrayBuffer = await audioBlob.arrayBuffer();
      const audioBuffer = await audioContext.decodeAudioData(arrayBuffer);
      
      // Get channel data
      const channelData = audioBuffer.getChannelData(0);
      
      // Sample the waveform (take every nth sample for visualization)
      const samples = 200; // Number of points to display
      const blockSize = Math.floor(channelData.length / samples);
      const waveformData = [];
      
      for (let i = 0; i < samples; i++) {
        const start = blockSize * i;
        let sum = 0;
        let max = 0;
        
        for (let j = 0; j < blockSize; j++) {
          const val = Math.abs(channelData[start + j]);
          sum += val;
          max = Math.max(max, val);
        }
        
        // Use RMS for smoother visualization
        const rms = Math.sqrt(sum / blockSize);
        waveformData.push(rms);
      }
      
      // Calculate frequency data using FFT
      const analyser = audioContext.createAnalyser();
      analyser.fftSize = 2048;
      const bufferLength = analyser.frequencyBinCount;
      const frequencyData = new Uint8Array(bufferLength);
      
      // Create a source from the audio buffer
      const source = audioContext.createBufferSource();
      source.buffer = audioBuffer;
      source.connect(analyser);
      
      // Get frequency data (we'll use a snapshot)
      analyser.getByteFrequencyData(frequencyData);
      
      // Sample frequency data for visualization
      const freqSamples = 40;
      const freqBlockSize = Math.floor(bufferLength / freqSamples);
      const sampledFreqData = [];
      
      for (let i = 0; i < freqSamples; i++) {
        let sum = 0;
        for (let j = 0; j < freqBlockSize; j++) {
          sum += frequencyData[i * freqBlockSize + j];
        }
        sampledFreqData.push(sum / freqBlockSize / 255); // Normalize to 0-1
      }
      
      return {
        waveform: waveformData,
        frequency: sampledFreqData
      };
    } catch (error) {
      console.error("Error extracting audio data:", error);
      return null;
    }
  };

  async function record() {
    try {
      if (!ffmpegLoaded) {
        await ffmpeg.load();
        ffmpegLoaded = true;
      }

      const mediaDevice =
        audioInput === "device"
          ? navigator.mediaDevices.getDisplayMedia.bind(navigator.mediaDevices)
          : navigator.mediaDevices.getUserMedia.bind(navigator.mediaDevices);

      if (!registeredMediaEncoder) {
        await register(await connect());
        setRegisteredMediaEncoder(true);
      }

      const constraints = {
        audio: {
          autoGainControl: false,
          channelCount: 1,
          echoCancellation: false,
          noiseSuppression: false,
          sampleSize: 16,
        },
      };

      const stream = await mediaDevice(constraints);
      const audioTracks = stream.getAudioTracks();
      if (audioTracks.length === 0) {
        toast.error("No audio track detected. Ensure you shared with sound.");
        stream.getTracks().forEach((track) => track.stop());
        return;
      }

      const audioStream = new MediaStream(audioTracks);

      setStream(audioStream);

      audioTracks[0].onended = stopListening;

      for (const track of stream.getVideoTracks()) {
        track.stop();
      }

      const mediaRecorder = new MediaRecorder(audioStream, {
        mimeType: "audio/wav",
      });

      mediaRecorder.start();
      setisListening(true);
      sendRecordingRef.current = true;

      const chunks = [];
      mediaRecorder.ondataavailable = function (e) {
        chunks.push(e.data);
      };

      setTimeout(function () {
        mediaRecorder.stop();
      }, 20000);

      mediaRecorder.addEventListener("stop", async () => {
        const blob = new Blob(chunks, { type: "audio/wav" });

        // Extract audio visualization data
        const audioData = await extractAudioData(blob);
        if (audioData) {
          setAudioWaveform(audioData.waveform);
          setAudioFrequencyData(audioData.frequency);
        }

        cleanUp();

        const inputFile = "input.wav";
        const outputFile = "output_mono.wav";

        await ffmpeg.writeFile(inputFile, await fetchFile(blob));
        const exitCode = await ffmpeg.exec([
          "-i", inputFile,
          "-c", "pcm_s16le",
          "-ar", "44100",
          "-ac", "1",
          "-f", "wav",
          outputFile
        ]);
        if (exitCode !== 0) {
          throw new Error(`FFmpeg exec failed with exit code: ${exitCode}`);
        }

        const monoData = await ffmpeg.readFile(outputFile);
        const monoBlob = new Blob([monoData.buffer], { type: "audio/wav" });

        const reader = new FileReader();
        reader.readAsArrayBuffer(monoBlob);
        reader.onload = async (event) => {
          const arrayBuffer = event.target.result;
          const bytes = new Uint8Array(arrayBuffer);
          let binary = "";
          for (let i = 0; i < bytes.byteLength; i++) {
            binary += String.fromCharCode(bytes[i]);
          }

          const dataView = new DataView(arrayBuffer);
          
          let latitude = null;
          let longitude = null;
          
          const getLocation = () => {
            return new Promise((resolve) => {
              if (navigator.geolocation) {
                navigator.geolocation.getCurrentPosition(
                  (position) => {
                    latitude = position.coords.latitude;
                    longitude = position.coords.longitude;
                    console.log("Location captured:", latitude, longitude);
                    resolve({ latitude, longitude });
                  },
                  (error) => {
                    console.warn("Geolocation error:", error.message);
                    resolve({ latitude: null, longitude: null });
                  },
                  { timeout: 5000, maximumAge: 60000 }
                );
              } else {
                resolve({ latitude: null, longitude: null });
              }
            });
          };
          
          const location = await getLocation();
          
          const recordData = {
            audio: btoa(binary),
            channels: dataView.getUint16(22, true),
            sampleRate: dataView.getUint32(24, true),
            sampleSize: dataView.getUint16(34, true),
            duration: 0,
            latitude: location.latitude,
            longitude: location.longitude,
          };

          const bytesPerSample = recordData.sampleSize / 8;
          if (bytesPerSample > 0 && recordData.sampleRate) {
            const frameCount = bytes.byteLength / (bytesPerSample * recordData.channels);
            recordData.duration = frameCount / recordData.sampleRate;
          }

          if (sendRecordingRef.current) {
            const jsonData = JSON.stringify(recordData);
            console.log("Sending audio classification request", {
              sampleRate: recordData.sampleRate,
              channels: recordData.channels,
              duration: recordData.duration,
              audioLength: recordData.audio?.length || 0,
              latitude: recordData.latitude,
              longitude: recordData.longitude,
            });
            
            fetch(`${server}/api/audio/classify`, {
              method: 'POST',
              headers: {
                'Content-Type': 'application/json',
              },
              body: jsonData,
            })
              .then(response => {
                if (!response.ok) {
                  return response.json().then(err => {
                    throw new Error(err.message || `HTTP error! status: ${response.status}`);
                  });
                }
                return response.json();
              })
              .then(result => {
                console.log("Received classification result", result);
                setClassification(result);
                setLastUpdated(new Date());
                cleanUp();
              })
              .catch(error => {
                console.error("Error classifying audio:", error);
                toast.error(error.message || "Failed to classify audio");
                cleanUp();
              });
            
            sendRecordingRef.current = false;
          }
        };
      });
    } catch (error) {
      console.error("error:", error);
      cleanUp();
    }
  }

  function cleanUp() {
    const currentStream = streamRef.current;
    if (currentStream) {
      currentStream.getTracks().forEach((track) => track.stop());
    }

    setStream(null);
    setisListening(false);
  }

  function stopListening() {
    cleanUp();
  }

  function handleLaptopIconClick() {
    setAudioInput("device");
  }

  function handleMicrophoneIconClick() {
    setAudioInput("mic");
  }

  function handleFileClassification(result, audioData) {
    console.log("File classification complete:", result);
    setClassification(result);
    setLastUpdated(new Date());
    if (audioData) {
      setAudioWaveform(audioData.waveform);
      setAudioFrequencyData(audioData.frequency);
    }
  }

  async function sendDroneAlert() {
    try {
      if (!classification || classification.isDrone === false || !classification.predictions || classification.predictions.length === 0) {
        toast.error("No prediction data available to send SMS");
        return;
      }

      const firstPrediction = JSON.stringify(classification.predictions[0]);
      const smsBody = firstPrediction;

      console.log("Sending SMS alert:", smsBody);

      const response = await fetch(`${server}/api/sms/send`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ body: smsBody }),
      });

      if (!response.ok) {
        const error = await response.json().catch(() => ({}));
        throw new Error(error.message || `HTTP error! status: ${response.status}`);
      }

      const result = await response.json();
      console.log("SMS sent successfully:", result);
      toast.success("SMS alert sent successfully!");

    } catch (error) {
      console.error("Error sending SMS:", error);
      toast.error(error.message || "Failed to send SMS alert");
    }
  }

  return (
    <div className="DetectionPage">
      <div className="MainGrid" style={{
        display: 'grid',
        gridTemplateColumns: isPhone ? '1fr' : '1fr 1fr',
        gap: '1.5rem',
        maxWidth: '1400px',
        margin: '0 auto',
        padding: '1.5rem'
      }}>
        {/* Left Column */}
        <section className="GridColumn" style={{
          display: 'flex',
          flexDirection: 'column',
          gap: '1.5rem'
        }}>
          {/* Listen Card */}
          <div className="Card listenCard" style={{
            borderRadius: '20px',
            padding: '2rem',
            boxShadow: '0 10px 30px rgba(0,0,0,0.2)',
            position: 'relative',
            overflow: 'hidden'
          }}>
            <div style={{
              position: 'absolute',
              top: 0,
              left: 0,
              right: 0,
              bottom: 0,
              background: 'radial-gradient(circle at 30% 50%, rgba(255,255,255,0.1) 0%, transparent 50%)',
              pointerEvents: 'none'
            }}></div>
            
            <div className="CardHeader" style={{
              textAlign: 'center',
              marginBottom: '2rem',
              position: 'relative',
              zIndex: 1
            }}>
              <h3 style={{
                color: 'white',
                fontSize: '1.75rem',
                fontWeight: '700',
                marginBottom: '0.5rem',
                textShadow: '0 2px 4px rgba(0,0,0,0.2)'
              }}>Realtime Detection</h3>
              <p style={{
                color: 'rgba(255,255,255,0.9)',
                fontSize: '0.95rem',
                lineHeight: '1.5'
              }}>Capture up to 20 seconds of live audio to analyze against your signature library</p>
            </div>
            
            <div className="ListenWrapper" style={{
              display: 'flex',
              justifyContent: 'center',
              marginBottom: '1.5rem',
              position: 'relative',
              zIndex: 1
            }}>
              <Listen
                stopListening={stopListening}
                disable={!socketConnected}
                startListening={record}
                isListening={isListening}
              />
            </div>
            
            {!isPhone && (
              <div className="audio-input" style={{
                display: 'flex',
                justifyContent: 'center',
                gap: '1rem',
                position: 'relative',
                zIndex: 1
              }}>
                <button
                  type="button"
                  onClick={handleLaptopIconClick}
                  className={audioInput === "device" ? "audio-input-device active-audio-input" : "audio-input-device"}
                  aria-label="Use system audio"
                  style={{
                    background: audioInput === "device" ? 'rgba(255,255,255,0.3)' : 'rgba(255,255,255,0.15)',
                    border: 'none',
                    borderRadius: '12px',
                    padding: '12px 20px',
                    color: 'white',
                    cursor: 'pointer',
                    transition: 'all 0.3s ease',
                    backdropFilter: 'blur(10px)'
                  }}
                >
                  <LiaLaptopSolid style={{ height: 20, width: 20 }} />
                </button>
                <button
                  type="button"
                  onClick={handleMicrophoneIconClick}
                  className={audioInput === "mic" ? "audio-input-mic active-audio-input" : "audio-input-mic"}
                  aria-label="Use microphone input"
                  style={{
                    background: audioInput === "mic" ? 'rgba(255,255,255,0.3)' : 'rgba(255,255,255,0.15)',
                    border: 'none',
                    borderRadius: '12px',
                    padding: '12px 20px',
                    color: 'white',
                    cursor: 'pointer',
                    transition: 'all 0.3s ease',
                    backdropFilter: 'blur(10px)'
                  }}
                >
                  <FaMicrophoneLines style={{ height: 20, width: 20 }} />
                </button>
              </div>
            )}
          </div>

          {/* File Upload Detection Card */}
          <div className="Card uploadCard">
            <FileUploadDetection
              backendUrl={server}
              onClassificationComplete={handleFileClassification}
            />
          </div>

          {/* Results Panel */}
          <ResultsPanel
            classification={classification}
            modelInfo={modelInfo}
            lastUpdated={lastUpdated}
            isListening={isListening}
          />
        </section>
        
        {/* Right Column */}
        <section className="GridColumn" style={{
          display: 'flex',
          flexDirection: 'column',
          gap: '1.5rem'
        }}>
          {/* Map Card */}
          <div className="Card" style={{
            borderRadius: '20px',
            overflow: 'hidden',
            boxShadow: '0 4px 15px rgba(0,0,0,0.1)',
            flex: 'none'
          }}>
            <div className="CardHeader" style={{
              padding: '1.5rem',
              borderBottom: '1px solid #e5e7eb'
            }}>
              <h3 style={{
                fontSize: '1.25rem',
                fontWeight: '600',
                marginBottom: '0.25rem'
              }}>Drone Location</h3>
              <p style={{
                color: '#6b7280',
                fontSize: '0.9rem'
              }}>Visualization of detected drone location on map</p>
            </div>
            <DroneMap
              latitude={classification?.latitude}
              longitude={classification?.longitude}
              classification={classification}
            />
          </div>

          {/* Audio Signature Visualization - Real Data */}
          {classification && audioWaveform && (
            <div className="Card" style={{
              borderRadius: '20px',
              overflow: 'hidden',
              boxShadow: '0 4px 15px rgba(0,0,0,0.1)',
            }}>
              <div className="CardHeader" style={{
                padding: '1.5rem',
                borderBottom: '1px solid rgba(255,255,255,0.1)'
              }}>
                <h3 style={{
                  fontSize: '1.25rem',
                  fontWeight: '600',
                  marginBottom: '0.25rem',
                  color: 'white'
                }}>Audio Signature</h3>
                <p style={{
                  color: 'rgba(255,255,255,0.7)',
                  fontSize: '0.9rem'
                }}>Real-time captured sound wave visualization</p>
              </div>
              <div style={{
                padding: '2rem',
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                gap: '1.5rem'
              }}>
                {/* Real Waveform Visualization */}
                <svg width="100%" height="120" viewBox="0 0 400 120" style={{ maxWidth: '500px' }}>
                  {/* Waveform bars from real audio data */}
                  {audioWaveform.map((amplitude, i) => {
                    const normalizedHeight = Math.max(amplitude * 100, 5); // Minimum height of 5
                    const x = (i / audioWaveform.length) * 400;
                    const color = classification.isDrone ? '#dc3545' : '#ea6666ff';
                    
                    return (
                      <rect
                        key={i}
                        x={x}
                        y={60 - normalizedHeight / 2}
                        width={400 / audioWaveform.length - 1}
                        height={normalizedHeight}
                        fill={color}
                        opacity={0.8}
                        rx="2"
                      />
                    );
                  })}
                  {/* Center line */}
                  <line
                    x1="0"
                    y1="60"
                    x2="400"
                    y2="60"
                    stroke="rgba(255,255,255,0.2)"
                    strokeWidth="1"
                    strokeDasharray="5,5"
                  />
                </svg>

                {/* Audio Stats */}
                <div style={{
                  display: 'flex',
                  gap: '2rem',
                  justifyContent: 'center',
                  flexWrap: 'wrap',
                  width: '100%'
                }}>
                  {classification.predictions && classification.predictions.length > 0 && (
                    <>
                      <div style={{
                        textAlign: 'center',
                        padding: '1rem',
                        background: 'rgba(255,255,255,0.05)',
                        borderRadius: '12px',
                        minWidth: '120px'
                      }}>
                        <div style={{
                          fontSize: '0.75rem',
                          color: 'rgba(255,255,255,0.6)',
                          marginBottom: '0.25rem',
                          textTransform: 'uppercase',
                          letterSpacing: '0.5px'
                        }}>Confidence</div>
                        <div style={{
                          fontSize: '1.5rem',
                          fontWeight: '700',
                          color: 'white'
                        }}>
                          {(classification.predictions[0].confidence * 100).toFixed(1)}%
                        </div>
                      </div>
                      <div style={{
                        textAlign: 'center',
                        padding: '1rem',
                        background: 'rgba(255,255,255,0.05)',
                        borderRadius: '12px',
                        minWidth: '120px'
                      }}>
                        <div style={{
                          fontSize: '0.75rem',
                          color: 'rgba(255,255,255,0.6)',
                          marginBottom: '0.25rem',
                          textTransform: 'uppercase',
                          letterSpacing: '0.5px'
                        }}>Category</div>
                        <div style={{
                          fontSize: '1.25rem',
                          fontWeight: '600',
                          color: 'white'
                        }}>
                          {classification.predictions[0].category || 'Unknown'}
                        </div>
                      </div>
                      <div style={{
                        textAlign: 'center',
                        padding: '1rem',
                        background: 'rgba(255,255,255,0.05)',
                        borderRadius: '12px',
                        minWidth: '120px'
                      }}>
                        <div style={{
                          fontSize: '0.75rem',
                          color: 'rgba(255,255,255,0.6)',
                          marginBottom: '0.25rem',
                          textTransform: 'uppercase',
                          letterSpacing: '0.5px'
                        }}>Label</div>
                        <div style={{
                          fontSize: '1.25rem',
                          fontWeight: '600',
                          color: 'white'
                        }}>
                          {classification.predictions[0].label}
                        </div>
                      </div>
                    </>
                  )}
                </div>

                {/* Real Frequency Spectrum from audio data */}
                {audioFrequencyData && (
                  <div style={{
                    display: 'flex',
                    gap: '4px',
                    alignItems: 'flex-end',
                    height: '80px',
                    width: '100%',
                    maxWidth: '500px',
                    padding: '1rem 0'
                  }}>
                    {audioFrequencyData.map((intensity, i) => {
                      const height = Math.max(intensity * 100, 10); // Minimum 10%
                      const color = classification.isDrone ? '#dc3545' : '#ea6666ff';
                      return (
                        <div
                          key={i}
                          style={{
                            flex: 1,
                            height: `${height}%`,
                            background: `linear-gradient(to top, ${color}, ${color}88)`,
                            borderRadius: '2px 2px 0 0',
                            transition: 'height 0.3s ease'
                          }}
                        />
                      );
                    })}
                  </div>
                )}
              </div>
            </div>
          )}

          {/* Alert Card - Only shown when drone detected */}
          {classification && classification.isDrone && (
            <div className="Card alertCard" style={{
              borderRadius: '20px',
              overflow: 'hidden',
              boxShadow: '0 4px 15px rgba(220, 53, 69, 0.2)',
              border: '2px solid #dc3545',
            }}>
              <div className="CardHeader" style={{
                padding: '1.5rem',
                borderBottom: '1px solid #ffcdd2',
              }}>
                <h3 style={{
                  fontSize: '1.25rem',
                  fontWeight: '600',
                  marginBottom: '0.25rem',
                  color: '#dc3545',
                  display: 'flex',
                  alignItems: 'center',
                  gap: '0.5rem'
                }}>
                  <span style={{
                    display: 'inline-block',
                    width: '10px',
                    height: '10px',
                    borderRadius: '50%',
                    background: '#dc3545',
                    animation: 'pulse 2s infinite'
                  }}></span>
                  Drone Alert
                </h3>
                <p style={{
                  color: '#721c24',
                  fontSize: '0.9rem'
                }}>Drone detected! Send SMS notification.</p>
              </div>
              <div style={{ 
                padding: '1.5rem',
                display: 'flex',
                justifyContent: 'center'
              }}>
                <button
                  type="button"
                  onClick={sendDroneAlert}
                  className="btn btn-danger"
                  style={{
                    backgroundColor: '#dc3545',
                    color: 'white',
                    border: 'none',
                    padding: '14px 32px',
                    borderRadius: '12px',
                    cursor: 'pointer',
                    fontSize: '16px',
                    fontWeight: '600',
                    boxShadow: '0 4px 12px rgba(220, 53, 69, 0.3)',
                    transition: 'all 0.3s ease',
                    width: '100%',
                    maxWidth: '300px'
                  }}
                  onMouseEnter={(e) => {
                    e.target.style.backgroundColor = '#c82333';
                    e.target.style.transform = 'translateY(-2px)';
                    e.target.style.boxShadow = '0 6px 16px rgba(220, 53, 69, 0.4)';
                  }}
                  onMouseLeave={(e) => {
                    e.target.style.backgroundColor = '#dc3545';
                    e.target.style.transform = 'translateY(0)';
                    e.target.style.boxShadow = '0 4px 12px rgba(220, 53, 69, 0.3)';
                  }}
                >
                  Send SMS Alert
                </button>
              </div>
            </div>
          )}
        </section>
      </div>
      
      <style>{`
        @keyframes pulse {
          0%, 100% {
            opacity: 1;
          }
          50% {
            opacity: 0.5;
          }
        }
      `}</style>
    </div>
  );
}

export default DetectionPage;