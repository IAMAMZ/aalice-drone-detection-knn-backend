import React, { useMemo, useRef, useState, useEffect } from "react";
import { toast } from "react-toastify";
import { MediaRecorder, register } from "extendable-media-recorder";
import { connect } from "extendable-media-recorder-wav-encoder";
import { FFmpeg } from "@ffmpeg/ffmpeg";
import { fetchFile } from "@ffmpeg/util";
import { FaMicrophoneLines } from "react-icons/fa6";
import styles from "./styles/PrototypeUpload.module.css";

const normaliseBaseUrl = (url) => {
  if (!url) return "";
  return url.endsWith("/") ? url.slice(0, -1) : url;
};

const PrototypeUpload = ({ backendUrl, onUploadSuccess }) => {
  const [label, setLabel] = useState("");
  const [category, setCategory] = useState("drone");
  const [description, setDescription] = useState("");
  const [model, setModel] = useState("");
  const [platformType, setPlatformType] = useState("");
  const [rotorCount, setRotorCount] = useState("");
  const [manufacturer, setManufacturer] = useState("");
  const [droneOrigin, setDroneOrigin] = useState("");
  const [threatLevel, setThreatLevel] = useState("");
  const [riskCategory, setRiskCategory] = useState("");
  const [payloadCapacity, setPayloadCapacity] = useState("");
  const [maxRange, setMaxRange] = useState("");
  const [maxSpeed, setMaxSpeed] = useState("");
  const [flightTime, setFlightTime] = useState("");
  const [jammingSusceptible, setJammingSusceptible] = useState("");
  const [countermeasures, setCountermeasures] = useState("");
  const [files, setFiles] = useState([]);
  const [fileAudioUrls, setFileAudioUrls] = useState(new Map());
  const [isUploading, setIsUploading] = useState(false);
  const [lastUpload, setLastUpload] = useState(null);
  const [isRecording, setIsRecording] = useState(false);
  const [recordingTime, setRecordingTime] = useState(0);
  const [stream, setStream] = useState(null);

  const fileInputRef = useRef(null);
  const mediaRecorderRef = useRef(null);
  const streamRef = useRef(null);
  const ffmpegRef = useRef(null);
  const ffmpegLoadedRef = useRef(false);
  const registeredMediaEncoderRef = useRef(false);
  const recordingIntervalRef = useRef(null);
  const recordingStartTimeRef = useRef(null);
  const apiEndpoint = useMemo(
    () => `${normaliseBaseUrl(backendUrl)}/api/prototypes/upload`,
    [backendUrl]
  );

  useEffect(() => {
    streamRef.current = stream;
  }, [stream]);

  const audioUrlsRef = useRef(new Map());
  
  useEffect(() => {
    audioUrlsRef.current = fileAudioUrls;
  }, [fileAudioUrls]);

  useEffect(() => {
    return () => {
      if (streamRef.current) {
        streamRef.current.getTracks().forEach((track) => track.stop());
      }
      if (recordingIntervalRef.current) {
        clearInterval(recordingIntervalRef.current);
      }
      // Clean up audio URLs
      audioUrlsRef.current.forEach((url) => {
        URL.revokeObjectURL(url);
      });
    };
  }, []);

  const handleFileChange = (event) => {
    const selected = Array.from(event.target.files || []);
    const newUrls = new Map(fileAudioUrls);
    
    selected.forEach((file) => {
      if (file.type.startsWith('audio/') && !newUrls.has(file.name)) {
        const url = URL.createObjectURL(file);
        newUrls.set(file.name, url);
      }
    });
    
    setFileAudioUrls(newUrls);
    setFiles((prevFiles) => [...prevFiles, ...selected]);
  };

  const resetForm = () => {
    // Clean up audio URLs
    fileAudioUrls.forEach((url) => {
      URL.revokeObjectURL(url);
    });
    
    setLabel("");
    setCategory("drone");
    setDescription("");
    setModel("");
    setPlatformType("");
    setRotorCount("");
    setManufacturer("");
    setDroneOrigin("");
    setThreatLevel("");
    setRiskCategory("");
    setPayloadCapacity("");
    setMaxRange("");
    setMaxSpeed("");
    setFlightTime("");
    setJammingSusceptible("");
    setCountermeasures("");
    setFiles([]);
    setFileAudioUrls(new Map());
    setRecordingTime(0);
    if (fileInputRef.current) {
      fileInputRef.current.value = "";
    }
    stopRecording();
  };

  const stopRecording = () => {
    if (mediaRecorderRef.current && mediaRecorderRef.current.state !== "inactive") {
      mediaRecorderRef.current.stop();
    }
    if (streamRef.current) {
      streamRef.current.getTracks().forEach((track) => track.stop());
    }
    if (recordingIntervalRef.current) {
      clearInterval(recordingIntervalRef.current);
      recordingIntervalRef.current = null;
    }
    recordingStartTimeRef.current = null;
    setIsRecording(false);
    setStream(null);
    setRecordingTime(0);
  };

  const startRecording = async () => {
    try {
      if (!ffmpegLoadedRef.current) {
        if (!ffmpegRef.current) {
          ffmpegRef.current = new FFmpeg();
        }
        await ffmpegRef.current.load();
        ffmpegLoadedRef.current = true;
      }

      if (!registeredMediaEncoderRef.current) {
        await register(await connect());
        registeredMediaEncoderRef.current = true;
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

      const audioStream = await navigator.mediaDevices.getUserMedia(constraints);
      setStream(audioStream);
      streamRef.current = audioStream;

      const mediaRecorder = new MediaRecorder(audioStream, {
        mimeType: "audio/wav",
      });

      mediaRecorderRef.current = mediaRecorder;

      const chunks = [];
      mediaRecorder.ondataavailable = (e) => {
        if (e.data.size > 0) {
          chunks.push(e.data);
        }
      };

      mediaRecorder.addEventListener("stop", async () => {
        try {
          const blob = new Blob(chunks, { type: "audio/wav" });

          if (blob.size === 0) {
            toast.error("No audio data recorded");
            stopRecording();
            return;
          }

          const inputFile = "input.wav";
          const outputFile = "output_mono.wav";

          await ffmpegRef.current.writeFile(inputFile, await fetchFile(blob));
          const exitCode = await ffmpegRef.current.exec([
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

          const monoData = await ffmpegRef.current.readFile(outputFile);
          const monoBlob = new Blob([monoData.buffer], { type: "audio/wav" });

          const timestamp = new Date().toISOString().replace(/[:.]/g, "-");
          const fileName = `recording_${timestamp}.wav`;
          const file = new File([monoBlob], fileName, { type: "audio/wav" });

          // Create object URL for audio playback
          const audioUrl = URL.createObjectURL(monoBlob);
          setFileAudioUrls((prevUrls) => {
            const newUrls = new Map(prevUrls);
            newUrls.set(fileName, audioUrl);
            return newUrls;
          });

          setFiles((prevFiles) => [...prevFiles, file]);
          toast.success(`Recording added: ${fileName}`);

          stopRecording();
        } catch (error) {
          console.error("Error processing recording:", error);
          toast.error("Failed to process recording: " + error.message);
          stopRecording();
        }
      });

      // Clear any existing interval
      if (recordingIntervalRef.current) {
        clearInterval(recordingIntervalRef.current);
      }

      mediaRecorder.start();
      setIsRecording(true);
      setRecordingTime(0);
      recordingStartTimeRef.current = Date.now();

      // Use a more accurate timer based on actual elapsed time
      recordingIntervalRef.current = setInterval(() => {
        if (recordingStartTimeRef.current) {
          const elapsed = Math.floor((Date.now() - recordingStartTimeRef.current) / 1000);
          setRecordingTime(elapsed);
        }
      }, 100);

      toast.info("Recording started. Click stop when finished.");
    } catch (error) {
      console.error("Error starting recording:", error);
      toast.error("Failed to start recording: " + error.message);
      stopRecording();
    }
  };

  const handleSubmit = async (event) => {
    event.preventDefault();
    const trimmedLabel = label.trim();

    if (!trimmedLabel) {
      toast.error("Provide a label for the uploaded signatures.");
      return;
    }

    if (files.length === 0) {
      toast.error("Attach at least one audio file to ingest.");
      return;
    }

    setIsUploading(true);

    try {
      const formData = new FormData();
      formData.append("label", trimmedLabel);
      formData.append("category", category);

      if (description.trim()) {
        formData.append("description", description.trim());
      }
      if (model.trim()) {
        formData.append("model", model.trim());
      }
      if (platformType.trim()) {
        formData.append("type", platformType.trim());
      }
      if (rotorCount.trim()) {
        formData.append("rotor_count", rotorCount.trim());
      }
      if (manufacturer.trim()) {
        formData.append("manufacturer", manufacturer.trim());
      }
      if (droneOrigin.trim()) {
        formData.append("drone_origin", droneOrigin.trim());
      }
      if (threatLevel.trim()) {
        formData.append("threat_level", threatLevel.trim());
      }
      if (riskCategory.trim()) {
        formData.append("risk_category", riskCategory.trim());
      }
      if (payloadCapacity.trim()) {
        formData.append("payload_capacity_kg", payloadCapacity.trim());
      }
      if (maxRange.trim()) {
        formData.append("max_range_km", maxRange.trim());
      }
      if (maxSpeed.trim()) {
        formData.append("max_speed_ms", maxSpeed.trim());
      }
      if (flightTime.trim()) {
        formData.append("flight_time_minutes", flightTime.trim());
      }
      if (jammingSusceptible.trim()) {
        formData.append("jamming_susceptible", jammingSusceptible.trim());
      }
      if (countermeasures.trim()) {
        formData.append("countermeasure_recommendations", countermeasures.trim());
      }

      files.forEach((file) => {
        formData.append("samples", file, file.name);
      });

      const response = await fetch(apiEndpoint, {
        method: "POST",
        body: formData,
      });

      const payload = await response.json().catch(() => null);

      if (!response.ok) {
        const message = payload?.message || "Upload failed";
        throw new Error(message);
      }

      const added = payload?.added || [];
      setLastUpload({
        label: trimmedLabel,
        count: added.length,
        timestamp: new Date(),
        prototypes: added,
      });

      toast.success(`Registered ${added.length} signatures for ${trimmedLabel}.`);
      resetForm();

      if (typeof onUploadSuccess === "function") {
        onUploadSuccess(payload);
      }
    } catch (error) {
      toast.error(error.message || "Unable to upload prototypes.");
    } finally {
      setIsUploading(false);
    }
  };

  return (
    <section className={styles.container}>
      <header className={styles.header}>
        <div>
          <h3>Upload Acoustic Signatures</h3>
          <p>Ingest labelled audio clips to expand the realtime classifier.</p>
        </div>
        {lastUpload && (
          <div className={styles.summary}>
            <span className={styles.summaryLabel}>{lastUpload.label}</span>
            <span className={styles.summaryCount}>{lastUpload.count} added</span>
            <span className={styles.summaryTime}>
              {lastUpload.timestamp.toLocaleTimeString()}
            </span>
          </div>
        )}
      </header>

      <form className={styles.form} onSubmit={handleSubmit}>
        <div className={styles.fieldRow}>
          <label className={styles.field}>
            <span>Label *</span>
            <input
              type="text"
              value={label}
              onChange={(event) => setLabel(event.target.value)}
              placeholder="e.g. dji_mavic"
            />
          </label>
          <label className={styles.field}>
            <span>Category</span>
            <select value={category} onChange={(event) => setCategory(event.target.value)}>
              <option value="drone">drone</option>
              <option value="noise">noise</option>
              <option value="unknown">unknown</option>
            </select>
          </label>
        </div>

        <div className={styles.fieldRow}>
          <label className={styles.field}>
            <span>Description</span>
            <input
              type="text"
              value={description}
              onChange={(event) => setDescription(event.target.value)}
              placeholder="DJI Mavic 2 Pro hover"
            />
          </label>
          <label className={styles.field}>
            <span>Model</span>
            <input
              type="text"
              value={model}
              onChange={(event) => setModel(event.target.value)}
              placeholder="DJI Mavic 2 Pro"
            />
          </label>
        </div>

        <div className={styles.fieldRow}>
          <label className={styles.field}>
            <span>Platform Type</span>
            <input
              type="text"
              value={platformType}
              onChange={(event) => setPlatformType(event.target.value)}
              placeholder="Quadcopter"
            />
          </label>
          <label className={styles.field}>
            <span>Rotor Count</span>
            <input
              type="text"
              value={rotorCount}
              onChange={(event) => setRotorCount(event.target.value)}
              placeholder="4"
            />
          </label>
          <label className={styles.field}>
            <span>Manufacturer</span>
            <input
              type="text"
              value={manufacturer}
              onChange={(event) => setManufacturer(event.target.value)}
              placeholder="DJI"
            />
          </label>
          <label className={styles.field}>
            <span>Drone Origin</span>
            <input
              type="text"
              value={droneOrigin}
              onChange={(event) => setDroneOrigin(event.target.value)}
              placeholder="e.g. Iran, China, Russia"
            />
          </label>
        </div>

        <div className={styles.fieldRow}>
          <label className={styles.field}>
            <span>Threat Level</span>
            <select value={threatLevel} onChange={(event) => setThreatLevel(event.target.value)}>
              <option value="">Select...</option>
              <option value="low">Low</option>
              <option value="medium">Medium</option>
              <option value="high">High</option>
              <option value="critical">Critical</option>
            </select>
          </label>
          <label className={styles.field}>
            <span>Risk Category</span>
            <select value={riskCategory} onChange={(event) => setRiskCategory(event.target.value)}>
              <option value="">Select...</option>
              <option value="surveillance">Surveillance</option>
              <option value="payload_delivery">Payload Delivery</option>
              <option value="reconnaissance">Reconnaissance</option>
              <option value="swarm">Swarm</option>
              <option value="commercial">Commercial</option>
              <option value="hobbyist">Hobbyist</option>
            </select>
          </label>
        </div>

        <div className={styles.fieldRow}>
          <label className={styles.field}>
            <span>Payload Capacity (kg)</span>
            <input
              type="text"
              value={payloadCapacity}
              onChange={(event) => setPayloadCapacity(event.target.value)}
              placeholder="2.5"
            />
          </label>
          <label className={styles.field}>
            <span>Max Range (km)</span>
            <input
              type="text"
              value={maxRange}
              onChange={(event) => setMaxRange(event.target.value)}
              placeholder="8"
            />
          </label>
          <label className={styles.field}>
            <span>Max Speed (m/s)</span>
            <input
              type="text"
              value={maxSpeed}
              onChange={(event) => setMaxSpeed(event.target.value)}
              placeholder="20"
            />
          </label>
        </div>

        <div className={styles.fieldRow}>
          <label className={styles.field}>
            <span>Flight Time (minutes)</span>
            <input
              type="text"
              value={flightTime}
              onChange={(event) => setFlightTime(event.target.value)}
              placeholder="31"
            />
          </label>
          <label className={styles.field}>
            <span>Jamming Susceptible</span>
            <select value={jammingSusceptible} onChange={(event) => setJammingSusceptible(event.target.value)}>
              <option value="">Select...</option>
              <option value="true">Yes</option>
              <option value="false">No</option>
            </select>
          </label>
        </div>

        <div className={styles.fieldRow}>
          <label className={styles.field} style={{ flex: "1 1 100%" }}>
            <span>Countermeasure Recommendations</span>
            <input
              type="text"
              value={countermeasures}
              onChange={(event) => setCountermeasures(event.target.value)}
              placeholder="RF jamming, GPS spoofing, net launcher"
            />
          </label>
        </div>

        <div className={styles.audioInputSection}>
          <label className={`${styles.field} ${styles.fileField}`}>
            <span>Audio Samples *</span>
            <input
              type="file"
              accept="audio/*"
              multiple
              ref={fileInputRef}
              onChange={handleFileChange}
              disabled={isRecording}
            />
            <small>Attach mono or stereo clips; the server will normalise format automatically.</small>
          </label>

          <div className={styles.recordingControls}>
            <div className={styles.recordingButtons}>
              {!isRecording ? (
                <button
                  type="button"
                  className={styles.recordButton}
                  onClick={startRecording}
                  disabled={isUploading}
                >
                  <FaMicrophoneLines style={{ height: 16, width: 16, marginRight: 8 }} />
                  Record from Microphone
                </button>
              ) : (
                <button
                  type="button"
                  className={styles.stopButton}
                  onClick={stopRecording}
                >
                  <span className={styles.recordingIndicator}></span>
                  Stop Recording ({recordingTime}s)
                </button>
              )}
            </div>
            {isRecording && (
              <small className={styles.recordingHint}>
                Recording in progress... Click stop when finished.
              </small>
            )}
          </div>
        </div>

        {files.length > 0 && (
          <ul className={styles.fileList}>
            {files.map((file, index) => {
              const audioUrl = fileAudioUrls.get(file.name);
              return (
                <li key={`${file.name}-${index}`}>
                  <div className={styles.fileInfo}>
                    <span>{file.name}</span>
                    <span className={styles.fileSize}>{(file.size / 1024 / 1024).toFixed(2)} MB</span>
                  </div>
                  {audioUrl && file.type.startsWith('audio/') && (
                    <div className={styles.audioPlayer}>
                      <audio controls src={audioUrl} preload="metadata">
                        Your browser does not support the audio element.
                      </audio>
                    </div>
                  )}
                </li>
              );
            })}
          </ul>
        )}

        <div className={styles.actions}>
          <button type="submit" disabled={isUploading || isRecording}>
            {isUploading ? "Uploading..." : "Ingest prototypes"}
          </button>
          <button
            type="button"
            className={styles.secondaryButton}
            onClick={resetForm}
            disabled={isUploading || isRecording}
          >
            Reset
          </button>
        </div>
      </form>

      {lastUpload?.prototypes?.length > 0 && (
        <div className={styles.uploadDetail}>
          <span className={styles.detailTitle}>Latest ingestion</span>
          <ul>
            {lastUpload.prototypes.slice(0, 5).map((proto) => (
              <li key={proto.id}>
                <span>{proto.id}</span>
                <span className={styles.detailCategory}>{proto.category}</span>
              </li>
            ))}
          </ul>
          {lastUpload.prototypes.length > 5 && (
            <small className={styles.detailHint}>
              {lastUpload.prototypes.length - 5} more stored on the backend.
            </small>
          )}
        </div>
      )}
    </section>
  );
};

export default PrototypeUpload;

