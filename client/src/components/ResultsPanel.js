import React, { useEffect, useRef, useState } from "react";
import * as THREE from "three";
import { OrbitControls } from "three/examples/jsm/controls/OrbitControls";
import { GLTFLoader } from "three/examples/jsm/loaders/GLTFLoader";
import styles from "./styles/ResultsPanel.module.css";

const formatConfidence = (value) => `${Math.round((value || 0) * 100)}%`;

const formatLatency = (value) => {
  if (!value && value !== 0) return "";
  return `${value.toFixed(1)} ms`;
};

const formatMetadataKey = (key) => {
  return key
    .replace(/_/g, " ")
    .replace(/\b\w/g, (char) => char.toUpperCase());
};

const formatMetadataValue = (key, value) => {
  if (value === null || value === undefined || value === "") {
    return "N/A";
  }

  if (typeof value === "boolean") {
    return value ? "Yes" : "No";
  }

  if (key.includes("time") && key.includes("minute")) return `${value} min`;
  if (key.includes("range") && key.includes("km")) return `${value} km`;
  if (key.includes("speed") && key.includes("ms")) return `${value} m/s`;
  if (key.includes("capacity") && key.includes("kg")) return `${value} kg`;

  return value;
};

const DroneModel3D = ({ modelName }) => {
  const mountRef = useRef(null);
  const sceneRef = useRef(null);
  const rendererRef = useRef(null);
  const cameraRef = useRef(null);
  const controlsRef = useRef(null);
  const modelRef = useRef(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    if (!mountRef.current || !modelName) return;

    // Scene setup
    const scene = new THREE.Scene();
    scene.background = new THREE.Color(0x000000);
    sceneRef.current = scene;

    // Camera setup
    const camera = new THREE.PerspectiveCamera(
      45,
      mountRef.current.clientWidth / mountRef.current.clientHeight,
      0.1,
      1000
    );
    camera.position.set(3, 2, 3);
    cameraRef.current = camera;

    // Renderer setup
    const renderer = new THREE.WebGLRenderer({ antialias: true });
    renderer.setSize(mountRef.current.clientWidth, mountRef.current.clientHeight);
    renderer.setPixelRatio(window.devicePixelRatio);
    renderer.shadowMap.enabled = true;
    renderer.shadowMap.type = THREE.PCFSoftShadowMap;
    mountRef.current.appendChild(renderer.domElement);
    rendererRef.current = renderer;

    // Lighting
    const ambientLight = new THREE.AmbientLight(0xffffff, 0.6);
    scene.add(ambientLight);

    const directionalLight = new THREE.DirectionalLight(0xffffff, 0.8);
    directionalLight.position.set(5, 10, 5);
    directionalLight.castShadow = true;
    directionalLight.shadow.mapSize.width = 2048;
    directionalLight.shadow.mapSize.height = 2048;
    scene.add(directionalLight);

    const fillLight = new THREE.DirectionalLight(0x4a90e2, 0.3);
    fillLight.position.set(-5, 5, -5);
    scene.add(fillLight);

    // Grid helper
    const gridHelper = new THREE.GridHelper(10, 10, 0x444444, 0x222222);
    scene.add(gridHelper);

    // Controls
    const controls = new OrbitControls(camera, renderer.domElement);
    controls.enableDamping = true;
    controls.dampingFactor = 0.05;
    controls.screenSpacePanning = false;
    controls.minDistance = 1;
    controls.maxDistance = 10;
    controls.maxPolarAngle = Math.PI / 2;
    controlsRef.current = controls;

    // Load model
    const loader = new GLTFLoader();
    const modelPath = `/models/${modelName}.glb`;
    
    loader.load(
      modelPath,
      (gltf) => {
        const model = gltf.scene;
        
        // Center and scale model
        const box = new THREE.Box3().setFromObject(model);
        const center = box.getCenter(new THREE.Vector3());
        const size = box.getSize(new THREE.Vector3());
        const maxDim = Math.max(size.x, size.y, size.z);
        const scale = 2 / maxDim;
        
        model.scale.multiplyScalar(scale);
        model.position.sub(center.multiplyScalar(scale));
        model.position.y = 0;
        
        // Enable shadows
        model.traverse((child) => {
          if (child.isMesh) {
            child.castShadow = true;
            child.receiveShadow = true;
          }
        });
        
        scene.add(model);
        modelRef.current = model;
        setLoading(false);
      },
      (progress) => {
        console.log(`Loading model: ${(progress.loaded / progress.total * 100).toFixed(0)}%`);
      },
      (error) => {
        console.error('Error loading model:', error);
        setError(`Failed to load ${modelName}.glb`);
        setLoading(false);
      }
    );

    // Animation loop
    let animationId;
    const animate = () => {
      animationId = requestAnimationFrame(animate);
      
      if (modelRef.current) {
        modelRef.current.rotation.y += 0.005;
      }
      
      controls.update();
      renderer.render(scene, camera);
    };
    animate();

    // Handle resize
    const handleResize = () => {
      if (!mountRef.current) return;
      
      const width = mountRef.current.clientWidth;
      const height = mountRef.current.clientHeight;
      
      camera.aspect = width / height;
      camera.updateProjectionMatrix();
      renderer.setSize(width, height);
    };
    window.addEventListener('resize', handleResize);

    // Cleanup
    return () => {
      window.removeEventListener('resize', handleResize);
      cancelAnimationFrame(animationId);
      
      if (controlsRef.current) {
        controlsRef.current.dispose();
      }
      
      if (rendererRef.current && mountRef.current) {
        mountRef.current.removeChild(rendererRef.current.domElement);
        rendererRef.current.dispose();
      }
      
      if (sceneRef.current) {
        sceneRef.current.traverse((object) => {
          if (object.geometry) object.geometry.dispose();
          if (object.material) {
            if (Array.isArray(object.material)) {
              object.material.forEach(material => material.dispose());
            } else {
              object.material.dispose();
            }
          }
        });
      }
    };
  }, [modelName]);

  if (!modelName) return null;

  return (
    <div style={{ position: 'relative', width: '100%', height: '400px', borderRadius: '8px', overflow: 'hidden' }}>
      <div ref={mountRef} style={{ width: '100%', height: '100%' }} />
      {loading && (
        <div style={{
          position: 'absolute',
          top: 0,
          left: 0,
          right: 0,
          bottom: 0,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          background: 'rgba(26, 26, 46, 0.9)',
          color: '#fff',
          fontSize: '14px'
        }}>
          Loading 3D model...
        </div>
      )}
      {error && (
        <div style={{
          position: 'absolute',
          top: 0,
          left: 0,
          right: 0,
          bottom: 0,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          background: 'rgba(26, 26, 46, 0.9)',
          color: '#ff6b6b',
          fontSize: '14px',
          padding: '20px',
          textAlign: 'center'
        }}>
          {error}
        </div>
      )}
      <div style={{
        position: 'absolute',
        bottom: '10px',
        left: '10px',
        background: 'rgba(0, 0, 0, 0.6)',
        color: '#fff',
        padding: '8px 12px',
        borderRadius: '4px',
        fontSize: '12px'
      }}>
        🖱️ Drag to rotate • Scroll to zoom
      </div>
    </div>
  );
};

