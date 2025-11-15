import React from "react";
import styles from "./styles/SciencePage.module.css";

function SciencePage() {
  return (
    <div className={styles.sciencePage}>
      <div className={styles.container}>
        <header className={styles.header}>
          <h1>How It Works</h1>
          <p className={styles.subtitle}>
            Understanding how AALIS detects and classifies drone acoustic signatures through signal processing and machine learning
          </p>
        </header>

        <section className={styles.section}>
          <h2>Overview</h2>
          <p>
            AALIS (Acoustic Autonomous Lightweight Interception System) uses advanced signal processing 
            and machine learning to identify drone acoustic signatures in real-time. The system analyzes 
            audio recordings to extract distinctive features that characterize different types of drones, 
            then compares these features against a library of known prototypes.
          </p>
        </section>

        <section className={styles.section}>
          <h2>Why Acoustic Detection Works</h2>
          <p>
            Drone propellers produce unique acoustic signatures that distinguish them from other sound sources:
          </p>
          <div className={styles.featureGrid}>
            <div className={styles.featureCard}>
              <h3>Harmonic Content</h3>
              <p>
                Rotor blades create periodic frequency patterns as they rotate. The blade passing frequency 
                (RPM × blade_count / 60 Hz) produces strong harmonics that are characteristic of each drone type.
              </p>
            </div>
            <div className={styles.featureCard}>
              <h3>Motor RPM</h3>
              <p>
                Electric motors generate specific frequency components based on their rotation speed. Different 
                drone models operate at distinct RPM ranges, creating unique frequency signatures.
              </p>
            </div>
            <div className={styles.featureCard}>
              <h3>Resonance Patterns</h3>
              <p>
                Propeller and frame resonances add characteristic frequencies to the acoustic signature. These 
                structural vibrations create additional spectral features that help identify specific drone models.
              </p>
            </div>
            <div className={styles.featureCard}>
              <h3>Blade Passing Frequency</h3>
              <p>
                The fundamental frequency created by blades passing through the air follows the formula: 
                <code>BPF = RPM × blade_count / 60</code> Hz. This creates a distinctive "buzzing" sound 
                unique to each drone configuration.
              </p>
            </div>
          </div>
        </section>

        <section className={styles.section}>
          <h2>The Detection Pipeline</h2>
          <div className={styles.pipeline}>
            <div className={styles.pipelineStep}>
              <div className={styles.stepNumber}>1</div>
              <div className={styles.stepContent}>
                <h3>Audio Capture</h3>
                <p>
                  The browser captures up to 20 seconds of audio from a microphone or system audio source. 
                  The audio is encoded as WAV format and converted to mono channel at 44.1 kHz sample rate.
                </p>
              </div>
            </div>

            <div className={styles.pipelineStep}>
              <div className={styles.stepNumber}>2</div>
              <div className={styles.stepContent}>
                <h3>Audio Preprocessing</h3>
                <p>
                  The server applies several filters to enhance the signal:
                </p>
                <ul>
                  <li><strong>High-pass filter:</strong> Removes frequencies below 50 Hz (low-frequency noise)</li>
                  <li><strong>Band-pass filter:</strong> Focuses on 100-5000 Hz range where drone frequencies typically occur</li>
                  <li><strong>Automatic Gain Control (AGC):</strong> Normalizes audio levels to ensure consistent analysis</li>
                  <li><strong>Noise reduction:</strong> Basic spectral subtraction to reduce background noise</li>
                </ul>
              </div>
            </div>

            <div className={styles.pipelineStep}>
              <div className={styles.stepNumber}>3</div>
              <div className={styles.stepContent}>
                <h3>SNR Estimation</h3>
                <p>
                  Signal-to-noise ratio is calculated from the first 10% of the audio. This value is used 
                  to adaptively adjust the confidence threshold—higher SNR allows for more reliable detection 
                  with lower thresholds.
                </p>
              </div>
            </div>

            <div className={styles.pipelineStep}>
              <div className={styles.stepNumber}>4</div>
              <div className={styles.stepContent}>
                <h3>Feature Extraction</h3>
                <p>
                  The system extracts 19 spectral and temporal features that capture the acoustic signature:
                </p>
                <div className={styles.featureList}>
                  <div>
                    <h4>Temporal Features</h4>
                    <ul>
                      <li>Energy (RMS): Overall signal strength</li>
                      <li>Zero Crossing Rate: Frequency of sign changes</li>
                      <li>Variance: Signal variability</li>
                      <li>Temporal Centroid: Location of energy within the window</li>
                      <li>Onset Rate: Frequency of amplitude onsets</li>
                      <li>Amplitude Modulation Depth: Variation of amplitude envelope</li>
                    </ul>
                  </div>
                  <div>
                    <h4>Spectral Features</h4>
                    <ul>
                      <li>Spectral Centroid: "Brightness" of sound</li>
                      <li>Spectral Bandwidth: Spread of frequencies</li>
                      <li>Spectral Rolloff: Frequency containing 85% of energy</li>
                      <li>Spectral Flatness: Measure of noisiness</li>
                      <li>Spectral Crest Factor: Peak-to-average ratio</li>
                      <li>Spectral Entropy: Randomness in frequency distribution</li>
                      <li>Dominant Frequency: Frequency bin with maximum magnitude</li>
                      <li>Spectral Skewness: Asymmetry of frequency distribution</li>
                      <li>Spectral Kurtosis: Peakedness of frequency distribution</li>
                      <li>Peak Prominence: Contrast between peaks and average</li>
                    </ul>
                  </div>
                  <div>
                    <h4>Harmonic Features</h4>
                    <ul>
                      <li>Harmonic Ratio: Ratio of harmonic energy to total energy</li>
                      <li>Harmonic Count: Number of significant harmonic peaks</li>
                      <li>Harmonic Strength: Average magnitude of harmonic components</li>
                    </ul>
                  </div>
                </div>
              </div>
            </div>

            <div className={styles.pipelineStep}>
              <div className={styles.stepNumber}>5</div>
              <div className={styles.stepContent}>
                <h3>Feature Normalization</h3>
                <p>
                  The extracted feature vector is normalized to unit length (L2 normalization). This ensures 
                  that distance-based comparisons are not biased by the magnitude of individual features, 
                  allowing the classifier to focus on the relative patterns rather than absolute values.
                </p>
              </div>
            </div>

            <div className={styles.pipelineStep}>
              <div className={styles.stepNumber}>6</div>
              <div className={styles.stepContent}>
                <h3>Classification</h3>
                <p>
                  The normalized feature vector is compared against a library of known drone prototypes using 
                  a k-nearest neighbors (KNN) algorithm:
                </p>
                <ul>
                  <li><strong>Distance Calculation:</strong> Weighted cosine similarity is computed between 
                      the input features and each prototype in the library</li>
                  <li><strong>K Selection:</strong> The k nearest prototypes are selected (default k=5, 
                      adaptively adjusted if fewer prototypes exist)</li>
                  <li><strong>Weight Aggregation:</strong> For each label, weights are aggregated from 
                      matching prototypes using the formula: <code>weight = 1 / (distance + ε)</code></li>
                  <li><strong>Confidence Calculation:</strong> Confidence for each label is computed as: 
                      <code>confidence = sum(weights for label) / total_weight</code></li>
                </ul>
              </div>
            </div>

            <div className={styles.pipelineStep}>
              <div className={styles.stepNumber}>7</div>
              <div className={styles.stepContent}>
                <h3>Adaptive Thresholding</h3>
                <p>
                  The system uses adaptive thresholding based on signal-to-noise ratio:
                </p>
                <ul>
                  <li>Higher SNR values allow for lower confidence thresholds (more sensitive detection)</li>
                  <li>Lower SNR values require higher confidence thresholds (reduces false positives)</li>
                  <li>Default threshold is 0.55, adjusted dynamically based on SNR</li>
                </ul>
              </div>
            </div>

            <div className={styles.pipelineStep}>
              <div className={styles.stepNumber}>8</div>
              <div className={styles.stepContent}>
                <h3>Results</h3>
                <p>
                  The system returns ranked predictions with:
                </p>
                <ul>
                  <li>Label identification (drone type or "noise")</li>
                  <li>Confidence scores (0.0 to 1.0)</li>
                  <li>Average distance to matching prototypes</li>
                  <li>Support count (number of matching prototypes)</li>
                  <li>SNR information</li>
                  <li>Threat assessment (for defense applications)</li>
                </ul>
              </div>
            </div>
          </div>
        </section>

        <section className={styles.section}>
          <h2>Technical Architecture</h2>
          <div className={styles.architecture}>
            <div className={styles.archBlock}>
              <h3>Frontend (React)</h3>
              <ul>
                <li>Audio capture via MediaRecorder API</li>
                <li>WAV encoding using extendable-media-recorder</li>
                <li>FFmpeg integration for audio conversion</li>
                <li>WebSocket and HTTP POST for server communication</li>
                <li>Real-time results visualization</li>
              </ul>
            </div>
            <div className={styles.archBlock}>
              <h3>Backend (Go)</h3>
              <ul>
                <li>Audio preprocessing pipeline</li>
                <li>FFT-based spectral analysis</li>
                <li>Feature extraction algorithms</li>
                <li>KNN classifier implementation</li>
                <li>Prototype library management</li>
              </ul>
            </div>
            <div className={styles.archBlock}>
              <h3>Signal Processing</h3>
              <ul>
                <li>Fast Fourier Transform (FFT) for frequency analysis</li>
                <li>Hann windowing to reduce spectral leakage</li>
                <li>Digital filtering (high-pass, band-pass)</li>
                <li>Harmonic analysis for fundamental frequency detection</li>
                <li>SNR estimation algorithms</li>
              </ul>
            </div>
          </div>
        </section>

        <section className={styles.section}>
          <h2>Performance Characteristics</h2>
          <div className={styles.performanceGrid}>
            <div className={styles.perfCard}>
              <h3>Latency</h3>
              <p className={styles.perfValue}>40-100ms</p>
              <p>Typical classification time from audio submission to results</p>
            </div>
            <div className={styles.perfCard}>
              <h3>Accuracy</h3>
              <p className={styles.perfValue}>High</p>
              <p>Depends on prototype library quality and diversity</p>
            </div>
            <div className={styles.perfCard}>
              <h3>Scalability</h3>
              <p className={styles.perfValue}>O(n)</p>
              <p>Linear time complexity where n = number of prototypes</p>
            </div>
            <div className={styles.perfCard}>
              <h3>Memory</h3>
              <p className={styles.perfValue}>Low</p>
              <p>All prototypes loaded in memory for fast access</p>
            </div>
          </div>
        </section>

        <section className={styles.section}>
          <h2>Why These Features Work</h2>
          <p>
            The combination of spectral, temporal, and harmonic features creates a comprehensive acoustic 
            fingerprint for each drone type:
          </p>
          <ul className={styles.bulletList}>
            <li><strong>Spectral features</strong> identify frequency content and harmonics that are unique 
                to each drone's motor and propeller configuration</li>
            <li><strong>Temporal features</strong> capture amplitude and variability patterns that distinguish 
                steady drone operation from transient noise</li>
            <li><strong>Harmonic features</strong> are particularly critical for drone detection, as they capture 
                the periodic patterns created by rotating propellers</li>
            <li><strong>Combined together</strong>, these 19 features create a unique "fingerprint" that allows 
                the system to distinguish between different drone types and reject non-drone sounds</li>
          </ul>
        </section>

        <section className={styles.section}>
          <h2>Learning and Adaptation</h2>
          <p>
            AALIS supports dynamic prototype addition, allowing the system to learn new drone types without 
            retraining:
          </p>
          <ul className={styles.bulletList}>
            <li>New audio samples can be uploaded via the web interface</li>
            <li>Each sample is processed through the same feature extraction pipeline</li>
            <li>Prototypes are automatically normalized and added to the library</li>
            <li>The system immediately begins using new prototypes for classification</li>
            <li>This enables continuous improvement and adaptation to new threats</li>
          </ul>
        </section>
      </div>
    </div>
  );
}

export default SciencePage;

