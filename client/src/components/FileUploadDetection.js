import React, { useState, useRef } from "react";
import { toast } from "react-toastify";
import { FaUpload, FaFileAudio } from "react-icons/fa";
import styles from "./styles/FileUploadDetection.module.css";

const FileUploadDetection = ({ backendUrl, onClassificationComplete }) => {
  const [selectedFile, setSelectedFile] = useState(null);
  const [isProcessing, setIsProcessing] = useState(false);
  const [dragActive, setDragActive] = useState(false);
  const fileInputRef = useRef(null);

  const handleFileSelect = (event) => {
    const file = event.target.files?.[0];
    if (file) {
      validateAndSetFile(file);
    }
  };

  const validateAndSetFile = (file) => {
    if (!file.type.startsWith('audio/')) {
      toast.error("Please select an audio file");
      return;
    }
    
    const maxSize = 50 * 1024 * 1024; // 50MB
    if (file.size > maxSize) {
      toast.error("File size must be less than 50MB");
      return;
    }
    
    setSelectedFile(file);
    toast.success(`Selected: ${file.name}`);
  };

  const handleDrag = (e) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.type === "dragenter" || e.type === "dragover") {
      setDragActive(true);
    } else if (e.type === "dragleave") {
      setDragActive(false);
    }
  };

  const handleDrop = (e) => {
    e.preventDefault();
    e.stopPropagation();
    setDragActive(false);
    
    if (e.dataTransfer.files && e.dataTransfer.files[0]) {
      validateAndSetFile(e.dataTransfer.files[0]);
    }
  };

  const handleUploadClick = () => {
    fileInputRef.current?.click();
  };

  const processAndClassifyAudio = async () => {
    if (!selectedFile) {
      toast.error("Please select an audio file first");
      return;
    }

    setIsProcessing(true);

    try {
      // Read the audio file
      const arrayBuffer = await selectedFile.arrayBuffer();
      
      // Decode audio to get proper format
      const audioContext = new (window.AudioContext || window.webkitAudioContext)();
      const audioBuffer = await audioContext.decodeAudioData(arrayBuffer);
      
      // Convert to mono PCM 16-bit WAV at 44100Hz
      const sampleRate = 44100;
      const channels = 1;
      
      // Resample if needed
      let monoSamples;
      if (audioBuffer.sampleRate !== sampleRate) {
        const offlineContext = new OfflineAudioContext(
          channels,
          Math.ceil(audioBuffer.duration * sampleRate),
          sampleRate
        );
        const source = offlineContext.createBufferSource();
        source.buffer = audioBuffer;
        source.connect(offlineContext.destination);
        source.start(0);
        const resampled = await offlineContext.startRendering();
        monoSamples = resampled.getChannelData(0);
      } else {
        monoSamples = audioBuffer.getChannelData(0);
      }
      
      // Create WAV file
      const wavBuffer = createWavBuffer(monoSamples, sampleRate, channels);
      const wavBlob = new Blob([wavBuffer], { type: "audio/wav" });
      
      // Extract audio visualization data
      const audioData = await extractAudioData(wavBlob);
      
      // Convert to base64
      const reader = new FileReader();
      reader.readAsArrayBuffer(wavBlob);
      
      reader.onload = async () => {
        const arrayBuffer = reader.result;
        const bytes = new Uint8Array(arrayBuffer);
        let binary = "";
        for (let i = 0; i < bytes.byteLength; i++) {
          binary += String.fromCharCode(bytes[i]);
        }
        
        const dataView = new DataView(arrayBuffer);
        
        // Get location if available
        let latitude = null;
        let longitude = null;
        
        if (navigator.geolocation) {
          try {
            const position = await new Promise((resolve, reject) => {
              navigator.geolocation.getCurrentPosition(resolve, reject, {
                timeout: 5000,
                maximumAge: 60000
              });
            });
            latitude = position.coords.latitude;
            longitude = position.coords.longitude;
          } catch (error) {
            console.warn("Could not get location:", error.message);
          }
        }
        
        const recordData = {
          audio: btoa(binary),
          channels: dataView.getUint16(22, true),
          sampleRate: dataView.getUint32(24, true),
          sampleSize: dataView.getUint16(34, true),
          duration: monoSamples.length / sampleRate,
          latitude: latitude,
          longitude: longitude,
        };
        
        console.log("Sending file classification request", {
          fileName: selectedFile.name,
          sampleRate: recordData.sampleRate,
          channels: recordData.channels,
          duration: recordData.duration,
          audioLength: recordData.audio?.length || 0,
        });
        
        // Send to backend (normalize URL to avoid double slashes)
        const normalizedUrl = backendUrl.replace(/\/$/, ''); // Remove trailing slash
        const response = await fetch(`${normalizedUrl}/api/audio/classify`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify(recordData),
        });
        
        if (!response.ok) {
          const error = await response.json().catch(() => ({}));
          throw new Error(error.message || `HTTP error! status: ${response.status}`);
        }
        
        const result = await response.json();
        console.log("Received classification result for file:", result);
        
        toast.success(`File classified: ${selectedFile.name}`);
        
        // Call the callback with results and audio data
        if (onClassificationComplete) {
          onClassificationComplete(result, audioData);
        }
        
        setIsProcessing(false);
        setSelectedFile(null);
        if (fileInputRef.current) {
          fileInputRef.current.value = "";
        }
      };
      
      reader.onerror = () => {
        throw new Error("Failed to read audio file");
      };
      
    } catch (error) {
      console.error("Error processing audio file:", error);
      toast.error(error.message || "Failed to process audio file");
      setIsProcessing(false);
    }
  };

  const createWavBuffer = (samples, sampleRate, channels) => {
    const dataLength = samples.length * 2; // 16-bit = 2 bytes per sample
    const buffer = new ArrayBuffer(44 + dataLength);
    const view = new DataView(buffer);
    
    // WAV header
    writeString(view, 0, 'RIFF');
    view.setUint32(4, 36 + dataLength, true);
    writeString(view, 8, 'WAVE');
    writeString(view, 12, 'fmt ');
    view.setUint32(16, 16, true); // PCM format
    view.setUint16(20, 1, true); // PCM = 1
    view.setUint16(22, channels, true);
    view.setUint32(24, sampleRate, true);
    view.setUint32(28, sampleRate * channels * 2, true); // byte rate
    view.setUint16(32, channels * 2, true); // block align
    view.setUint16(34, 16, true); // bits per sample
    writeString(view, 36, 'data');
    view.setUint32(40, dataLength, true);
    
    // Write PCM samples
    let offset = 44;
    for (let i = 0; i < samples.length; i++) {
      const s = Math.max(-1, Math.min(1, samples[i]));
      view.setInt16(offset, s < 0 ? s * 0x8000 : s * 0x7FFF, true);
      offset += 2;
    }
    
    return buffer;
  };

  const writeString = (view, offset, string) => {
    for (let i = 0; i < string.length; i++) {
      view.setUint8(offset + i, string.charCodeAt(i));
    }
  };

  const extractAudioData = async (audioBlob) => {
    try {
      const audioContext = new (window.AudioContext || window.webkitAudioContext)();
      const arrayBuffer = await audioBlob.arrayBuffer();
      const audioBuffer = await audioContext.decodeAudioData(arrayBuffer);
      
      const channelData = audioBuffer.getChannelData(0);
      
      // Sample the waveform
      const samples = 200;
      const blockSize = Math.floor(channelData.length / samples);
      const waveformData = [];
      
      for (let i = 0; i < samples; i++) {
        const start = blockSize * i;
        let sum = 0;
        
        for (let j = 0; j < blockSize; j++) {
          const val = Math.abs(channelData[start + j]);
          sum += val;
        }
        
        const rms = Math.sqrt(sum / blockSize);
        waveformData.push(rms);
      }
      
      // Calculate frequency data
      const analyser = audioContext.createAnalyser();
      analyser.fftSize = 2048;
      const bufferLength = analyser.frequencyBinCount;
      const frequencyData = new Uint8Array(bufferLength);
      
      const source = audioContext.createBufferSource();
      source.buffer = audioBuffer;
      source.connect(analyser);
      
      analyser.getByteFrequencyData(frequencyData);
      
      const freqSamples = 40;
      const freqBlockSize = Math.floor(bufferLength / freqSamples);
      const sampledFreqData = [];
      
      for (let i = 0; i < freqSamples; i++) {
        let sum = 0;
        for (let j = 0; j < freqBlockSize; j++) {
          sum += frequencyData[i * freqBlockSize + j];
        }
        sampledFreqData.push(sum / freqBlockSize / 255);
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

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <h3 className={styles.title}>File Upload Detection</h3>
        <p className={styles.description}>
          Upload an audio file to analyze for drone signatures
        </p>
      </div>
      
      <div
        className={`${styles.dropzone} ${dragActive ? styles.dragActive : ''} ${selectedFile ? styles.hasFile : ''}`}
        onDragEnter={handleDrag}
        onDragLeave={handleDrag}
        onDragOver={handleDrag}
        onDrop={handleDrop}
        onClick={handleUploadClick}
      >
        <input
          ref={fileInputRef}
          type="file"
          accept="audio/*"
          onChange={handleFileSelect}
          style={{ display: 'none' }}
          disabled={isProcessing}
        />
        
        {selectedFile ? (
          <div className={styles.fileInfo}>
            <FaFileAudio className={styles.fileIcon} />
            <div className={styles.fileDetails}>
              <div className={styles.fileName}>{selectedFile.name}</div>
              <div className={styles.fileSize}>
                {(selectedFile.size / 1024 / 1024).toFixed(2)} MB
              </div>
            </div>
          </div>
        ) : (
          <div className={styles.uploadPrompt}>
            <FaUpload className={styles.uploadIcon} />
            <p className={styles.uploadText}>
              Drop an audio file here or click to browse
            </p>
            <p className={styles.uploadHint}>
              Supports WAV, MP3, and other audio formats (max 50MB)
            </p>
          </div>
        )}
      </div>
      
      {selectedFile && (
        <button
          className={styles.classifyButton}
          onClick={processAndClassifyAudio}
          disabled={isProcessing}
        >
          {isProcessing ? (
            <>
              <span className={styles.spinner}></span>
              Processing...
            </>
          ) : (
            <>
              <FaFileAudio style={{ marginRight: '8px' }} />
              Classify Audio File
            </>
          )}
        </button>
      )}
    </div>
  );
};

export default FileUploadDetection;