const ResultsPanel = ({ classification, modelInfo, lastUpdated, isListening }) => {
  const predictions = classification?.predictions || [];
  const latencyLabel = formatLatency(classification?.latencyMs);
  const primaryType = classification?.primaryType;

  // Get only the top confidence prediction
  const topPrediction = predictions.length > 0 ? predictions[0] : null;

  // Extract model name from metadata.model field
  const getModelName = (prediction) => {
    if (!prediction) return null;
    
    // First, try to get the model from metadata
    if (prediction.metadata && prediction.metadata.model) {
      return prediction.metadata.model;
    }
    
    // Fallback to label if no model in metadata
    return prediction.label.replace(/\s+/g, '_');
  };

  let header = "Awaiting audio stream";
  let headerDetail = "";

  if (isListening) {
    header = "Listening for airborne activity";
  } else if (topPrediction) {
    header = "Drone signature analysis";
    if (classification?.isDrone && primaryType) {
      headerDetail = primaryType;
    }
  }

  return (
    <section className={styles.panel}>
      <header className={styles.panelHeader}>
        <div>
          <h3>{header}</h3>
          {headerDetail && <span className={styles.detectorDetail}>{headerDetail}</span>}
          {lastUpdated && (
            <span className={styles.timestamp}>
              Last analysed {lastUpdated.toLocaleTimeString()}
            </span>
          )}
        </div>

        {latencyLabel && (
          <span className={styles.latency}>Pipeline latency {latencyLabel}</span>
        )}
      </header>

      {!topPrediction ? (
        <div className={styles.placeholder}>
          <p>
            Record up to 20 seconds of audio to classify drone versus background noise in
            real time.
          </p>

          {modelInfo && modelInfo.labels && (
            <div className={styles.modelSummary}>
              <strong>Tracked classes</strong>
              <ul>
                {modelInfo.labels.map((label) => (
                  <li key={label.label}>
                    <span className={styles.labelName}>{label.label}</span>
                    <span className={styles.labelMeta}>{label.category}</span>
                    <span className={styles.labelCount}>
                      {label.prototypes} signatures
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      ) : (
        <div className={styles.resultsBody}>
          <article
            className={`${styles.prediction} ${
              topPrediction.category === "drone"
                ? styles.dronePrediction
                : styles.noisePrediction
            }`}
          >
            <div className={styles.predictionHeader}>
              <div>
                <h4 className={styles.predictionLabel}>{topPrediction.label}</h4>
                {topPrediction.type && (
                  <span className={styles.predictionType}>{topPrediction.type}</span>
                )}
                <span className={styles.predictionCategory}>
                  {topPrediction.category}
                </span>
                {topPrediction.description && (
                  <div className={styles.predictionDescription}>
                    {topPrediction.description}
                  </div>
                )}
              </div>

              <div className={styles.confidenceBox}>
                <span className={styles.confidenceValue}>
                  {formatConfidence(topPrediction.confidence)}
                </span>
                <span className={styles.confidenceCaption}>confidence</span>
              </div>
            </div>

            <div className={styles.progressBar}>
              <div
                className={styles.progressInner}
                style={{ width: formatConfidence(topPrediction.confidence) }}
              ></div>
            </div>

            <div className={styles.predictionMeta}>
              <span>Support {topPrediction.support}</span>
              <span>Avg distance {topPrediction.averageDistance.toFixed(3)}</span>
            </div>

            {/* 3D Model Viewer */}
            {topPrediction.category === "drone" && (
              <div style={{ marginTop: '1.5rem' }}>
                <DroneModel3D modelName={getModelName(topPrediction)} />
              </div>
            )}

            {topPrediction.threatAssessment && (
              <div className={styles.threatAssessmentBlock}>
                <span className={styles.threatAssessmentTitle}>
                  Threat Assessment
                </span>

                <div className={styles.threatGrid}>
                  {topPrediction.threatAssessment.threatLevel && (
                    <div className={styles.threatItem}>
                      <span className={styles.threatLabel}>Threat Level</span>
                      <span
                        className={`${styles.threatValue} ${
                          styles[
                            `threat${
                              topPrediction.threatAssessment.threatLevel
                                .charAt(0)
                                .toUpperCase() +
                              topPrediction.threatAssessment.threatLevel.slice(1)
                            }`
                          ]
                        }`}
                      >
                        {topPrediction.threatAssessment.threatLevel.toUpperCase()}
                      </span>
                    </div>
                  )}

                  {topPrediction.threatAssessment.riskCategory && (
                    <div className={styles.threatItem}>
                      <span className={styles.threatLabel}>Risk Category</span>
                      <span className={styles.threatValue}>
                        {topPrediction.threatAssessment.riskCategory.replace(/_/g, " ")}
                      </span>
                    </div>
                  )}

                  {topPrediction.threatAssessment.payloadCapacityKg > 0 && (
                    <div className={styles.threatItem}>
                      <span className={styles.threatLabel}>Payload Capacity</span>
                      <span className={styles.threatValue}>
                        {topPrediction.threatAssessment.payloadCapacityKg} kg
                      </span>
                    </div>
                  )}

                  {topPrediction.threatAssessment.maxRangeKm > 0 && (
                    <div className={styles.threatItem}>
                      <span className={styles.threatLabel}>Max Range</span>
                      <span className={styles.threatValue}>
                        {topPrediction.threatAssessment.maxRangeKm} km
                      </span>
                    </div>
                  )}

                  {topPrediction.threatAssessment.jammingSusceptible !== undefined && (
                    <div className={styles.threatItem}>
                      <span className={styles.threatLabel}>Jamming Susceptible</span>
                      <span className={styles.threatValue}>
                        {topPrediction.threatAssessment.jammingSusceptible ? "Yes" : "No"}
                      </span>
                    </div>
                  )}

                  {topPrediction.threatAssessment.countermeasureRecommendations && (
                    <div
                      className={styles.threatItem}
                      style={{ gridColumn: "1 / -1" }}
                    >
                      <span className={styles.threatLabel}>Countermeasures</span>
                      <span className={styles.threatValue}>
                        {topPrediction.threatAssessment.countermeasureRecommendations}
                      </span>
                    </div>
                  )}
                </div>
              </div>
            )}

            {topPrediction.metadata &&
              Object.keys(topPrediction.metadata).length > 0 && (
                <div className={styles.metadataBlock}>
                  <span className={styles.metadataTitle}>Signal Traits</span>

                  <dl className={styles.metadataList}>
                    {Object.entries(topPrediction.metadata)
                      .filter(([key]) => key !== "description")
                      .sort(([a], [b]) => {
                        const order = [
                          "model",
                          "manufacturer",
                          "type",
                          "operator_type",
                          "rotor_count",
                          "flight_time_minutes",
                          "max_range_km",
                          "max_speed_ms",
                          "payload_capacity_kg",
                          "jamming_susceptible",
                          "countermeasure_recommendations",
                        ];
                        const iA = order.indexOf(a);
                        const iB = order.indexOf(b);
                        if (iA === -1 && iB === -1) return a.localeCompare(b);
                        if (iA === -1) return 1;
                        if (iB === -1) return -1;
                        return iA - iB;
                      })
                      .map(([key, value]) => (
                        <div
                          key={`${topPrediction.label}-${key}`}
                          className={styles.metadataRow}
                        >
                          <dt>{formatMetadataKey(key)}</dt>
                          <dd>{formatMetadataValue(key, value)}</dd>
                        </div>
                      ))}
                  </dl>
                </div>
              )}
          </article>
        </div>
      )}
    </section>
  );
};

export default ResultsPanel;